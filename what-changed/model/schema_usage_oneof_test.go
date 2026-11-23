// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"context"
	"strings"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestRequiredOneOfBranches(t *testing.T) {
	const widget = "{$ref: '#/components/schemas/Widget'}"
	const extra = `
    List: {type: array, items: {$ref: '#/components/schemas/Widget'}}
    Page: {type: object, properties: {item: {$ref: '#/components/schemas/Widget'}}}`
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
		{"reference next to a type", "oneOf: [" + widget + ", {$ref: '#/components/schemas/Page', type: string}]", true},
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
	const other = "\n    Other: {type: object, properties: {other: {type: string}}}"
	for _, tc := range []struct {
		name, schema string
		breaking     bool
	}{
		// OpenAPI 3.0 ignores the siblings of $ref, so the second branch is an object like Widget.
		{"reference next to a type", "oneOf: [" + widget + ", {$ref: '#/components/schemas/Other', type: string}]", true},
		{"nullable object or string", "oneOf: [{type: object, nullable: true, properties: {item: " + widget + "}}, {type: string}]", false},
		{"nullable branches", "oneOf: [{type: object, nullable: true, properties: {item: " + widget + "}}, {type: string, nullable: true}]", true},
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
    Loop: {$ref: '#/components/schemas/Loop'}`
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
		{"reference to nothing", "{oneOf: [" + widget + ", {$ref: '#/components/schemas/Missing'}]}", 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			direction, known := widgetUsage(t, tc.schema)
			assert.Equal(t, tc.known, known)
			assert.Equal(t, tc.direction, direction)
		})
	}
}
