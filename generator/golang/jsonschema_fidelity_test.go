// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"strings"
	"testing"

	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
)

func TestJSONSchema202012KeywordDiagnostics(t *testing.T) {
	source, err := RenderSchema("full fidelity", schemaProxyFromYAML(t, `
$schema: https://json-schema.org/draft/2020-12/schema
$id: https://example.com/schemas/full
$anchor: full
$dynamicAnchor: fullNode
$comment: generator diagnostics should report metadata
$vocabulary:
  https://json-schema.org/draft/2020-12/vocab/core: true
type: object
minProperties: 1
maxProperties: 8
required: [id]
if:
  properties:
    kind:
      const: business
then:
  required: [tax_id]
else:
  required: [ssn]
not:
  required: [forbidden]
properties:
  id:
    type: string
    const: fixed
  payload:
    type: string
    minLength: 1
    maxLength: 100
    pattern: "^[a-z]+$"
    contentEncoding: base64
    contentMediaType: application/json
    contentSchema:
      type: object
  amount:
    type: number
    multipleOf: 0.01
    minimum: 0
    exclusiveMaximum: 100
  tuple:
    type: array
    minItems: 1
    maxItems: 3
    uniqueItems: true
    prefixItems:
      - type: string
      - type: integer
    items: false
    contains:
      type: string
    minContains: 1
    maxContains: 2
    unevaluatedItems:
      type: boolean
  object_rules:
    type: object
    minProperties: 1
    maxProperties: 4
    propertyNames:
      pattern: "^[a-z_]+$"
    dependentSchemas:
      card:
        type: object
    dependentRequired:
      card: [billing]
    patternProperties:
      "^x-":
        type: string
    additionalProperties: false
    unevaluatedProperties: false
  dynamic:
    $dynamicRef: '#/components/schemas/Meta'
`))
	if err != nil {
		t.Fatal(err)
	}
	if len(source) == 0 {
		t.Fatal("expected generated source")
	}

	file, err := NewGenerator().RenderSchemas(singleSchemaMap(t, "full fidelity", schemaProxyFromYAML(t, `
$schema: https://json-schema.org/draft/2020-12/schema
$id: https://example.com/schemas/full
$anchor: full
$dynamicAnchor: fullNode
$comment: generator diagnostics should report metadata
$vocabulary:
  https://json-schema.org/draft/2020-12/vocab/core: true
type: object
minProperties: 1
maxProperties: 8
required: [id]
if:
  properties:
    kind:
      const: business
then:
  required: [tax_id]
else:
  required: [ssn]
not:
  required: [forbidden]
properties:
  id:
    type: string
    const: fixed
  payload:
    type: string
    minLength: 1
    maxLength: 100
    pattern: "^[a-z]+$"
    contentEncoding: base64
    contentMediaType: application/json
    contentSchema:
      type: object
  amount:
    type: number
    multipleOf: 0.01
    minimum: 0
    exclusiveMaximum: 100
  tuple:
    type: array
    minItems: 1
    maxItems: 3
    uniqueItems: true
    prefixItems:
      - type: string
      - type: integer
    items: false
    contains:
      type: string
    minContains: 1
    maxContains: 2
    unevaluatedItems:
      type: boolean
  object_rules:
    type: object
    minProperties: 1
    maxProperties: 4
    propertyNames:
      pattern: "^[a-z_]+$"
    dependentSchemas:
      card:
        type: object
    dependentRequired:
      card: [billing]
    patternProperties:
      "^x-":
        type: string
    additionalProperties: false
    unevaluatedProperties: false
  dynamic:
    $dynamicRef: '#/components/schemas/Meta'
`)))
	if err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{
		DiagnosticAdditionalPropertiesFalse,
		DiagnosticArrayContains,
		DiagnosticBooleanItems,
		DiagnosticConditionalSchema,
		DiagnosticConstKeyword,
		DiagnosticContentSchema,
		DiagnosticDependentRequired,
		DiagnosticDependentSchemas,
		DiagnosticDynamicReference,
		DiagnosticNotSchema,
		DiagnosticPatternProperties,
		DiagnosticPrefixItems,
		DiagnosticPropertyNames,
		DiagnosticSchemaMetadata,
		DiagnosticUnevaluatedItems,
		DiagnosticUnevaluatedProperties,
		DiagnosticValidationKeyword,
	} {
		if !hasDiagnosticCode(file.Diagnostics, code) {
			t.Fatalf("missing diagnostic code %s: %#v", code, file.Diagnostics)
		}
	}
}

