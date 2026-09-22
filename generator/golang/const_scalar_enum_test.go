// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
	"go.yaml.in/yaml/v4"
)

// renderConstEnumSpec renders a spec whose components.schemas hold the given
// declarations and returns the generated Go source. It exercises the real
// document -> model -> generator pipeline (populateUnion + fold path) through
// the public RenderSchemas API, so these are true render regressions, not
// unit tests of the guard helper.
func renderConstEnumSpec(t *testing.T, body string) string {
	t.Helper()
	spec := "openapi: 3.1.0\ninfo:\n  title: const enum regression\n  version: \"1\"\n" +
		"components:\n  schemas:\n" + body
	doc, err := libopenapi.NewDocument([]byte(spec))
	if err != nil {
		t.Fatalf("parse spec: %v", err)
	}
	model, err := doc.BuildV3Model()
	if err != nil {
		t.Fatalf("build model: %v", err)
	}
	file, err := NewGenerator(WithEnumConstants(true)).RenderSchemas(model.Model.Components.Schemas)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	return string(file.Source)
}

// TestRenderUntypedObjectConstStaysUnion is the P1 regression: an object
// constant written without any "type" keyword must NOT be folded into a
// KindEnum. Before the scalar-node guard it collapsed to "type ObjectConst
// string" with empty constants, which cannot unmarshal object JSON.
func TestRenderUntypedObjectConstStaysUnion(t *testing.T) {
	src := renderConstEnumSpec(t, `    ObjectConst:
      oneOf:
        - const:
            kind: a
        - const:
            kind: b
`)
	if strings.Contains(src, "type ObjectConst string") {
		t.Fatalf("untyped object const was wrongly folded into a string enum:\n%s", src)
	}
	if !strings.Contains(src, "type ObjectConstUnion struct") || !strings.Contains(src, "Raw json.RawMessage") {
		t.Fatalf("untyped object const should remain a raw union:\n%s", src)
	}
}

// TestRenderUntypedArrayConstStaysUnion is the P1 regression for sequences:
// an array constant written without any "type" keyword must NOT be folded
// into a KindEnum either.
func TestRenderUntypedArrayConstStaysUnion(t *testing.T) {
	src := renderConstEnumSpec(t, `    ArrayConst:
      oneOf:
        - const: [1, 2]
        - const: [3, 4]
`)
	if strings.Contains(src, "type ArrayConst string") {
		t.Fatalf("untyped array const was wrongly folded into a string enum:\n%s", src)
	}
	if !strings.Contains(src, "type ArrayConstUnion struct") || !strings.Contains(src, "Raw json.RawMessage") {
		t.Fatalf("untyped array const should remain a raw union:\n%s", src)
	}
}

// TestRenderScalarConstFoldsToEnum is the positive control: the scalar const
// oneOf that the feature targets must still fold into a typed enum with
// constants, so the scalar guard does not regress the intended behaviour.
func TestRenderScalarConstFoldsToEnum(t *testing.T) {
	src := renderConstEnumSpec(t, `    PetStatus:
      oneOf:
        - const: available
          title: Available
        - const: pending
        - const: sold
`)
	if !strings.Contains(src, "type PetStatus string") {
		t.Fatalf("scalar const oneOf should fold into a string enum:\n%s", src)
	}
	// gofmt may pad the "=" column, so match on the value side only.
	for _, want := range []string{`= "available"`, `= "pending"`, `= "sold"`} {
		if !strings.Contains(src, want) {
			t.Fatalf("missing folded constant %q in:\n%s", want, src)
		}
	}
	if !strings.Contains(src, "PetStatusAvailable") ||
		!strings.Contains(src, "PetStatusPending") ||
		!strings.Contains(src, "PetStatusSold") {
		t.Fatalf("missing folded constant names:\n%s", src)
	}
}

// TestConstScalarEnumFromVariantsRejectsUntypedStructuralConsts pins the guard
// at the unit level: an untyped mapping or sequence const (a yaml node that is
// not a scalar) must veto the fold even though the schema declares no "type"
// and no structural keyword. This is the branch the render regressions above
// exercise end-to-end.
func TestConstScalarEnumFromVariantsRejectsUntypedStructuralConsts(t *testing.T) {
	mapping := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	sequence := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	cases := map[string]struct {
		variants []*SchemaIR
	}{
		"untyped mapping const": {
			variants: []*SchemaIR{{Const: mapping}, {Const: mapping}},
		},
		"untyped sequence const": {
			variants: []*SchemaIR{{Const: sequence}, {Const: sequence}},
		},
		"mapping const with const sibling": {
			variants: []*SchemaIR{{Const: stringNode("a")}, {Const: mapping}},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if values, _, ok := constScalarEnumFromVariants(tc.variants); ok {
				t.Fatalf("ok = true (%d values), want reject: %v", len(values), tc.variants)
			}
		})
	}
}
