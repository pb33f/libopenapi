// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

// usageSpec builds a document whose operation and components are given as YAML
// fragments indented for their place in the document.
func usageSpec(operation, schemas string) string {
	return `openapi: 3.1.0
info: {title: Direction, version: 1.0.0}
paths:
  /widgets:
    post:
` + operation + `
components:
  schemas:
` + schemas
}

const (
	usageRequestOnly = `      requestBody:
        content:
          application/json:
            schema: {$ref: '#/components/schemas/Widget'}
      responses:
        '204': {description: ok}`
	usageResponseOnly = `      responses:
        '200':
          description: ok
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Widget'}`
	usageShared = `      requestBody:
        content:
          application/json:
            schema: {$ref: '#/components/schemas/Widget'}
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Widget'}`
	usageCallback = `      responses:
        '204': {description: ok}
      callbacks:
        done:
          '{$request.body#/url}':
            post:
              requestBody:
                content:
                  application/json:
                    schema: {$ref: '#/components/schemas/Widget'}
              responses:
                '204': {description: ok}`
	usageUnused = `      responses:
        '204': {description: ok}`
)

func usageWidget(required string) string {
	return `    Widget:
      type: object
      required: [` + required + `]
      properties:
        name: {type: string}
        id: {type: string}`
}

func compareUsageDocuments(t *testing.T, left, right string) *DocumentChanges {
	t.Helper()
	return CompareDocuments(buildUsageDocument(t, left), buildUsageDocument(t, right))
}

func buildUsageDocument(t *testing.T, spec string) *v3.Document {
	t.Helper()
	info, err := datamodel.ExtractSpecInfo([]byte(spec))
	require.NoError(t, err)
	doc, err := v3.CreateDocumentFromConfig(info, datamodel.NewDocumentConfiguration())
	require.NoError(t, err)
	return doc
}

func TestCompareDocuments_SchemaRequiredDirectionInlineAndNested(t *testing.T) {
	const operation = `      parameters:
        - name: filter
          in: query
          schema: {$ref: '#/components/schemas/Filter'}
      requestBody:
        $ref: '#/components/requestBodies/WidgetInput'
      responses:
        '200':
          $ref: '#/components/responses/WidgetOutput'
        '201':
          description: ok
          headers:
            X-Page: {schema: {$ref: '#/components/schemas/Page'}}
          content:
            application/json:
              schema:
                type: object
                required: [%s]
                properties:
                  name: {type: string}
                  id: {type: string}`
	const schemas = `    Filter: {type: object, required: [%s], properties: {name: {type: string}, id: {type: string}}}
    Page: {type: object, required: [%s], properties: {name: {type: string}, id: {type: string}}}
    Input: {type: object, properties: {part: {$ref: '#/components/schemas/Part'}}}
    Output: {type: object, properties: {part: {$ref: '#/components/schemas/Part'}, detail: {$ref: '#/components/schemas/Detail'}}}
    Part: {type: object, required: [%s], properties: {name: {type: string}, id: {type: string}}}
    Detail:
      type: object
      required: [%s]
      properties: {name: {type: string}, id: {type: string}}
  requestBodies:
    WidgetInput:
      content:
        application/json:
          schema: {$ref: '#/components/schemas/Input'}
  responses:
    WidgetOutput:
      description: ok
      content:
        application/json:
          schema: {$ref: '#/components/schemas/Output'}`
	build := func(inline, filter, page, part, detail string) string {
		return usageSpec(fmt.Sprintf(operation, inline), fmt.Sprintf(schemas, filter, page, part, detail))
	}
	cases := []struct {
		name          string
		old, next     [5]string
		breakingCount int
	}{
		{"inline response schema", [5]string{"name", "name", "name", "name", "name"}, [5]string{"name, id", "name", "name", "name", "name"}, 0},
		{"parameter schema", [5]string{"name", "name", "name", "name", "name"}, [5]string{"name", "name, id", "name", "name", "name"}, 1},
		{"response header schema", [5]string{"name", "name", "name", "name", "name"}, [5]string{"name", "name", "name, id", "name", "name"}, 0},
		{"schema reached from request and response", [5]string{"name", "name", "name", "name", "name"}, [5]string{"name", "name", "name", "name, id", "name"}, 1},
		{"schema reached through a response component", [5]string{"name", "name", "name", "name", "name"}, [5]string{"name", "name", "name", "name", "name, id"}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			changes := compareUsageDocuments(t,
				build(tc.old[0], tc.old[1], tc.old[2], tc.old[3], tc.old[4]),
				build(tc.next[0], tc.next[1], tc.next[2], tc.next[3], tc.next[4]))
			require.NotNil(t, changes)
			assert.Equal(t, 1, changes.TotalChanges())
			assert.Equal(t, tc.breakingCount, changes.TotalBreakingChanges())
		})
	}
}

