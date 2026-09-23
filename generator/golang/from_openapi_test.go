// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"testing"

	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

// TestConstScalarEnumFromVariantsGuards exercises every rejection branch of
// constScalarEnumFromVariants so that a oneOf/anyOf is only ever folded into a
// KindEnum when each non-null variant is a bare scalar const. Any structural
// marker (a declared object/array type, a $dynamicRef, a nested composition
// keyword, properties/items, or a nil / null-only variant list) must veto the
// fold.
func TestConstScalarEnumFromVariantsGuards(t *testing.T) {
	schemaProxy := func(s *highbase.Schema) *highbase.SchemaProxy {
		return highbase.CreateSchemaProxy(s)
	}
	propMap := func() *orderedmap.Map[string, *highbase.SchemaProxy] {
		m := orderedmap.New[string, *highbase.SchemaProxy]()
		m.Set("p", schemaProxy(&highbase.Schema{Type: []string{"string"}}))
		return m
	}

	cases := map[string]struct {
		variants   []*SchemaIR
		wantOK     bool
		wantNull   bool
		wantValues int
	}{
		"empty variant list": {
			variants: nil,
			wantOK:   false,
		},
		"nil variant member": {
			variants: []*SchemaIR{nil},
			wantOK:   false,
		},
		"variant without const": {
			variants: []*SchemaIR{{}},
			wantOK:   false,
		},
		"declared object array type": {
			variants: []*SchemaIR{
				{Const: stringNode("a"), SourceSchema: &highbase.Schema{Type: []string{"object"}}},
				{Const: stringNode("b"), SourceSchema: &highbase.Schema{Type: []string{"array"}}},
			},
			wantOK: false,
		},
		"dynamic ref variant": {
			variants: []*SchemaIR{
				{Const: stringNode("a"), SourceSchema: &highbase.Schema{DynamicRef: "#/components/schemas/X"}},
			},
			wantOK: false,
		},
		"nested composition keyword variant": {
			variants: []*SchemaIR{
				{Const: stringNode("a"), SourceSchema: &highbase.Schema{AllOf: []*highbase.SchemaProxy{schemaProxy(&highbase.Schema{})}}},
			},
			wantOK: false,
		},
		"explicit enum keyword variant": {
			variants: []*SchemaIR{
				{Const: stringNode("a"), SourceSchema: &highbase.Schema{Enum: []*yaml.Node{stringNode("x")}}},
			},
			wantOK: false,
		},
		"properties variant": {
			variants: []*SchemaIR{
				{Const: stringNode("a"), SourceSchema: &highbase.Schema{Properties: propMap()}},
			},
			wantOK: false,
		},
		"pattern properties variant": {
			variants: []*SchemaIR{
				{Const: stringNode("a"), SourceSchema: &highbase.Schema{PatternProperties: propMap()}},
			},
			wantOK: false,
		},
		"items variant": {
			variants: []*SchemaIR{
				{Const: stringNode("a"), SourceSchema: &highbase.Schema{Items: &highbase.DynamicValue[*highbase.SchemaProxy, bool]{A: schemaProxy(&highbase.Schema{})}}},
			},
			wantOK: false,
		},
		"prefix items variant": {
			variants: []*SchemaIR{
				{Const: stringNode("a"), SourceSchema: &highbase.Schema{PrefixItems: []*highbase.SchemaProxy{schemaProxy(&highbase.Schema{})}}},
			},
			wantOK: false,
		},
		"null-only variant list": {
			variants: []*SchemaIR{{Const: nullNode()}},
			wantOK:   false,
		},
		"scalar const plus null": {
			variants:   []*SchemaIR{{Const: nullNode()}, {Const: stringNode("a")}},
			wantOK:     true,
			wantNull:   true,
			wantValues: 1,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			values, nullable, ok := constScalarEnumFromVariants(tc.variants)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if !ok {
				return
			}
			if nullable != tc.wantNull {
				t.Fatalf("nullable = %v, want %v", nullable, tc.wantNull)
			}
			if len(values) != tc.wantValues {
				t.Fatalf("len(values) = %d, want %d", len(values), tc.wantValues)
			}
		})
	}
}