// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package typescript

import (
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/generator/golang"
)

const edgesSpec = `openapi: 3.1.0
info:
  title: Edges
  version: 1.0.0
paths: {}
components:
  schemas:
    Flags:
      type: object
      required: [on, level, mode, nothing]
      properties:
        on:
          const: true
        level:
          const: 2.5
        mode:
          type: [string, "null"]
          enum: [fast, slow, null]
        nothing:
          type: "null"
        either:
          type: [string, integer]
    Composite:
      allOf:
        - type: object
          properties:
            x:
              type: string
    Deep:
      $ref: '#/components/schemas/Composite/allOf/0'
    Both:
      allOf:
        - oneOf:
            - type: string
            - type: integer
        - $ref: '#/components/schemas/Flags'
    Choice:
      oneOf:
        - type: string
        - type: integer
    Paragraphs:
      type: string
      description: |-
        First paragraph.

        Second paragraph.
    Values:
      type: object
      additionalProperties:
        $ref: '#/components/schemas/Choice'
    Tail:
      $ref: '#/components/schemas/Values/additionalProperties'
`

func TestEdgeCaseRendering(t *testing.T) {
	doc, err := libopenapi.NewDocument([]byte(edgesSpec))
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
		"on: true;",
		"level: 2.5;",
		`mode: "fast" | "slow" | null;`,
		"nothing: null;",
		"either?: string | number;",
		"export type Deep = unknown;",
		"export type Both = (string | number) & Flags;",
		" * First paragraph.\n *\n * Second paragraph.",
		"export type Values = { [key: string]: Choice };",
		"export type Tail = Choice;",
	} {
		if !strings.Contains(source, want) {
			t.Errorf("missing %q in:\n%s", want, source)
		}
	}
	codes := make(map[string]bool)
	for _, d := range file.Diagnostics {
		codes[d.Code] = true
	}
	for _, code := range []string{DiagnosticUnsupportedPointer} {
		if !codes[code] {
			t.Errorf("missing diagnostic %s: %+v", code, file.Diagnostics)
		}
	}
}

// An external reference cannot be resolved by a document built offline, so
// the renderer is exercised directly: it names the type after the reference's
// final segment and reports the decision.
func TestExternalReferenceRendersByName(t *testing.T) {
	r := &render{names: map[string]string{}}
	got := r.expr(&golang.SchemaIR{Kind: golang.KindRef, Ref: "https://example.com/schemas/remote.yaml#/remote_thing"}, "")
	if got != "RemoteThing" {
		t.Fatalf("got %q", got)
	}
	if len(r.diagnostics) != 1 || r.diagnostics[0].Code != DiagnosticExternalReference {
		t.Fatalf("diagnostics: %+v", r.diagnostics)
	}
}
