// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

const hashTestWidgetRef = "$ref: '#/components/schemas/Widget'"

// buildHashTestSchemas indexes one document that holds a Widget component and every snippet as a component of its
// own, then builds each snippet. Sharing one indexed document lets snippets $ref Widget, and gives every schema its
// own position, so each one gets its own quick hash cache entry.
func buildHashTestSchemas(t *testing.T, snippets ...string) []*Schema {
	t.Helper()

	var spec strings.Builder
	spec.WriteString("openapi: 3.1.0\ncomponents:\n  schemas:\n" +
		"    Widget:\n      type: object\n      properties:\n        id:\n          type: string\n")
	for i, snippet := range snippets {
		fmt.Fprintf(&spec, "    Schema%d:\n", i)
		for _, line := range strings.Split(snippet, "\n") {
			spec.WriteString("      " + line + "\n")
		}
	}

	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(spec.String()), &root))
	idx := index.NewSpecIndexWithConfig(&root, index.CreateOpenAPIIndexConfig())

	_, _, components := utils.FindKeyNodeFullTop("components", root.Content[0].Content)
	_, _, schemas := utils.FindKeyNodeFullTop("schemas", components.Content)

	built := make([]*Schema, len(snippets))
	for i := range snippets {
		// Widget is the first key and value, so snippet i starts at index 2+2i.
		proxy := new(SchemaProxy)
		require.NoError(t, proxy.Build(context.Background(), schemas.Content[2+2*i], schemas.Content[3+2*i], idx))
		built[i] = proxy.Schema()
		require.NotNil(t, built[i], "snippet %d did not build: %v", i, proxy.GetBuildError())
	}
	return built
}

// assertSchemaHashesDiffer checks that no two snippets share a Hash() or a QuickHash().
func assertSchemaHashesDiffer(t *testing.T, snippets ...string) {
	t.Helper()
	schemas := buildHashTestSchemas(t, snippets...)
	for i := range schemas {
		for j := i + 1; j < len(schemas); j++ {
			assert.NotEqual(t, schemas[i].Hash(), schemas[j].Hash(),
				"Hash() collides for:\n%s\n---\n%s", snippets[i], snippets[j])
			assert.NotEqual(t, schemas[i].QuickHash(), schemas[j].QuickHash(),
				"QuickHash() collides for:\n%s\n---\n%s", snippets[i], snippets[j])
		}
	}
}

// assertSchemaHashesMatch checks that left and right share a Hash() and a QuickHash(). They are built in separate
// documents, as the same schema would be when it is parsed twice.
func assertSchemaHashesMatch(t *testing.T, left, right string) {
	t.Helper()
	l := buildHashTestSchemas(t, left)[0]
	r := buildHashTestSchemas(t, right)[0]
	assert.Equal(t, l.Hash(), r.Hash(), "Hash() differs for:\n%s\n---\n%s", left, right)
	assert.Equal(t, l.QuickHash(), r.QuickHash(), "QuickHash() differs for:\n%s\n---\n%s", left, right)
}

