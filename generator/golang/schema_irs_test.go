// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"os"
	"testing"

	"github.com/pb33f/libopenapi"
)

func TestSchemaIRsFollowInputOrderWithoutRendering(t *testing.T) {
	spec, err := os.ReadFile("testdata/train-travel.yaml")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := libopenapi.NewDocument(spec)
	if err != nil {
		t.Fatal(err)
	}
	model, err := doc.BuildV3Model()
	if err != nil {
		t.Fatal(err)
	}
	schemas := model.Model.Components.Schemas

	g := NewGenerator(WithTypeNameResolver(func(name string) string { return name }))
	irs, _, err := g.SchemaIRs(schemas)
	if err != nil {
		t.Fatal(err)
	}
	if len(irs) != schemas.Len() {
		t.Fatalf("got %d IRs for %d schemas", len(irs), schemas.Len())
	}
	i := 0
	for name := range schemas.FromOldest() {
		if irs[i].Name != name {
			t.Fatalf("IR %d is %q, want %q", i, irs[i].Name, name)
		}
		i++
	}

	// Building IR must leave the reusable generator able to render the same
	// document byte-for-byte as a fresh generator.
	again, err := g.RenderSchemas(schemas)
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := NewGenerator(WithTypeNameResolver(func(name string) string { return name })).RenderSchemas(schemas)
	if err != nil {
		t.Fatal(err)
	}
	if string(again.Source) != string(fresh.Source) {
		t.Fatal("SchemaIRs changed the configuration a later RenderSchemas uses")
	}

	if irs, diagnostics, err := g.SchemaIRs(nil); err != nil || irs != nil || diagnostics != nil {
		t.Fatalf("nil schemas: got %v, %v, %v", irs, diagnostics, err)
	}
}