// Check the required entry itself because changing a schema's use may also change the operation.
func requireEntryBreaking(t *testing.T, left, right string, breaking bool) {
	t.Helper()
	changes := compareUsageDocuments(t, left, right)
	require.NotNil(t, changes)
	for _, ch := range changes.GetAllChanges() {
		if ch.Property == v3.RequiredLabel {
			assert.Equal(t, breaking, ch.Breaking)
			return
		}
	}
	t.Fatal("missing required change")
}

func TestRequiredNeedsExclusiveUseInBothDocuments(t *testing.T) {
	for _, tc := range []struct{ name, left, right string }{
		{"old shared", usageShared, usageResponseOnly},
		{"new shared", usageResponseOnly, usageShared},
		{"old unused", usageUnused, usageResponseOnly},
		{"new unused", usageResponseOnly, usageUnused},
		{"callback", usageCallback, usageCallback},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requireEntryBreaking(t, usageSpec(tc.left, usageWidget("name")), usageSpec(tc.right, usageWidget("name, id")), true)
		})
	}
}

func TestRequiredConservativeSchemaContexts(t *testing.T) {
	const ref = "{$ref: '#/components/schemas/Widget'}"
	for _, tc := range []struct {
		name, wrapper string
		breaking      bool
	}{
		{"negated", "not: " + ref, true},
		{"conditional predicate", "if: " + ref, true},
		{"exclusive union of one branch", "oneOf: [" + ref + "]", false},
		{"array items", "type: array, items: " + ref, false},
		{"allOf reference", "allOf: [" + ref + "]", false},
		{"property named not", "properties: {not: " + ref + "}", false},
		{"literal reference in example", "allOf: [" + ref + "], example: {$ref: '#/components/schemas/Widget'}", false},
		{"dynamic reference", "allOf: [" + ref + ", {$dynamicRef: '#fallback'}]", true},
		{"resource scoped pointer", "$id: 'https://example.com/model', allOf: [" + ref + "]", true},
		{"discriminator", "allOf: [" + ref + "], discriminator: {propertyName: kind}", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			op := strings.Replace(usageResponseOnly, ref, "{"+tc.wrapper+"}", 1)
			requireEntryBreaking(t, usageSpec(op, usageWidget("name")), usageSpec(op, usageWidget("name, id")), tc.breaking)
		})
	}
}

func TestRequiredDocumentFallbacks(t *testing.T) {
	const webhook = "\nwebhooks:\n  incoming:\n    post:\n" + usageRequestOnly
	requireEntryBreaking(t, usageSpec(usageResponseOnly, usageWidget("name"))+webhook, usageSpec(usageResponseOnly, usageWidget("name, id"))+webhook, true)
	l := buildSchemaProxyFromYAML(t, "type: object\nrequired: [name]")
	r := buildSchemaProxyFromYAML(t, "type: object\nrequired: [name, id]")
	assert.Equal(t, 1, CompareSchemas(l, r).TotalBreakingChanges())

	defer ResetActiveBreakingRulesConfig()
	SetActiveBreakingRulesConfig(&BreakingRulesConfig{Schema: &SchemaRules{Required: rule(false, false, false)}})
	requireEntryBreaking(t, usageSpec(usageShared, usageWidget("name")), usageSpec(usageShared, usageWidget("name, id")), false)
}

