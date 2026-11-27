// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"context"
	"strings"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel"
	highv3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestRequiredOneOfBranches(t *testing.T) {
	const widget = "{$ref: '#/components/schemas/Widget'}"
	const extra = `
    List: {type: array, items: {$ref: '#/components/schemas/Widget'}}
    Page: {type: object, properties: {item: {$ref: '#/components/schemas/Widget'}}}
    Alias1: {$ref: '#/components/schemas/Alias2'}
    Alias2: {$ref: '#/components/schemas/Alias3'}
    Alias3: {$ref: '#/components/schemas/Alias4'}
    Alias4: {$ref: '#/components/schemas/Alias5'}
    Alias5: {$ref: '#/components/schemas/Alias6'}
    Alias6: {$ref: '#/components/schemas/Alias7'}
    Alias7: {$ref: '#/components/schemas/Alias8'}
    Alias8: {type: string}
    StringOrObject: {type: [string, object]}
    Narrowed: {$ref: '#/components/schemas/StringOrObject', type: string}
    Untyped: {minLength: 1}
    Anything: {}
    Everything: true
    Nothing: false
    Named: {type: &object string}
    AliasedType: {type: [*object]}`
	for _, tc := range []struct {
		name, schema string
		breaking     bool
	}{
		{"single branch", "oneOf: [" + widget + "]", false},
		{"array or object", "oneOf: [{type: array, items: " + widget + "}, {type: object, properties: {widgets: {type: array, items: " + widget + "}}}]", false},
		{"branches by reference", "oneOf: [{$ref: '#/components/schemas/List'}, {$ref: '#/components/schemas/Page'}]", false},
		{"null or object", "oneOf: [{type: 'null'}, " + widget + "]", false},
		{"list of types", "oneOf: [{type: [string, number]}, {type: [array, 'null'], items: " + widget + "}]", false},
		{"two objects", "oneOf: [" + widget + ", {type: object, required: [other], properties: {other: {type: string}}}]", true},
		{"two arrays", "oneOf: [{type: array, items: " + widget + "}, {type: array, items: {type: string}}]", true},
		{"integer and number", "oneOf: [{type: integer}, {type: number}, " + widget + "]", true},
		{"branch without a type", "oneOf: [" + widget + ", {properties: {other: {type: string}}}]", true},
		{"boolean branch", "oneOf: [" + widget + ", true]", true},
		{"type that is not a JSON type", "oneOf: [" + widget + ", {type: Object}]", true},
		{"integer and number in one branch", "oneOf: [{type: [integer, number]}, " + widget + "]", false},
		// The type next to the reference and the type behind it have nothing in common, which is more likely a mistake
		// than a schema that accepts nothing.
		{"reference next to a type", "oneOf: [" + widget + ", {$ref: '#/components/schemas/Page', type: string}]", true},
		// In OpenAPI 3.1 the siblings of $ref apply as well, so a type next to a reference narrows it.
		{"reference next to a narrower type", "oneOf: [{type: object, properties: {item: " + widget + "}}, {$ref: '#/components/schemas/StringOrObject', type: string}]", false},
		{"reference narrowed to the type of another branch", "oneOf: [{type: object, properties: {item: " + widget + "}}, {$ref: '#/components/schemas/StringOrObject', type: object}]", true},
		{"reference without a type next to a type", "oneOf: [{type: object, properties: {item: " + widget + "}}, {$ref: '#/components/schemas/Untyped', type: string}]", false},
		{"type next to a reference along the chain", "oneOf: [{type: object, properties: {item: " + widget + "}}, {$ref: '#/components/schemas/Narrowed'}]", false},
		{"reference next to a type that is not a JSON type", "oneOf: [{type: object, properties: {item: " + widget + "}}, {$ref: '#/components/schemas/Alias8', type: Object}]", true},
		// The boolean schema true accepts every instance like {}, so the type next to either reference is what counts.
		{"reference to an empty schema next to a type", "oneOf: [{type: object, properties: {item: " + widget + "}}, {$ref: '#/components/schemas/Anything', type: string}]", false},
		{"reference to true next to a type", "oneOf: [{type: object, properties: {item: " + widget + "}}, {$ref: '#/components/schemas/Everything', type: string}]", false},
		// The boolean schema false accepts nothing, which is more likely a mistake than a branch that no instance matches.
		{"reference to false next to a type", "oneOf: [{type: object, properties: {item: " + widget + "}}, {$ref: '#/components/schemas/Nothing', type: string}]", true},
		// An alias in a list of types reads the value it points to, not the name of its anchor.
		{"alias in a list of types", "oneOf: [" + widget + ", {$ref: '#/components/schemas/AliasedType'}]", false},
		// An unquoted null is not a type name: the schema model leaves the type unset, so the branch accepts anything.
		{"type that is a YAML null", "oneOf: [" + widget + ", {type: null}]", true},
		// In OpenAPI 3.1 nullable has no effect, so the object branch does not accept null.
		{"nullable in OpenAPI 3.1", "oneOf: [{type: object, nullable: true, properties: {item: " + widget + "}}, {type: 'null'}]", false},
		// The length of a chain of references does not change the types its end accepts.
		{"chain of eight references", "oneOf: [{type: object, properties: {item: " + widget + "}}, {$ref: '#/components/schemas/Alias1'}]", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			op := strings.Replace(usageResponseOnly, widget, "{"+tc.schema+"}", 1)
			schemas := func(required string) string {
				return usageWidget(required) + extra
			}
			requireEntryBreaking(t, usageSpec(op, schemas("name")), usageSpec(op, schemas("name, id")), tc.breaking)
		})
	}
}

