// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"strings"
	"testing"

	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
)

func TestRoundTripOpenAPIIRPreservesJSONSchemaFidelity(t *testing.T) {
	gen := NewGenerator()
	ir, err := gen.irFromOpenAPI("round trip", schemaProxyFromYAML(t, `
$schema: https://json-schema.org/draft/2020-12/schema
$id: https://example.com/schemas/round-trip
$anchor: root
$dynamicAnchor: node
$comment: retained metadata
title: Round Trip Root
type: object
minProperties: 1
maxProperties: 5
unevaluatedProperties:
  type: string
properties:
  value:
    type: [string, integer, "null"]
  tuple:
    type: array
    prefixItems:
      - type: string
    items: false
    contains:
      type: string
    minContains: 1
  dynamic:
    $dynamicRef: '#/components/schemas/Node'
  encoded:
    type: string
    contentEncoding: base64
    contentMediaType: application/json
    contentSchema:
      type: object
`), "round trip")
	if err != nil {
		t.Fatal(err)
	}

	roundTripped := gen.openapiFromIR(ir).Schema()
	if roundTripped == nil {
		t.Fatal("expected round-tripped schema")
	}
	if roundTripped.SchemaTypeRef == "" || roundTripped.Id == "" || roundTripped.Anchor == "" || roundTripped.DynamicAnchor == "" || roundTripped.Comment == "" || roundTripped.Title != "Round Trip Root" {
		t.Fatalf("metadata was not preserved: %#v", roundTripped)
	}
	if roundTripped.MinProperties == nil || *roundTripped.MinProperties != 1 || roundTripped.MaxProperties == nil || *roundTripped.MaxProperties != 5 {
		t.Fatalf("object validation keywords were not preserved: %#v", roundTripped)
	}
	if roundTripped.UnevaluatedProperties == nil || !roundTripped.UnevaluatedProperties.IsA() {
		t.Fatalf("schema-valued unevaluatedProperties was not preserved: %#v", roundTripped.UnevaluatedProperties)
	}

	value := roundTripped.Properties.GetOrZero("value").Schema()
	if got := strings.Join(value.Type, ","); got != "string,integer,null" {
		t.Fatalf("multi-type schema was not preserved, got %q", got)
	}
	tuple := roundTripped.Properties.GetOrZero("tuple").Schema()
	if tuple.Items == nil || !tuple.Items.IsB() || tuple.Items.B {
		t.Fatalf("items:false was not preserved: %#v", tuple.Items)
	}
	if tuple.Contains == nil || tuple.MinContains == nil || *tuple.MinContains != 1 {
		t.Fatalf("contains keywords were not preserved: %#v", tuple)
	}
	dynamic := roundTripped.Properties.GetOrZero("dynamic").Schema()
	if dynamic.DynamicRef != "#/components/schemas/Node" {
		t.Fatalf("dynamic ref was not preserved: %#v", dynamic)
	}
	encoded := roundTripped.Properties.GetOrZero("encoded").Schema()
	if encoded.ContentEncoding != "base64" || encoded.ContentMediaType != "application/json" || encoded.ContentSchema == nil {
		t.Fatalf("content keywords were not preserved: %#v", encoded)
	}
}

