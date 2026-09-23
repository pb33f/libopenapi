// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package typescript

import (
	"errors"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/generator/golang"
	"github.com/pb33f/libopenapi/orderedmap"
)

const literalSpec = `openapi: 3.0.3
info:
  title: Literals
  version: 1.0.0
paths: {}
components:
  schemas:
    Untyped:
      type: array
    Structured:
      enum: [{a: 1}]
    Repeated:
      type: string
      enum: [a, a, b]
    Truthy:
      type: boolean
      enum: [True, FALSE]
    Hex:
      type: integer
      enum: [0x1F]
    Twice:
      oneOf:
        - type: string
        - type: string
    Either:
      oneOf:
        - type: string
        - type: integer
    Wrapped:
      oneOf:
        - $ref: '#/components/schemas/Either'
      discriminator:
        propertyName: kind
        mapping:
          either: '#/components/schemas/Either'
    Counted:
      type: object
      required: [kind]
      properties:
        kind:
          type: integer
    NotANumber:
      oneOf:
        - $ref: '#/components/schemas/Counted'
      discriminator:
        propertyName: kind
        mapping:
          abc: '#/components/schemas/Counted'
`

func TestLiteralAndUnionEdges(t *testing.T) {
	doc, err := libopenapi.NewDocument([]byte(literalSpec))
	if err != nil {
		t.Fatal(err)
	}
	model, err := doc.BuildV3Model()
	if err != nil {
		t.Fatal(err)
	}
	file, err := RenderSchemas(model.Model.Components.Schemas)
	if err != nil {
		t.Fatal(err)
	}
	source := string(file.Source)
	for _, want := range []string{
		"export type Untyped = unknown[];",   // array without items
		"export type Structured = unknown;",  // a non-scalar enum value has no literal type
		`export type Repeated = "a" | "b";`,  // duplicate enum values collapse
		"export type Truthy = true | false;", // YAML spellings of booleans
		"export type Hex = unknown;",         // a YAML integer that is not a JSON number
		"export type Twice = string;",        // identical variants collapse
		"export type Wrapped = Either;",      // a variant without properties is not narrowed
		"export type NotANumber = Counted;",  // a mapping key that does not fit an integer property
	} {
		if !strings.Contains(source, want) {
			t.Errorf("missing %q in:\n%s", want, source)
		}
	}
}

func TestRenderSchemasReportsIRErrors(t *testing.T) {
	schemas := orderedmap.New[string, *highbase.SchemaProxy]()
	schemas.Set("Missing", nil)
	if _, err := RenderSchemas(schemas); !errors.Is(err, golang.ErrNilSchema) {
		t.Fatalf("got %v, want %v", err, golang.ErrNilSchema)
	}
}

func TestIndexedAccessRejectsIncompletePointer(t *testing.T) {
	if expr, ok := indexedAccess("Account", []string{"properties"}); ok {
		t.Fatalf("pointer ending at properties rendered as %q", expr)
	}
}
