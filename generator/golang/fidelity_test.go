// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"reflect"
	"strings"
	"testing"

	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
)

func TestAPIPolishSchemaSetRootsAndOptionVariants(t *testing.T) {
	set, err := SchemasFromTypesWithOptions([]reflect.Type{
		reflect.TypeOf(PhaseTwoCustomer{}),
		reflect.TypeOf(PhaseTwoAddress{}),
	},
		WithOneOfTypes((*PhaseTwoPaymentMethod)(nil), PhaseTwoCard{}, PhaseTwoBank{}),
		WithDiscriminatorMapping((*PhaseTwoPaymentMethod)(nil), "object", map[string]string{
			"bank": "#/components/schemas/PhaseTwoBank",
			"card": "#/components/schemas/PhaseTwoCard",
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if set.Roots.Len() != 2 {
		t.Fatalf("expected two roots, got %d", set.Roots.Len())
	}
	if root, ok := set.Roots.Get("PhaseTwoCustomer"); !ok || !root.IsReference() {
		t.Fatalf("customer root should be a component reference: %#v", root)
	}
	if root, ok := set.Roots.Get("PhaseTwoAddress"); !ok || !root.IsReference() {
		t.Fatalf("address root should be a component reference: %#v", root)
	}

	values, err := SchemasFromValuesWithOptions([]any{PhaseTwoCustomer{}},
		WithOneOfTypes((*PhaseTwoPaymentMethod)(nil), PhaseTwoCard{}, PhaseTwoBank{}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if values.Roots.Len() != 1 {
		t.Fatalf("expected one value root, got %d", values.Roots.Len())
	}

	primitive, err := SchemasFromTypes(reflect.TypeOf(""))
	if err != nil {
		t.Fatal(err)
	}
	if primitive.Root == nil || primitive.Root.IsReference() {
		t.Fatalf("primitive root should render inline, got %#v", primitive.Root)
	}
	if primitive.Components.Len() != 0 {
		t.Fatalf("primitive roots should not create components: %d", primitive.Components.Len())
	}
}

type GraphReviewPaymentMethod interface {
	graphReviewPaymentMethod()
}

type GraphReviewCard struct {
	Object string `json:"object"`
	CVC    string `json:"cvc"`
}

func (GraphReviewCard) graphReviewPaymentMethod() {}

type GraphReviewBank struct {
	Object string `json:"object"`
	IBAN   string `json:"iban"`
}

func (GraphReviewBank) graphReviewPaymentMethod() {}

type GraphReviewNode struct {
	ID      string                     `json:"id"`
	Parent  *GraphReviewNode           `json:"parent,omitempty"`
	Labels  map[string]string          `json:"labels,omitempty"`
	Payment GraphReviewPaymentMethod   `json:"payment,omitempty"`
	History []GraphReviewPaymentMethod `json:"history,omitempty"`
}

type CustomSchemaScalar string

type CustomSchemaModel struct {
	ID       CustomSchemaScalar  `json:"id"`
	ParentID *CustomSchemaScalar `json:"parent_id,omitempty"`
}

func TestPreMergeReflectedComponentGraphReview(t *testing.T) {
	set, err := SchemasFromTypesWithOptions([]reflect.Type{reflect.TypeOf(GraphReviewNode{})},
		WithOneOfTypes((*GraphReviewPaymentMethod)(nil), GraphReviewCard{}, GraphReviewBank{}),
		WithDiscriminatorMapping((*GraphReviewPaymentMethod)(nil), "object", map[string]string{
			"bank": "#/components/schemas/GraphReviewBank",
			"card": "#/components/schemas/GraphReviewCard",
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if set.Root.GetReference() != "#/components/schemas/GraphReviewNode" {
		t.Fatalf("unexpected root: %q", set.Root.GetReference())
	}
	for _, name := range []string{"GraphReviewBank", "GraphReviewCard", "GraphReviewNode", "GraphReviewNode_Labels", "GraphReviewNode_Payment"} {
		if _, ok := set.Components.Get(name); !ok {
			t.Fatalf("missing component %s", name)
		}
	}
	node := componentSchema(t, set, "GraphReviewNode")
	parent, ok := node.Properties.Get("parent")
	if !ok {
		t.Fatal("missing parent property")
	}
	assertNullableRef(t, parent, "#/components/schemas/GraphReviewNode")
	if schemaTypeContains(node.Type, "null") || node.Nullable != nil {
		t.Fatalf("node component should not be nullable from recursive pointer usage, got %#v", node)
	}
	labels, ok := node.Properties.Get("labels")
	if !ok || !labels.IsReference() || labels.GetReference() != "#/components/schemas/GraphReviewNode_Labels" {
		t.Fatalf("labels should be map component ref, got %#v", labels)
	}
	labelSchema := componentSchema(t, set, "GraphReviewNode_Labels")
	if labelSchema.AdditionalProperties == nil || !labelSchema.AdditionalProperties.IsA() {
		t.Fatalf("labels should be schema-valued additionalProperties, got %#v", labelSchema)
	}
	payment, ok := node.Properties.Get("payment")
	if !ok || !payment.IsReference() || payment.GetReference() != "#/components/schemas/GraphReviewNode_Payment" {
		t.Fatalf("payment should be union component ref, got %#v", payment)
	}
	paymentSchema := componentSchema(t, set, "GraphReviewNode_Payment")
	if len(paymentSchema.OneOf) != 2 || paymentSchema.Discriminator == nil {
		t.Fatalf("payment should be discriminated oneOf, got %#v", paymentSchema)
	}
	history, ok := node.Properties.Get("history")
	if !ok {
		t.Fatal("missing history property")
	}
	historySchema := history.Schema()
	if historySchema == nil || historySchema.Items == nil || !historySchema.Items.IsA() || !historySchema.Items.A.IsReference() {
		t.Fatalf("history should be an array of union refs, got %#v", historySchema)
	}
	if historySchema.Items.A.GetReference() != "#/components/schemas/GraphReviewNode_Payment" {
		t.Fatalf("history item should reuse payment union component, got %q", historySchema.Items.A.GetReference())
	}
}

func TestReflectionFidelityTypeSchemaOverride(t *testing.T) {
	customSchema := highbase.CreateSchemaProxy(&highbase.Schema{
		Type:   []string{"string"},
		Format: "custom-id",
	})
	set, err := SchemasFromTypesWithOptions(
		[]reflect.Type{reflect.TypeOf(CustomSchemaModel{})},
		WithTypeSchema(reflect.TypeOf(CustomSchemaScalar("")), customSchema),
	)
	if err != nil {
		t.Fatal(err)
	}
	model := componentSchema(t, set, "CustomSchemaModel")
	id, ok := model.Properties.Get("id")
	if !ok {
		t.Fatal("missing id property")
	}
	idSchema := id.Schema()
	if idSchema == nil || idSchema.Format != "custom-id" {
		t.Fatalf("id should use custom schema format, got %#v", idSchema)
	}
	parent, ok := model.Properties.Get("parent_id")
	if !ok {
		t.Fatal("missing parent_id property")
	}
	parentSchema := parent.Schema()
	if parentSchema == nil || parentSchema.Format != "custom-id" || !schemaTypeContains(parentSchema.Type, "null") || parentSchema.Nullable != nil {
		t.Fatalf("parent_id should use nullable custom schema format, got %#v", parentSchema)
	}

	badGen := NewGenerator(WithTypeSchema(reflect.TypeOf(CustomSchemaScalar("")), &highbase.SchemaProxy{}))
	if _, err := badGen.SchemaFromType(reflect.TypeOf(CustomSchemaScalar(""))); err == nil {
		t.Fatal("expected bad custom schema error")
	}
}

func TestModelFidelityEnumConstantsAndResolvers(t *testing.T) {
	schemas := orderedmap.New[string, *highbase.SchemaProxy]()
	schemas.Set("status", schemaProxyFromYAML(t, `
type: string
enum:
  - ""
  - in-progress
  - in progress
  - "200"
`))
	file, err := NewGenerator(
		WithEnumConstants(true),
		WithTypeNameResolver(func(name string) string {
			if name == "status" {
				return "PaymentStatus"
			}
			return ""
		}),
		WithEnumValueNameResolver(func(name string) string {
			if name == "200" {
				return "OK"
			}
			return ""
		}),
	).RenderSchemas(schemas)
	if err != nil {
		t.Fatal(err)
	}
	src := string(file.Source)
	compact := strings.Join(strings.Fields(src), " ")
	assertContains(t, src, "type PaymentStatus string")
	assertContains(t, compact, "PaymentStatusEmpty PaymentStatus = \"\"")
	assertContains(t, compact, "PaymentStatusInProgress PaymentStatus = \"in-progress\"")
	assertContains(t, compact, "PaymentStatusInProgress__2 PaymentStatus = \"in progress\"")
	assertContains(t, compact, "PaymentStatusOK PaymentStatus = \"200\"")
	assertParsesAndCompiles(t, file.Source)

	broadSource, err := RenderSchema("broad enum", schemaProxyFromYAML(t, `
type: string
enum:
  - custom-value
`), WithEnumConstants(true), WithNameResolver(func(name string) string {
		if name == "custom-value" {
			return "CustomBroad"
		}
		return ""
	}))
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, string(broadSource), "BroadEnumCustomBroad BroadEnum = \"custom-value\"")
}

func TestModelFidelityAdditionalPropertiesRoundTrip(t *testing.T) {
	schema := schemaProxyFromYAML(t, `
type: object
required: [id]
properties:
  id:
    type: string
additionalProperties:
  type: integer
`)
	source, err := RenderSchema("extra model", schema)
	if err != nil {
		t.Fatal(err)
	}
	src := string(source)
	assertContains(t, src, "func (m *ExtraModel) UnmarshalJSON")
	assertContains(t, src, "func (m ExtraModel) MarshalJSON")
	assertParsesCompilesAndTests(t, source, "package models\n\n"+
		"import (\n"+
		"\t\"encoding/json\"\n"+
		"\t\"strings\"\n"+
		"\t\"testing\"\n"+
		")\n\n"+
		"func TestAdditionalPropertiesRoundTrip(t *testing.T) {\n"+
		"\tvar model ExtraModel\n"+
		"\tif err := json.Unmarshal([]byte(`{\"id\":\"abc\",\"x\":7}`), &model); err != nil {\n"+
		"\t\tt.Fatal(err)\n"+
		"\t}\n"+
		"\tif model.ID != \"abc\" || model.AdditionalProperties[\"x\"] != 7 {\n"+
		"\t\tt.Fatalf(\"unexpected model: %#v\", model)\n"+
		"\t}\n"+
		"\tout, err := json.Marshal(ExtraModel{ID: \"def\", AdditionalProperties: map[string]int{\"x\": 9}})\n"+
		"\tif err != nil {\n"+
		"\t\tt.Fatal(err)\n"+
		"\t}\n"+
		"\ttext := string(out)\n"+
		"\tif !strings.Contains(text, \"\\\"id\\\":\\\"def\\\"\") || !strings.Contains(text, \"\\\"x\\\":9\") {\n"+
		"\t\tt.Fatalf(\"missing encoded fields: %s\", text)\n"+
		"\t}\n"+
		"}\n")

	collisionSource, err := RenderSchema("extra collision", schemaProxyFromYAML(t, `
type: object
properties:
  additional_properties:
    type: string
additionalProperties:
  type: string
`))
	if err != nil {
		t.Fatal(err)
	}
	collisionText := strings.Join(strings.Fields(string(collisionSource)), " ")
	assertContains(t, collisionText, "AdditionalProperties *string `json:\"additional_properties,omitempty\"`")
	assertContains(t, collisionText, "AdditionalProperties__2 map[string]string `json:\"-\"`")
}

func TestGeneratedCodeQualityAdditionalPropertiesMethodOption(t *testing.T) {
	schema := schemaProxyFromYAML(t, `
type: object
properties:
  id:
    type: string
additionalProperties:
  type: string
`)
	source, err := RenderSchema("extra model", schema, WithAdditionalPropertiesMethods(false))
	if err != nil {
		t.Fatal(err)
	}
	src := string(source)
	assertContains(t, src, "AdditionalProperties map[string]string `json:\"-\"`")
	assertNotContains(t, src, "func (m *ExtraModel) UnmarshalJSON")
	assertNotContains(t, src, "func (m ExtraModel) MarshalJSON")
	assertNotContains(t, src, "encoding/json")
	assertParsesAndCompiles(t, source)
}

func TestModelFidelityRecursiveAndExternalReferences(t *testing.T) {
	nodeProps := orderedmap.New[string, *highbase.SchemaProxy]()
	nodeProps.Set("value", highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"string"}}))
	nodeProps.Set("next", highbase.CreateSchemaProxyRef("#/components/schemas/Node"))
	ownerProps := orderedmap.New[string, *highbase.SchemaProxy]()
	ownerProps.Set("pet", highbase.CreateSchemaProxyRef("../common.yaml#/components/schemas/Pet"))
	schemas := orderedmap.New[string, *highbase.SchemaProxy]()
	schemas.Set("Node", highbase.CreateSchemaProxy(&highbase.Schema{
		Type:       []string{"object"},
		Properties: nodeProps,
	}))
	schemas.Set("ExternalOwner", highbase.CreateSchemaProxy(&highbase.Schema{
		Type:       []string{"object"},
		Properties: ownerProps,
	}))
	file, err := NewGenerator().RenderSchemas(schemas)
	if err != nil {
		t.Fatal(err)
	}
	src := string(file.Source)
	compact := strings.Join(strings.Fields(src), " ")
	assertContains(t, src, "type Node struct")
	assertContains(t, compact, "Next *Node `json:\"next,omitempty\"`")
	assertContains(t, compact, "Pet *Pet `json:\"pet,omitempty\"`")
	if !hasDiagnosticCode(file.Diagnostics, DiagnosticExternalReference) {
		t.Fatalf("expected external ref diagnostic, got %#v", file.Diagnostics)
	}

	nodeOnly := orderedmap.New[string, *highbase.SchemaProxy]()
	node, ok := schemas.Get("Node")
	if !ok {
		t.Fatal("missing Node")
	}
	nodeOnly.Set("Node", node)
	compiled, err := NewGenerator().RenderSchemas(nodeOnly)
	if err != nil {
		t.Fatal(err)
	}
	assertParsesAndCompiles(t, compiled.Source)
}

func TestModelFidelityExternalReferenceResolver(t *testing.T) {
	props := orderedmap.New[string, *highbase.SchemaProxy]()
	props.Set("pet", highbase.CreateSchemaProxyRef("../common.yaml#/components/schemas/Pet"))
	schemas := orderedmap.New[string, *highbase.SchemaProxy]()
	schemas.Set("ExternalOwner", highbase.CreateSchemaProxy(&highbase.Schema{
		Type:       []string{"object"},
		Properties: props,
	}))

	file, err := NewGenerator(WithExternalRefTypeResolver(func(ref string) string {
		if ref == "../common.yaml#/components/schemas/Pet" {
			return "SharedPet"
		}
		return ""
	})).RenderSchemas(schemas)
	if err != nil {
		t.Fatal(err)
	}
	src := strings.Join(strings.Fields(string(file.Source)), " ")
	assertContains(t, src, "Pet *SharedPet `json:\"pet,omitempty\"`")
	if !hasDiagnosticCode(file.Diagnostics, DiagnosticExternalReference) {
		t.Fatalf("expected external ref diagnostic, got %#v", file.Diagnostics)
	}
	assertContains(t, file.Diagnostics[0].Message, "SharedPet")
}

func TestModelFidelityNullableOptionalMatrix(t *testing.T) {
	schema := schemaProxyFromYAML(t, `
type: object
required: [required_plain, required_nullable]
properties:
  required_plain:
    type: string
  required_nullable:
    type: [string, "null"]
  optional_plain:
    type: string
  optional_nullable:
    type: [string, "null"]
`)
	source, err := RenderSchema("nullability", schema, WithOptionalFieldsAsPointers(false))
	if err != nil {
		t.Fatal(err)
	}
	src := string(source)
	compact := strings.Join(strings.Fields(src), " ")
	assertContains(t, compact, "RequiredPlain string `json:\"required_plain\"`")
	assertContains(t, compact, "RequiredNullable *string `json:\"required_nullable\"`")
	assertContains(t, compact, "OptionalPlain string `json:\"optional_plain,omitempty\"`")
	assertContains(t, compact, "OptionalNullable *string `json:\"optional_nullable,omitempty\"`")
	assertParsesAndCompiles(t, source)
}

func TestGeneratedCodeQualityFieldResolverAndAcronyms(t *testing.T) {
	schema := schemaProxyFromYAML(t, `
type: object
properties:
  cvc:
    type: string
  callback_url:
    type: string
  account_id:
    type: string
  custom:
    type: string
`)
	source, err := RenderSchema("naming", schema, WithFieldNameResolver(func(name string) string {
		if name == "custom" {
			return "Special"
		}
		return ""
	}))
	if err != nil {
		t.Fatal(err)
	}
	src := string(source)
	compact := strings.Join(strings.Fields(src), " ")
	assertContains(t, compact, "CVC *string `json:\"cvc,omitempty\"`")
	assertContains(t, compact, "CallbackURL *string `json:\"callback_url,omitempty\"`")
	assertContains(t, compact, "AccountID *string `json:\"account_id,omitempty\"`")
	assertContains(t, compact, "Special *string `json:\"custom,omitempty\"`")
	assertParsesAndCompiles(t, source)

	broadSource, err := RenderSchema("broad field", schemaProxyFromYAML(t, `
type: object
properties:
  broad:
    type: string
`), WithNameResolver(func(name string) string {
		if name == "broad" {
			return "BroadField"
		}
		return ""
	}))
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, string(broadSource), "BroadField *string")
}

func hasDiagnosticCode(diagnostics []Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func hasDiagnosticCodeOrMessage(diagnostics []Diagnostic, code, substr string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code || strings.Contains(diagnostic.Message, substr) || strings.Contains(diagnostic.Path, substr) {
			return true
		}
	}
	return false
}