func TestGeneratedBehaviorDiscriminatedUnionJSON(t *testing.T) {
	catProperties := orderedmap.New[string, *highbase.SchemaProxy]()
	catProperties.Set("kind", highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"string"}}))
	catProperties.Set("name", highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"string"}}))
	dogProperties := orderedmap.New[string, *highbase.SchemaProxy]()
	dogProperties.Set("kind", highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"string"}}))
	dogProperties.Set("bark", highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"string"}}))
	mapping := orderedmap.New[string, string]()
	mapping.Set("cat", "#/components/schemas/Cat")
	mapping.Set("dog", "#/components/schemas/Dog")
	schemas := orderedmap.New[string, *highbase.SchemaProxy]()
	schemas.Set("Cat", highbase.CreateSchemaProxy(&highbase.Schema{
		Type:       []string{"object"},
		Required:   []string{"kind", "name"},
		Properties: catProperties,
	}))
	schemas.Set("Dog", highbase.CreateSchemaProxy(&highbase.Schema{
		Type:       []string{"object"},
		Required:   []string{"kind", "bark"},
		Properties: dogProperties,
	}))
	schemas.Set("Pet", highbase.CreateSchemaProxy(&highbase.Schema{
		OneOf: []*highbase.SchemaProxy{
			highbase.CreateSchemaProxyRef("#/components/schemas/Cat"),
			highbase.CreateSchemaProxyRef("#/components/schemas/Dog"),
		},
		Discriminator: &highbase.Discriminator{
			PropertyName: "kind",
			Mapping:      mapping,
		},
	}))
	holderProperties := orderedmap.New[string, *highbase.SchemaProxy]()
	holderProperties.Set("pet", highbase.CreateSchemaProxyRef("#/components/schemas/Pet"))
	schemas.Set("Holder", highbase.CreateSchemaProxy(&highbase.Schema{
		Type:       []string{"object"},
		Required:   []string{"pet"},
		Properties: holderProperties,
	}))
	file, err := NewGenerator().RenderSchemas(schemas)
	if err != nil {
		t.Fatal(err)
	}
	assertParsesCompilesAndTests(t, file.Source, `package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDiscriminatedUnionJSON(t *testing.T) {
	var pet PetUnion
	if err := json.Unmarshal([]byte("{\"kind\":\"cat\",\"name\":\"milo\"}"), &pet); err != nil {
		t.Fatal(err)
	}
	cat, ok := pet.Value.(Cat)
	if !ok || cat.Kind != "cat" || cat.Name != "milo" {
		t.Fatalf("unexpected cat value: %#v", pet.Value)
	}
	out, err := json.Marshal(pet)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "\"kind\":\"cat\"") || !strings.Contains(string(out), "\"name\":\"milo\"") {
		t.Fatalf("unexpected marshal output: %s", out)
	}
	if err := json.Unmarshal([]byte("{\"kind\":\"lizard\"}"), &pet); err == nil {
		t.Fatal("expected unknown discriminator error")
	}
	var holder Holder
	if err := json.Unmarshal([]byte("{\"pet\":{\"kind\":\"dog\",\"bark\":\"woof\"}}"), &holder); err != nil {
		t.Fatal(err)
	}
	dog, ok := holder.Pet.Value.(Dog)
	if !ok || dog.Bark != "woof" {
		t.Fatalf("ref to union field did not decode through union wrapper: %#v", holder.Pet.Value)
	}
}
`)
}

func TestGeneratedBehaviorRawUnionAndNullableEnumJSON(t *testing.T) {
	source, err := RenderSchema("union holder", schemaProxyFromYAML(t, `
type: object
required: [status]
properties:
  value:
    type: [string, integer]
  status:
    enum:
      - null
      - active
`))
	if err != nil {
		t.Fatal(err)
	}
	assertParsesCompilesAndTests(t, source, `package models

import (
	"encoding/json"
	"testing"
)

func TestRawUnionAndNullableEnumJSON(t *testing.T) {
	var holder UnionHolder
	if err := json.Unmarshal([]byte("{\"value\":123,\"status\":\"active\"}"), &holder); err != nil {
		t.Fatal(err)
	}
	if holder.Value == nil || string(holder.Value.Bytes()) != "123" {
		t.Fatalf("raw union did not capture bytes: %#v", holder.Value)
	}
	copied := holder.Value.Bytes()
	copied[0] = '9'
	if string(holder.Value.Bytes()) != "123" {
		t.Fatal("raw union Bytes should return a copy")
	}
	if holder.Status == nil || *holder.Status != UnionHolder_Status("active") {
		t.Fatalf("nullable enum did not decode active value: %#v", holder.Status)
	}
	out, err := json.Marshal(UnionHolder{Value: &UnionHolder_ValueUnion{Raw: json.RawMessage("\"abc\"")}})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "{\"value\":\"abc\",\"status\":null}" {
		t.Fatalf("unexpected raw union marshal: %s", out)
	}
	var empty UnionHolder_ValueUnion
	if !empty.IsZero() {
		t.Fatal("zero raw union should report IsZero")
	}
	out, err = json.Marshal(empty)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "null" {
		t.Fatalf("zero raw union should marshal null, got %s", out)
	}
	if err := json.Unmarshal([]byte("{\"status\":null}"), &holder); err != nil {
		t.Fatal(err)
	}
	if holder.Status != nil {
		t.Fatalf("nullable enum should decode null to nil, got %#v", holder.Status)
	}
}
`)
}
