// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
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

func TestSchemaIRsReportsBuildErrors(t *testing.T) {
	schemas := orderedmap.New[string, *highbase.SchemaProxy]()
	schemas.Set("Missing", nil)
	if _, _, err := NewGenerator().SchemaIRs(schemas); !errors.Is(err, ErrNilSchema) {
		t.Fatalf("got %v, want %v", err, ErrNilSchema)
	}
}

// Properties beside oneOf/anyOf apply to every variant, so the union IR keeps
// them for emitters that combine them with the variants. Go renders unions as
// raw JSON and never reads them, so their diagnostics are not reported.
func TestUnionKeepsSiblingPropertiesWithoutDiagnostics(t *testing.T) {
	proxy := schemaProxyFromYAML(t, `type: object
required: [token]
properties:
  token:
    type: string
    maxLength: 12
oneOf:
  - properties:
      mode:
        enum: [a]
  - properties:
      mode:
        enum: [b]
`)
	schemas := orderedmap.New[string, *highbase.SchemaProxy]()
	schemas.Set("Sample", proxy)
	irs, diagnostics, err := NewGenerator().SchemaIRs(schemas)
	if err != nil {
		t.Fatal(err)
	}
	ir := irs[0]
	if ir.Kind != KindUnion || ir.Properties == nil {
		t.Fatalf("got kind %v with properties %v", ir.Kind, ir.Properties)
	}
	if token, ok := ir.Properties.Get("token"); !ok || token.Kind != KindString || !isRequired(ir, "token") {
		t.Fatalf("sibling property not kept: %+v", ir.Properties)
	}
	for _, d := range diagnostics {
		if strings.HasPrefix(d.Path, "Sample.token") {
			t.Fatalf("diagnostic reported for an unrendered sibling property: %+v", d)
		}
	}
}

func TestIsNullOnly(t *testing.T) {
	for yml, want := range map[string]bool{
		"type: 'null'":           true,
		"type: string":           false,
		"enum: [null]":           true,
		"const: null":            true,
		"type: [string, 'null']": false,
	} {
		schemas := orderedmap.New[string, *highbase.SchemaProxy]()
		schemas.Set("Sample", schemaProxyFromYAML(t, yml))
		irs, _, err := NewGenerator().SchemaIRs(schemas)
		if err != nil {
			t.Fatal(err)
		}
		if got := IsNullOnly(irs[0]); got != want {
			t.Errorf("%s: got %v, want %v", yml, got, want)
		}
	}
}