func TestRequiredNewReadOnlyResponseProperty(t *testing.T) {
	left := usageSpec(usageResponseOnly, usageWidget("name"))
	left = strings.Replace(left, "\n        id: {type: string}", "", 1)
	right := usageSpec(usageResponseOnly, strings.Replace(usageWidget("name, id"), "id: {type: string}", "id: {type: string, readOnly: true}", 1))
	ch := compareUsageDocuments(t, left, right)
	require.NotNil(t, ch)
	assert.Equal(t, 2, ch.TotalChanges())
	assert.Zero(t, ch.TotalBreakingChanges())
}

func TestRequiredUnknownDialectCanHideRequestUse(t *testing.T) {
	const ref = "{$ref: '#/components/schemas/Widget'}"
	const dialect = "http://json-schema.org/draft-07/schema#"
	op := strings.Replace(usageShared, ref, "{dependencies: {name: "+ref+"}}", 1)
	for _, root := range []bool{false, true} {
		build := func(required string) string {
			spec := usageSpec(op, usageWidget(required))
			if root {
				return spec + "\njsonSchemaDialect: " + dialect + "\n"
			}
			return strings.Replace(spec, "{dependencies:", "{$schema: '"+dialect+"', dependencies:", 1)
		}
		requireEntryBreaking(t, build("name"), build("name, id"), true)
	}
}

func TestRequiredClosedResponse(t *testing.T) {
	for _, restriction := range []string{"additionalProperties: false", "unevaluatedProperties: false", "additionalProperties: {type: integer}"} {
		t.Run(restriction, func(t *testing.T) {
			old := usageSpec(usageResponseOnly, usageWidget("name")+"\n      "+restriction)
			next := strings.Replace(old, "required: [name]", "required: [name, id]", 1)
			requireEntryBreaking(t, old, next, false)
			// Property names are case sensitive. An old validator still rejects lowercase id.
			requireEntryBreaking(t, strings.Replace(old, "id: {", "ID: {", 1), next, true)
		})
	}
}

func TestRequiredIgnoresResponseExtensions(t *testing.T) {
	op := usageRequestOnly + "\n        x-docs: {content: {application/json: {schema: {$ref: '#/components/schemas/Widget'}}}}"
	requireEntryBreaking(t, usageSpec(op, usageWidget("name, id")), usageSpec(op, usageWidget("name")), false)
}

func TestRequiredReferencedMergedInlineSchema(t *testing.T) {
	const spec = `openapi: 3.1.0
info: {title: merge, version: '1'}
x-base: &base {type: object}
paths:
  /x:
    post:
      requestBody:
        content:
          application/json:
            schema: {$ref: '#/paths/~1x/post/responses/200/content/application~1json/schema'}
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                <<: *base
                properties: {name: {type: string}, id: {type: string}}
                required: [name]
`
	requireEntryBreaking(t, spec, strings.Replace(spec, "required: [name]", "required: [name, id]", 1), true)
}

func TestRequiredOpenAPI32AdditionalOperation(t *testing.T) {
	const spec = `openapi: 3.2.0
jsonSchemaDialect: https://json-schema.org/draft/2020-12/schema
info: {title: index, version: '1'}
paths:
  /widgets:
    additionalOperations:
      COPY:
        responses:
          '200':
            description: ok
            content: {application/json: {schema: {$ref: '#/components/schemas/Widget'}}}
components:
  schemas:
    Widget: {type: object, properties: {id: {type: string}, name: {type: string}, next: {$ref: '#/components/schemas/Widget'}}, required: [id]}
`
	requireEntryBreaking(t, spec, strings.Replace(spec, "required: [id]", "required: [id, name]", 1), false)
}