func TestJSONSchema202012MultiTypeRawUnion(t *testing.T) {
	source, err := RenderSchema("multi type", schemaProxyFromYAML(t, `
type: object
properties:
  value:
    type: [string, integer, "null"]
`))
	if err != nil {
		t.Fatal(err)
	}
	src := strings.Join(strings.Fields(string(source)), " ")
	assertContains(t, src, "Value *MultiType_ValueUnion `json:\"value,omitempty\"`")
	assertContains(t, string(source), "type MultiType_ValueUnion struct")
	assertContains(t, string(source), "Raw json.RawMessage")
	assertParsesAndCompiles(t, source)
}

func TestJSONSchema202012NullOnlyUnionVariantsCollapse(t *testing.T) {
	gen := NewGenerator()
	ir, err := gen.irFromOpenAPI("null union", schemaProxyFromYAML(t, `
type: object
properties:
  const_null:
    anyOf:
      - const: null
      - type: string
  enum_null:
    anyOf:
      - enum: [null]
      - type: integer
  nullable_any:
    anyOf:
      - nullable: true
      - type: string
`), "null union")
	if err != nil {
		t.Fatal(err)
	}
	constNull := ir.Properties.GetOrZero("const_null")
	if constNull == nil || constNull.Kind != KindString || !constNull.Nullable {
		t.Fatalf("const:null anyOf variant should collapse to nullable string, got %#v", constNull)
	}
	enumNull := ir.Properties.GetOrZero("enum_null")
	if enumNull == nil || enumNull.Kind != KindInteger || !enumNull.Nullable {
		t.Fatalf("enum:[null] anyOf variant should collapse to nullable integer, got %#v", enumNull)
	}
	nullableAny := ir.Properties.GetOrZero("nullable_any")
	if nullableAny == nil || nullableAny.Kind != KindUnion {
		t.Fatalf("nullable unconstrained schema is not null-only and should remain a union, got %#v", nullableAny)
	}

	source, err := gen.renderFile([]*SchemaIR{ir})
	if err != nil {
		t.Fatal(err)
	}
	src := strings.Join(strings.Fields(string(source.Source)), " ")
	assertContains(t, src, "ConstNull *string `json:\"const_null,omitempty\"`")
	assertContains(t, src, "EnumNull *int `json:\"enum_null,omitempty\"`")
	assertContains(t, src, "NullableAny *NullUnion_NullableAnyUnion `json:\"nullable_any,omitempty\"`")
	assertParsesAndCompiles(t, source.Source)
}

func TestJSONSchema202012EnumScalarVariants(t *testing.T) {
	schemas := orderedmap.New[string, *highbase.SchemaProxy]()
	schemas.Set("string enum", schemaProxyFromYAML(t, `
type: string
enum: [open, closed]
`))
	schemas.Set("int enum", schemaProxyFromYAML(t, `
type: integer
enum: [1, 2]
`))
	schemas.Set("float enum", schemaProxyFromYAML(t, `
type: number
enum: [1.5, 2]
`))
	schemas.Set("bool enum", schemaProxyFromYAML(t, `
type: boolean
enum: [true, false]
`))
	schemas.Set("nullable enum", schemaProxyFromYAML(t, `
enum:
  - null
  - active
`))
	schemas.Set("mixed enum", schemaProxyFromYAML(t, `
enum:
  - active
  - 2
  - true
`))

	file, err := NewGenerator(WithEnumConstants(true)).RenderSchemas(schemas)
	if err != nil {
		t.Fatal(err)
	}
	src := strings.Join(strings.Fields(string(file.Source)), " ")
	assertContains(t, src, "type StringEnum string")
	assertContains(t, src, "StringEnumOpen StringEnum = \"open\"")
	assertContains(t, src, "type IntEnum int")
	assertContains(t, src, "IntEnumValue1 IntEnum = 1")
	assertContains(t, src, "type FloatEnum float64")
	assertContains(t, src, "FloatEnumValue15 FloatEnum = 1.5")
	assertContains(t, src, "type BoolEnum bool")
	assertContains(t, src, "BoolEnumTrue BoolEnum = true")
	assertContains(t, src, "type NullableEnum string")
	assertContains(t, src, "NullableEnumActive NullableEnum = \"active\"")
	assertContains(t, src, "type MixedEnum any")
	assertNotContains(t, src, "MixedEnumActive")
	if !hasDiagnosticCode(file.Diagnostics, DiagnosticNullEnum) {
		t.Fatalf("expected nullable enum diagnostic: %#v", file.Diagnostics)
	}
	if !hasDiagnosticCode(file.Diagnostics, DiagnosticMixedEnum) {
		t.Fatalf("expected mixed enum diagnostic: %#v", file.Diagnostics)
	}
	assertParsesAndCompiles(t, file.Source)
}

