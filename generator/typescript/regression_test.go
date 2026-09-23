// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package typescript

import (
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/generator/golang"
)

// Each case pins a shape found by differential testing against JSON Schema
// semantics or by review, with the output that type-checks and accepts the
// same values as the schema.
const regressionSpec = `openapi: 3.1.0
info:
  title: Regressions
  version: 1.0.0
paths: {}
components:
  schemas:
    JsonValue:
      oneOf:
        - type: string
        - type: number
        - type: boolean
        - type: 'null'
        - type: array
          items:
            $ref: '#/components/schemas/JsonValue'
        - type: object
          additionalProperties:
            $ref: '#/components/schemas/JsonValue'
    Tree:
      type: object
      additionalProperties:
        $ref: '#/components/schemas/Tree'
    Base:
      type: object
      properties:
        id:
          type: string
    Saved:
      allOf:
        - $ref: '#/components/schemas/Base'
        - required: [id]
    Described:
      $ref: '#/components/schemas/Base'
      description: A Base with its own description.
    Prefixed:
      type: object
      patternProperties:
        '^x-':
          type: string
      additionalProperties: false
    Named:
      type: object
      properties:
        name:
          type: string
      patternProperties:
        '^x-':
          type: string
    Pair:
      type: array
      prefixItems:
        - type: integer
        - type: string
      items:
        type: boolean
    Closed:
      type: array
      prefixItems:
        - enum: ['a"b', c]
      items: false
    Open:
      type: array
      prefixItems:
        - type: integer
    Level:
      type: [string, 'null']
      enum: [low, high]
    MaybeLevel:
      type: [string, 'null']
      enum: [low, null]
    Fixed:
      type: [string, 'null']
      const: fixed
    Name:
      type: [string, 'null']
    Maybe:
      type: object
      required: [id]
      properties:
        id:
          type: string
      anyOf:
        - type: [object, 'null']
          properties:
            y:
              type: string
    OnlyDescribed:
      allOf:
        - description: Adds nothing to the shape.
    Extended:
      allOf:
        - $ref: '#/components/schemas/Base'
      properties:
        extra:
          type: object
          properties:
            c:
              type: object
              additionalProperties: [x]
    Unbuildable:
      allOf:
        - $ref: '#/components/schemas/Base'
      properties:
        missing:
          type: object
          additionalProperties: [x]
    Broken:
      type: object
      properties:
        c:
          type: object
          additionalProperties: [x]
`

func TestRegressionShapes(t *testing.T) {
	doc, err := libopenapi.NewDocument([]byte(regressionSpec))
	if err != nil {
		t.Fatal(err)
	}
	model, _ := doc.BuildV3Model()
	if model == nil {
		t.Fatal("model was not built")
	}
	file, err := RenderSchemas(model.Model.Components.Schemas)
	if err != nil {
		t.Fatal(err)
	}
	source := string(file.Source)
	for _, want := range []string{
		// Recursive maps use index-signature literals, which may refer back
		// to their own alias; Record<string, T> may not.
		"export type JsonValue = string | number | boolean | null | JsonValue[] | { [key: string]: JsonValue };",
		"export type Tree = { [key: string]: Tree };",
		// A member that only makes an inherited property required keeps it
		// required.
		"export type Saved = Base & {\n  id: unknown;\n};",
		// A description beside $ref does not add an unknown member.
		"export type Described = Base;",
		// patternProperties admit their keys even when additionalProperties
		// is false.
		"export type Prefixed = { [key: string]: unknown };",
		"export interface Named {\n  name?: string;\n  [key: string]: unknown;\n}",
		// prefixItems are optional tuple positions; items types the rest,
		// items: false closes the tuple, and no items leaves it open.
		"export type Pair = [number?, string?, ...boolean[]];",
		`export type Closed = [("a\"b" | "c")?];`,
		"export type Open = [number?, ...unknown[]];",
		// A 3.1 enum or const that leaves out null excludes it.
		`export type Level = "low" | "high";`,
		`export type MaybeLevel = "low" | null;`,
		`export type Fixed = "fixed";`,
		"export type Name = string | null;",
		// Sibling members intersect the whole union, including its null.
		"} & ({\n  y?: string;\n} | null);",
		// A nested schema that cannot be built renders as unknown.
		"c?: unknown;",
		// allOf members that add nothing leave nothing to intersect.
		"export type OnlyDescribed = unknown;",
	} {
		if !strings.Contains(source, want) {
			t.Errorf("missing %q in:\n%s", want, source)
		}
	}
	// Nested build failures are reported for component properties and for
	// properties declared beside allOf; a sibling set that cannot be built
	// at all is reported and left out.
	paths := map[string]bool{}
	for _, d := range file.Diagnostics {
		if d.Code == golang.DiagnosticChildSchema {
			paths[d.Path] = true
		}
	}
	for _, want := range []string{"Broken.c", "extra.c", "Unbuildable"} {
		if !paths[want] {
			t.Errorf("missing %s diagnostic for %s: %+v", golang.DiagnosticChildSchema, want, file.Diagnostics)
		}
	}
	if strings.Contains(source, "missing?:") {
		t.Errorf("unbuildable sibling property rendered:\n%s", source)
	}
}

func TestParenthesizeUnionOnlyWrapsTopLevelUnions(t *testing.T) {
	for expr, want := range map[string]string{
		"A":                          "A",
		"A | B":                      "(A | B)",
		"{\n  a?: string | null;\n}": "{\n  a?: string | null;\n}",
		"Array<A | B>":               "Array<A | B>",
		`"x | y"`:                    `"x | y"`,
		`"a\" | b"`:                  `"a\" | b"`,
		`"q" | "r"`:                  `("q" | "r")`,
	} {
		if got := parenthesizeUnion(expr); got != want {
			t.Errorf("parenthesizeUnion(%q) = %q, want %q", expr, got, want)
		}
	}
}
