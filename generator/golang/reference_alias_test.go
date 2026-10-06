// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
)

func TestRenderSchemasPreservesReferenceOnlyComponentName(t *testing.T) {
	document, err := libopenapi.NewDocument([]byte(`openapi: 3.1.0
info: {title: aliases, version: 1.0.0}
paths: {}
components:
  schemas:
    Canonical: {type: object, properties: {id: {type: string}}}
    Public: {$ref: "#/components/schemas/Canonical"}
`))
	if err != nil {
		t.Fatal(err)
	}
	model, err := document.BuildV3Model()
	if err != nil {
		t.Fatal(err)
	}
	generated, err := NewGenerator().RenderSchemas(model.Model.Components.Schemas)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(generated.Source), "type Public = Canonical") {
		t.Fatalf("reference-only component alias was not rendered:\n%s", generated.Source)
	}
}

func TestReferenceAliasSkipsNilAndSelfReferences(t *testing.T) {
	generator := NewGenerator()
	generator.renderReferenceAliasDecl(nil)
	generator.renderReferenceAliasDecl(&SchemaIR{Name: "Canonical", Ref: "#/components/schemas/Canonical", Kind: KindRef})
	if len(generator.decls) != 0 {
		t.Fatalf("self reference emitted an alias: %v", generator.decls)
	}
}

func TestReferenceAliasTargetsGeneratedUnionName(t *testing.T) {
	document, err := libopenapi.NewDocument([]byte(`openapi: 3.1.0
info: {title: aliases, version: 1.0.0}
paths: {}
components:
  schemas:
    Canonical:
      oneOf: [{type: string}, {type: integer}]
    Public: {$ref: "#/components/schemas/Canonical"}
`))
	if err != nil {
		t.Fatal(err)
	}
	model, err := document.BuildV3Model()
	if err != nil {
		t.Fatal(err)
	}
	generated, err := NewGenerator().RenderSchemas(model.Model.Components.Schemas)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(generated.Source), "type Public = CanonicalUnion") {
		t.Fatalf("reference-only union alias targeted the wrong type:\n%s", generated.Source)
	}
}
