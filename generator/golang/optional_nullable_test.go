// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"strings"
	"testing"

	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
)

func TestOptionalNullableDoublePointer(t *testing.T) {
	properties := orderedmap.New[string, *highbase.SchemaProxy]()
	properties.Set("value", highbase.CreateSchemaProxy(&highbase.Schema{
		Type:     []string{"string", "null"},
		Nullable: boolPointer(true),
	}))
	schema := highbase.CreateSchemaProxy(&highbase.Schema{
		Type:       []string{"object"},
		Properties: properties,
	})

	source, err := RenderSchema("Patch", schema, WithOptionalNullableAsDoublePointer(true))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(source), "Value **string `json:\"value,omitempty\"`") {
		t.Fatalf("optional nullable field did not preserve omitted/null/value states:\n%s", source)
	}
}

func TestOptionalNullableDoublePointerUsesOnePointerForCompoundValues(t *testing.T) {
	properties := orderedmap.New[string, *highbase.SchemaProxy]()
	properties.Set("value", highbase.CreateSchemaProxy(&highbase.Schema{
		Type:     []string{"array", "null"},
		Nullable: boolPointer(true),
		Items: &highbase.DynamicValue[*highbase.SchemaProxy, bool]{
			A: highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"string"}}),
		},
	}))
	schema := highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"object"}, Properties: properties})
	source, err := RenderSchema("Patch", schema, WithOptionalNullableAsDoublePointer(true))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(source), "Value *[]string `json:\"value,omitempty\"`") {
		t.Fatalf("optional nullable compound field did not use one pointer:\n%s", source)
	}
}

func TestPublicNameMatchesGeneratorNaming(t *testing.T) {
	if got := PublicName("account_id"); got != "AccountID" {
		t.Fatalf("PublicName(account_id) = %q", got)
	}
}

func boolPointer(value bool) *bool {
	return &value
}