// Each case lists schemas that differ in which keyword carries a value. Hashing values without their keywords made
// every schema in a case hash the same, so a cache keyed on the hash handed one schema's compiled validator to
// another, and what-changed reported no change between them.
func TestSchemaHash_KeywordsDoNotCollide(t *testing.T) {
	widgetBelow := func(keyword string) string {
		return keyword + ":\n  " + hashTestWidgetRef
	}
	widgetInList := func(keyword string) string {
		return keyword + ":\n  - " + hashTestWidgetRef
	}
	widgetKeyed := func(keyword string) string {
		return keyword + ":\n  a:\n    " + hashTestWidgetRef
	}

	tests := []struct {
		name     string
		snippets []string
	}{
		{
			name: "numeric bounds",
			snippets: []string{
				"type: integer\nminimum: 5",
				"type: integer\nmaximum: 5",
				"type: integer\nexclusiveMinimum: 5",
				"type: integer\nexclusiveMaximum: 5",
				"type: integer\nmultipleOf: 5",
			},
		},
		{
			name: "exclusive bound flags",
			snippets: []string{
				"type: integer\nminimum: 5\nexclusiveMinimum: true",
				"type: integer\nminimum: 5\nexclusiveMaximum: true",
			},
		},
		{
			name: "length and count limits",
			snippets: []string{
				"minLength: 3",
				"maxLength: 3",
				"minItems: 3",
				"maxItems: 3",
				"minProperties: 3",
				"maxProperties: 3",
				"minContains: 3",
				"maxContains: 3",
			},
		},
		{
			name: "string keywords",
			snippets: []string{
				"title: same",
				"description: same",
				"pattern: same",
				"format: same",
				"contentEncoding: same",
				"contentMediaType: same",
				"$comment: same",
				"$id: same",
				"$anchor: same",
				"$dynamicAnchor: same",
				"$dynamicRef: same",
				"$schema: same",
				"type: same",
				"required:\n  - same",
				"enum:\n  - same",
			},
		},
		{
			name: "boolean keywords set to true",
			snippets: []string{
				"readOnly: true",
				"writeOnly: true",
				"deprecated: true",
				"nullable: true",
				"uniqueItems: true",
				"exclusiveMinimum: true",
				"exclusiveMaximum: true",
				"items: true",
				"additionalProperties: true",
				"unevaluatedProperties: true",
			},
		},
		{
			name: "boolean keywords set to false",
			snippets: []string{
				"readOnly: false",
				"writeOnly: false",
				"deprecated: false",
				"nullable: false",
				"uniqueItems: false",
				"exclusiveMinimum: false",
				"exclusiveMaximum: false",
				"items: false",
				"additionalProperties: false",
				"unevaluatedProperties: false",
			},
		},
		{
			name: "composition keywords over a reference and an inline schema",
			snippets: []string{
				"oneOf:\n  - " + hashTestWidgetRef + "\n  - type: object",
				"allOf:\n  - " + hashTestWidgetRef + "\n  - type: object",
				"anyOf:\n  - " + hashTestWidgetRef + "\n  - type: object",
				"prefixItems:\n  - " + hashTestWidgetRef + "\n  - type: object",
			},
		},
		{
			name: "composition keywords over inline schemas",
			snippets: []string{
				"oneOf:\n  - type: string\n  - type: object",
				"allOf:\n  - type: string\n  - type: object",
				"anyOf:\n  - type: string\n  - type: object",
				"prefixItems:\n  - type: string\n  - type: object",
			},
		},
		{
			name: "one subschema under different keywords",
			snippets: []string{
				widgetBelow("not"),
				widgetBelow("items"),
				widgetBelow("contains"),
				widgetBelow("if"),
				widgetBelow("then"),
				widgetBelow("else"),
				widgetBelow("propertyNames"),
				widgetBelow("unevaluatedItems"),
				widgetBelow("contentSchema"),
				widgetBelow("additionalProperties"),
				widgetBelow("unevaluatedProperties"),
				widgetInList("oneOf"),
				widgetInList("allOf"),
				widgetInList("anyOf"),
				widgetInList("prefixItems"),
			},
		},
		{
			name: "keyed subschemas under different keywords",
			snippets: []string{
				widgetKeyed("properties"),
				widgetKeyed("patternProperties"),
				widgetKeyed("dependentSchemas"),
				widgetKeyed("$defs"),
			},
		},
		{
			name: "instance values under different keywords",
			snippets: []string{
				"default: 5",
				"const: 5",
				"example: 5",
				"examples:\n  - 5",
				"enum:\n  - 5",
			},
		},
		{
			name: "enum values of different types",
			snippets: []string{
				"enum:\n  - 1",
				"enum:\n  - '1'",
				"enum:\n  - true",
				"enum:\n  - 'true'",
				"enum:\n  - null",
				"enum:\n  - 'null'",
			},
		},
		{
			name: "vocabulary and dependentRequired",
			snippets: []string{
				"$vocabulary:\n  https://example.com/vocab: true",
				"dependentRequired:\n  https://example.com/vocab:\n    - 'true'",
			},
		},
		{
			name: "prefixItems order",
			snippets: []string{
				"prefixItems:\n  - type: string\n  - type: integer",
				"prefixItems:\n  - type: integer\n  - type: string",
			},
		},
		{
			name: "separators inside values",
			snippets: []string{
				"title: 'a|b'",
				"title: a\ndescription: b",
				"title: 'a|description:b'",
				"required:\n  - 'a|b'",
				"required:\n  - a\n  - b",
				"type:\n  - 'a|b'",
				"type:\n  - a\n  - b",
			},
		},
		{
			name: "xml fields",
			snippets: []string{
				"xml:\n  name: same",
				"xml:\n  namespace: same",
				"xml:\n  prefix: same",
				"xml:\n  nodeType: same",
				"xml:\n  attribute: true",
				"xml:\n  wrapped: true",
			},
		},
		{
			name: "externalDocs fields",
			snippets: []string{
				"externalDocs:\n  url: same",
				"externalDocs:\n  description: same",
			},
		},
		{
			name: "discriminator fields",
			snippets: []string{
				"discriminator:\n  propertyName: same",
				"discriminator:\n  defaultMapping: same",
				"discriminator:\n  propertyName: kind\n  mapping:\n    cat: '#/components/schemas/Widget'",
				"discriminator:\n  propertyName: kind\n  mapping:\n    dog: '#/components/schemas/Widget'",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertSchemaHashesDiffer(t, tc.snippets...)
		})
	}
}