// Rendering a document and reading it back must not change whether the branches are exclusive.
func TestRequiredOneOfBranchesAfterRendering(t *testing.T) {
	const widget = "{$ref: '#/components/schemas/Widget'}"
	for _, tc := range []struct{ name, schema, extra string }{
		{"alias in a list of types", "oneOf: [" + widget + ", {$ref: '#/components/schemas/Other'}]", "\n    x-type: &object string\n    Other: {type: [*object]}"},
		{"reference to true next to a type", "oneOf: [{type: object, properties: {item: " + widget + "}}, {$ref: '#/components/schemas/Other', type: string}]", "\n    Other: true"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			op := strings.Replace(usageResponseOnly, widget, "{"+tc.schema+"}", 1)
			left, right := usageSpec(op, usageWidget("name")+tc.extra), usageSpec(op, usageWidget("name, id")+tc.extra)
			requireEntryBreaking(t, left, right, false)
			render := func(spec string) string {
				rendered, err := highv3.NewDocument(buildUsageDocument(t, spec)).Render()
				require.NoError(t, err)
				return string(rendered)
			}
			requireEntryBreaking(t, render(left), render(right), false)
		})
	}
}

func TestRequiredOneOfBranchesKeepTheirDirection(t *testing.T) {
	const widget = "{$ref: '#/components/schemas/Widget'}"
	union := "oneOf: [{type: array, items: " + widget + "}, {type: object, properties: {widgets: {type: array, items: " + widget + "}}}]"
	// The same union in a request makes the schema shared with the response, so it must stay breaking.
	op := strings.Replace(usageShared, widget, "{"+union+"}", 1)
	requireEntryBreaking(t, usageSpec(op, usageWidget("name")), usageSpec(op, usageWidget("name, id")), true)
	// A removed requirement is only safe for a request, so a union used by a response alone must stay breaking.
	op = strings.Replace(usageResponseOnly, widget, "{"+union+"}", 1)
	requireEntryBreaking(t, usageSpec(op, usageWidget("name, id")), usageSpec(op, usageWidget("name")), true)
}

func TestRequiredOneOfBranchesOpenAPI30(t *testing.T) {
	const widget = "{$ref: '#/components/schemas/Widget'}"
	const other = "\n    Other: {type: object, properties: {other: {type: string}}}\n    Untyped: {minLength: 1}"
	for _, tc := range []struct {
		name, schema string
		breaking     bool
	}{
		// OpenAPI 3.0 ignores the siblings of $ref, so the second branch is an object like Widget.
		{"reference next to a type", "oneOf: [" + widget + ", {$ref: '#/components/schemas/Other', type: string}]", true},
		{"reference without a type next to a type", "oneOf: [{type: object, properties: {item: " + widget + "}}, {$ref: '#/components/schemas/Untyped', type: string}]", true},
		{"nullable object or string", "oneOf: [{type: object, nullable: true, properties: {item: " + widget + "}}, {type: string}]", false},
		{"nullable branches", "oneOf: [{type: object, nullable: true, properties: {item: " + widget + "}}, {type: string, nullable: true}]", true},
		// YAML resolves True and TRUE to the same boolean as true, and the schema model reads them as nullable.
		{"nullable branches spelled True", "oneOf: [{type: object, nullable: true, properties: {item: " + widget + "}}, {type: string, nullable: True}]", true},
		{"nullable branches spelled TRUE", "oneOf: [{type: object, nullable: true, properties: {item: " + widget + "}}, {type: string, nullable: TRUE}]", true},
		// A quoted 'true' is a string, which the schema model does not read as nullable.
		{"nullable that is a string", "oneOf: [{type: object, nullable: true, properties: {item: " + widget + "}}, {type: string, nullable: 'true'}]", false},
		// An empty list of types cannot be compared, and nullable does not change that.
		{"nullable empty list of types", "oneOf: [" + widget + ", {type: [], nullable: true}]", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			op := strings.Replace(usageResponseOnly, widget, "{"+tc.schema+"}", 1)
			build := func(required string) string {
				return strings.Replace(usageSpec(op, usageWidget(required)+other), "openapi: 3.1.0", "openapi: 3.0.3", 1)
			}
			requireEntryBreaking(t, build("name"), build("name, id"), tc.breaking)
		})
	}
}