func TestRequiredBehindRetargetedResponseReference(t *testing.T) {
	const schemas = `    Before: {type: object, properties: {id: {type: string}}}
    After: {type: object, required: [id], properties: {id: {type: string}}}`
	const widget = "#/components/schemas/Widget"
	for _, op := range []string{usageResponseOnly, usageShared} {
		before := strings.ReplaceAll(op, widget, "#/components/schemas/Before")
		i := strings.LastIndex(before, "#/components/schemas/Before")
		after := before[:i] + strings.Replace(before[i:], "/Before", "/After", 1)
		requireEntryBreaking(t, usageSpec(before, schemas), usageSpec(after, schemas), op == usageShared)
	}
}

func TestSchemaUsageUnresolvedReferences(t *testing.T) {
	for _, ref := range []string{"external.yaml#/Thing", "#/components/pathItems/Missing"} {
		var root yaml.Node
		require.NoError(t, yaml.Unmarshal([]byte("paths: {/widgets: {$ref: '"+ref+"'}}"), &root))
		idx := index.NewSpecIndexWithConfig(&root, index.CreateClosedAPIIndexConfig())
		require.Nil(t, buildSchemaUsage(idx))
	}
}

func TestRequiredEncodingHeader(t *testing.T) {
	const op = `      requestBody:
        content:
          multipart/mixed:
            schema: {type: array}
            itemEncoding:
              headers:
                X-Widget: {schema: {$ref: '#/components/schemas/Widget'}}
      responses:
        '204': {description: ok}`
	build := func(required string) string {
		return strings.Replace(usageSpec(op, usageWidget(required)), "3.1.0", "3.2.0", 1)
	}
	requireEntryBreaking(t, build("name, id"), build("name"), false)
}

func TestRequiredDictionaryValue(t *testing.T) {
	old := usageSpec(usageResponseOnly, `    Widget: {type: object, additionalProperties: {type: string}, required: [name]}`)
	next := strings.Replace(old, "[name]", "[name, id]", 1)
	requireEntryBreaking(t, old, next, false)
	requireEntryBreaking(t, old, strings.Replace(next, "additionalProperties: {type: string}", "additionalProperties: true", 1), true)
}

func TestRequiredResponseClosedByParent(t *testing.T) {
	const ref = "{$ref: '#/components/schemas/Widget'}"
	for _, applicator := range []string{"allOf: [" + ref + "]", "if: {type: object}, then: " + ref} {
		op := strings.Replace(usageResponseOnly, ref, "{unevaluatedProperties: false, "+applicator+"}", 1)
		left := strings.Replace(usageSpec(op, usageWidget("name")), "\n        id: {type: string}", "", 1)
		requireEntryBreaking(t, left, usageSpec(op, usageWidget("name, id")), true)
	}
}

func TestUsageRequiresAReachableReferenceTarget(t *testing.T) {
	doc := buildUsageDocument(t, usageSpec(usageResponseOnly, usageWidget("name"))+"\n    Alias: {$ref: '#/components/schemas/Widget'}")
	proxy := doc.Components.Value.FindSchema("Alias").Value
	require.True(t, proxy.IsReference())
	// When only the reference site is known, its target may be used in either direction.
	usage := schemaUsage{proxy.GetValueNode(): responseSchema}
	assert.Zero(t, usage.direction(proxy))
}

func TestSchemaUsageEmptyPaths(t *testing.T) {
	spec := "openapi: 3.1.0\ninfo: {title: Empty, version: '1'}\npaths:\n"
	doc := buildUsageDocument(t, spec)
	assert.Empty(t, buildSchemaUsage(doc.Index))
}

func TestRequiredNewPatternInClosedResponse(t *testing.T) {
	old := usageSpec(usageResponseOnly, `    Widget: {type: object, additionalProperties: false, properties: {name: {type: string}}, required: [name]}`)
	next := strings.Replace(old, "required: [name]", "patternProperties: {'^id$': {type: string}}, required: [name, id]", 1)
	requireEntryBreaking(t, old, next, true)
}