// Labelling keywords must not change which schemas count as equal: the same schema still hashes the same however
// it is reached or ordered, and parts that mean the same in any order still ignore their order.
func TestSchemaHash_EquivalentSchemasStillMatch(t *testing.T) {
	everyKeyword := `$schema: https://json-schema.org/draft/2020-12/schema
$id: https://example.com/widget
$anchor: widget
$comment: every keyword
title: Widget
description: A widget
type:
  - object
  - 'null'
required:
  - id
  - name
enum:
  - a
  - 1
multipleOf: 2
minimum: 1
maximum: 10
exclusiveMinimum: 0
exclusiveMaximum: 11
minLength: 1
maxLength: 5
pattern: ^w
format: widget
minItems: 1
maxItems: 3
uniqueItems: true
minProperties: 1
maxProperties: 9
minContains: 1
maxContains: 2
readOnly: true
writeOnly: false
deprecated: true
nullable: true
default: a
const: a
example: a
examples:
  - a
  - b
contentEncoding: base64
contentMediaType: text/plain
contentSchema:
  type: string
properties:
  id:
    type: string
  name:
    ` + hashTestWidgetRef + `
patternProperties:
  ^x-:
    type: string
dependentSchemas:
  id:
    required:
      - name
dependentRequired:
  id:
    - name
$defs:
  thing:
    type: string
$vocabulary:
  https://example.com/vocab: true
additionalProperties: false
unevaluatedProperties: false
propertyNames:
  pattern: ^[a-z]+$
items:
  type: string
prefixItems:
  - type: string
  - type: integer
contains:
  type: string
unevaluatedItems:
  type: string
oneOf:
  - ` + hashTestWidgetRef + `
  - type: object
allOf:
  - type: object
anyOf:
  - type: object
not:
  type: string
if:
  type: object
then:
  required:
    - id
else:
  required:
    - name
xml:
  name: widget
externalDocs:
  url: https://example.com
discriminator:
  propertyName: kind
  mapping:
    widget: '#/components/schemas/Widget'
x-widget: true`

	tests := []struct {
		name        string
		left, right string
	}{
		{
			name:  "same schema parsed twice",
			left:  everyKeyword,
			right: everyKeyword,
		},
		{
			name:  "same schema reached through a reference",
			left:  hashTestWidgetRef,
			right: "type: object\nproperties:\n  id:\n    type: string",
		},
		{
			name:  "keyword order",
			left:  "title: Widget\ntype: integer\nminimum: 1\nmaximum: 5",
			right: "maximum: 5\nminimum: 1\ntype: integer\ntitle: Widget",
		},
		{
			name:  "property order",
			left:  "properties:\n  a:\n    type: string\n  b:\n    type: integer",
			right: "properties:\n  b:\n    type: integer\n  a:\n    type: string",
		},
		{
			name:  "required order",
			left:  "required:\n  - a\n  - b",
			right: "required:\n  - b\n  - a",
		},
		{
			name:  "enum order",
			left:  "enum:\n  - a\n  - 1\n  - true",
			right: "enum:\n  - true\n  - a\n  - 1",
		},
		{
			name:  "type array order",
			left:  "type:\n  - string\n  - 'null'",
			right: "type:\n  - 'null'\n  - string",
		},
		{
			name:  "single type and a one entry type array",
			left:  "type: string",
			right: "type:\n  - string",
		},
		{
			name:  "oneOf member order",
			left:  "oneOf:\n  - " + hashTestWidgetRef + "\n  - type: object",
			right: "oneOf:\n  - type: object\n  - " + hashTestWidgetRef,
		},
		{
			name:  "allOf member order",
			left:  "allOf:\n  - type: string\n  - type: object",
			right: "allOf:\n  - type: object\n  - type: string",
		},
		{
			name:  "anyOf member order",
			left:  "anyOf:\n  - type: string\n  - type: object",
			right: "anyOf:\n  - type: object\n  - type: string",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertSchemaHashesMatch(t, tc.left, tc.right)
		})
	}
}

func TestSchemaDynamicValue_HashSeparatesSides(t *testing.T) {
	a := &SchemaDynamicValue[string, string]{N: 0, A: "same"}
	b := &SchemaDynamicValue[string, string]{N: 1, B: "same"}
	assert.NotEqual(t, a.Hash(), b.Hash())
	assert.Equal(t, a.Hash(), (&SchemaDynamicValue[string, string]{N: 0, A: "same"}).Hash())
}