// Without TransformSiblingRefs the schema model reads only the reference, so a type next to it narrows nothing.
func TestRequiredOneOfBranchesWithoutSiblingTransform(t *testing.T) {
	const widget = "{$ref: '#/components/schemas/Widget'}"
	union := "oneOf: [{type: object, properties: {item: " + widget + "}}, {$ref: '#/components/schemas/StringOrObject', type: string}]"
	op := strings.Replace(usageResponseOnly, widget, "{"+union+"}", 1)
	build := func(required string) *v3.Document {
		info, err := datamodel.ExtractSpecInfo([]byte(usageSpec(op, usageWidget(required)+"\n    StringOrObject: {type: [string, object]}")))
		require.NoError(t, err)
		config := datamodel.NewDocumentConfiguration()
		config.TransformSiblingRefs = false
		doc, err := v3.CreateDocumentFromConfig(info, config)
		require.NoError(t, err)
		return doc
	}
	changes := CompareDocuments(build("name"), build("name, id"))
	require.NotNil(t, changes)
	for _, ch := range changes.GetAllChanges() {
		if ch.Property == v3.RequiredLabel {
			assert.True(t, ch.Breaking)
			return
		}
	}
	t.Fatal("missing required change")
}

func TestRequiredOneOfBranchesInARequest(t *testing.T) {
	const widget = "{$ref: '#/components/schemas/Widget'}"
	for _, tc := range []struct {
		name, schema string
		breaking     bool
	}{
		// A request may now leave out the property, and no instance can match the other branch as well.
		{"branches that accept different types", "oneOf: [" + widget + ", {type: array}]", false},
		{"two objects", "oneOf: [" + widget + ", {type: object, required: [other], properties: {other: {type: string}}}]", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			op := strings.Replace(usageRequestOnly, widget, "{"+tc.schema+"}", 1)
			requireEntryBreaking(t, usageSpec(op, usageWidget("name, id")), usageSpec(op, usageWidget("name")), tc.breaking)
		})
	}
}

// widgetUsage returns the direction of Widget in a document whose only response schema is the given YAML. The second
// result is false when the usage of the document is unknown.
func widgetUsage(t *testing.T, schema string) (schemaDirection, bool) {
	t.Helper()
	spec := `openapi: 3.1.0
info: {title: Direction, version: 1.0.0}
paths:
  /widgets:
    get:
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema: ` + schema + `
components:
  schemas:
    Widget: {type: object}
    Loop: {$ref: '#/components/schemas/Loop'}
    LoopA: {$ref: '#/components/schemas/LoopB'}
    LoopB: {$ref: '#/components/schemas/LoopA'}
    Untyped: {minLength: 1}`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(spec), &root))
	idx := index.NewSpecIndexWithConfig(&root, index.CreateClosedAPIIndexConfig())
	usage := buildSchemaUsage(idx)
	if usage == nil {
		return 0, false
	}
	widget := idx.FindComponentInRoot(context.Background(), "#/components/schemas/Widget")
	require.NotNil(t, widget)
	return usage[widget.Node], true
}

func TestSchemaUsageOneOfBranches(t *testing.T) {
	const widget = "{$ref: '#/components/schemas/Widget'}"
	for _, tc := range []struct {
		name, schema string
		direction    schemaDirection
		known        bool
	}{
		{"branches that accept different types", "{oneOf: [" + widget + ", {type: array}]}", responseSchema, true},
		{"oneOf that is not a list", "{oneOf: {first: " + widget + "}}", sharedSchema, true},
		{"type that is not a name", "{oneOf: [" + widget + ", {type: [string, {a: b}]}]}", sharedSchema, true},
		{"type that is not a name or a list", "{oneOf: [" + widget + ", {type: {a: b}}]}", sharedSchema, true},
		{"reference that cannot be resolved next to a type", "{oneOf: [" + widget + ", {$ref: '#/components/schemas/Missing', type: string}]}", 0, false},
		{"empty list of types", "{oneOf: [" + widget + ", {type: []}]}", sharedSchema, true},
		{"reference cycle", "{oneOf: [" + widget + ", {$ref: '#/components/schemas/Loop'}]}", sharedSchema, true},
		{"reference cycle through two schemas", "{oneOf: [" + widget + ", {$ref: '#/components/schemas/LoopA'}]}", sharedSchema, true},
		{"empty list of types that is nullable", "{oneOf: [" + widget + ", {type: [], nullable: true}]}", sharedSchema, true},
		// Without a known version, the siblings of $ref may be ignored, so only the reference counts.
		{"reference without a type next to a type in an unknown version", "{oneOf: [" + widget + ", {$ref: '#/components/schemas/Untyped', type: string}]}", sharedSchema, true},
		{"reference to nothing", "{oneOf: [" + widget + ", {$ref: '#/components/schemas/Missing'}]}", 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			direction, known := widgetUsage(t, tc.schema)
			assert.Equal(t, tc.known, known)
			assert.Equal(t, tc.direction, direction)
		})
	}
}