func TestJSONSchema202012ClosedNestedObjectUsesStruct(t *testing.T) {
	source, err := RenderSchema("closed parent", schemaProxyFromYAML(t, `
type: object
properties:
  config:
    type: object
    additionalProperties: false
`))
	if err != nil {
		t.Fatal(err)
	}
	src := strings.Join(strings.Fields(string(source)), " ")
	assertContains(t, src, "type ClosedParent_Config struct { }")
	assertContains(t, src, "Config *ClosedParent_Config `json:\"config,omitempty\"`")
	assertParsesAndCompiles(t, source)
}

func TestJSONSchema202012ImplicitTypeInference(t *testing.T) {
	source, err := RenderSchema("implicit", schemaProxyFromYAML(t, `
type: object
properties:
  name:
    minLength: 1
  tags:
    items:
      type: string
  loose_object:
    minProperties: 1
  ambiguous:
    minLength: 1
    minimum: 0
`))
	if err != nil {
		t.Fatal(err)
	}
	src := strings.Join(strings.Fields(string(source)), " ")
	assertContains(t, src, "Name *string `json:\"name,omitempty\"`")
	assertContains(t, src, "Tags []string `json:\"tags,omitempty\"`")
	assertContains(t, src, "LooseObject map[string]any `json:\"loose_object,omitempty\"`")
	assertContains(t, src, "Ambiguous any `json:\"ambiguous,omitempty\"`")
	assertParsesAndCompiles(t, source)
}

func TestJSONSchema202012DynamicRefRendersNamedReference(t *testing.T) {
	schemas := orderedmap.New[string, *highbase.SchemaProxy]()
	schemas.Set("Meta", highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"string"}}))
	schemas.Set("Holder", schemaProxyFromYAML(t, `
type: object
properties:
  meta:
    $dynamicRef: '#/components/schemas/Meta'
  nullable_meta:
    description: Nullable dynamic metadata.
    $dynamicRef: '#/components/schemas/Meta'
    nullable: true
`))
	file, err := NewGenerator().RenderSchemas(schemas)
	if err != nil {
		t.Fatal(err)
	}
	src := strings.Join(strings.Fields(string(file.Source)), " ")
	assertContains(t, src, "Meta *Meta `json:\"meta,omitempty\"`")
	if !hasDiagnosticCode(file.Diagnostics, DiagnosticDynamicReference) {
		t.Fatalf("expected dynamic reference diagnostic: %#v", file.Diagnostics)
	}
	assertParsesAndCompiles(t, file.Source)

	gen := NewGenerator()
	ir, err := gen.irFromOpenAPI("Holder", schemas.GetOrZero("Holder"), "Holder")
	if err != nil {
		t.Fatal(err)
	}
	nullableMeta := ir.Properties.GetOrZero("nullable_meta")
	if nullableMeta == nil || !nullableMeta.DynamicRef || !nullableMeta.Nullable {
		t.Fatalf("nullable dynamic ref should preserve both dynamic ref and nullability, got %#v", nullableMeta)
	}
	if nullableMeta.Description != "Nullable dynamic metadata." {
		t.Fatalf("nullable dynamic ref should preserve description, got %#v", nullableMeta)
	}
	rendered := gen.openapiFromIR(nullableMeta).Schema()
	if rendered == nil || len(rendered.AnyOf) != 2 || rendered.AnyOf[0].Schema().DynamicRef != "#/components/schemas/Meta" {
		t.Fatalf("nullable dynamic ref should render as anyOf dynamicRef/null, got %#v", rendered)
	}
	if rendered.AnyOf[0].Schema().Description != "Nullable dynamic metadata." {
		t.Fatalf("nullable dynamic ref should render description on dynamicRef variant, got %#v", rendered.AnyOf[0].Schema())
	}
}

func singleSchemaMap(t *testing.T, name string, schema *highbase.SchemaProxy) *orderedmap.Map[string, *highbase.SchemaProxy] {
	t.Helper()
	schemas := orderedmap.New[string, *highbase.SchemaProxy]()
	schemas.Set(name, schema)
	return schemas
}
