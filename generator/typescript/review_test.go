// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package typescript

import (
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/generator/golang"
	"github.com/pb33f/libopenapi/orderedmap"
)

const shapesSpec = `openapi: 3.0.3
info:
  title: Shapes
  version: 1.0.0
paths: {}
components:
  schemas:
    Bag:
      type: object
      required: [id]
      properties:
        id:
          type: string
      additionalProperties:
        type: integer
    BagValue:
      $ref: '#/components/schemas/Bag/additionalProperties'
    Keyed:
      type: object
      required: [id]
    Counts:
      type: object
      required: [total]
      additionalProperties:
        type: integer
    Partial:
      type: object
      required: [id, kind]
      properties:
        id:
          type: string
    Composed:
      allOf:
        - $ref: '#/components/schemas/Partial'
      properties:
        extra:
          type: string
    Lost:
      $ref: '#/components/schemas/Composed/properties/extra'
    await:
      type: string
    as:
      type: string
    declare:
      type: string
`

func TestSchemaShapesFromReview(t *testing.T) {
	doc, err := libopenapi.NewDocument([]byte(shapesSpec))
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
		// A pointer to additionalProperties keeps the value type even though
		// the container's index signature cannot.
		"export type BagValue = number;",
		// Required names without declared properties stay required.
		"export type Keyed = {\n  id: unknown;\n};",
		"export type Counts = {\n  total: number;\n  [key: string]: number;\n};",
		"export interface Partial {\n  id: string;\n  kind: unknown;\n}",
		// Names TypeScript rejects are renamed; names it accepts are kept.
		"export type Await = string;",
		"export type As = string;",
		"export type declare = string;",
	} {
		if !strings.Contains(source, want) {
			t.Errorf("missing %q in:\n%s", want, source)
		}
	}
	// allOf sibling properties are a documented IR limitation, so a pointer to
	// one cannot be resolved and is reported rather than guessed.
	if !strings.Contains(source, "export type Lost = unknown;") {
		t.Errorf("unresolvable pointer not rendered as unknown:\n%s", source)
	}
	reported := false
	for _, d := range file.Diagnostics {
		if d.Code == DiagnosticUnsupportedPointer && strings.HasSuffix(d.Path, "Composed/properties/extra") {
			reported = true
		}
	}
	if !reported {
		t.Errorf("missing %s diagnostic: %+v", DiagnosticUnsupportedPointer, file.Diagnostics)
	}
}

func TestPointerTargetRejectsPathsTheIRCannotFollow(t *testing.T) {
	object := &golang.SchemaIR{Kind: golang.KindObject, Properties: orderedmap.New[string, *golang.SchemaIR]()}
	object.Properties.Set("name", &golang.SchemaIR{Kind: golang.KindString})
	r := &render{}
	for name, tc := range map[string]struct {
		ir       *golang.SchemaIR
		segments []string
	}{
		"ends at properties":    {object, []string{"properties"}},
		"no properties":         {&golang.SchemaIR{Kind: golang.KindString}, []string{"properties", "x"}},
		"undeclared property":   {object, []string{"properties", "missing"}},
		"no items":              {object, []string{"items"}},
		"other keyword":         {object, []string{"allOf", "0"}},
		"step past a primitive": {object, []string{"properties", "name", "items"}},
	} {
		if target, ok := r.pointerTarget(tc.ir, tc.segments); ok {
			t.Errorf("%s: resolved to %+v", name, target)
		}
	}
}
