// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"bytes"
	"context"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

func TestTrainTravelDefaultRawUnion(t *testing.T) {
	file := renderTrainTravel(t)
	src := string(file.Source)

	assertContains(t, src, "type Station struct")
	assertContains(t, src, "type Trip struct")
	assertContains(t, src, "type Booking struct")
	assertContains(t, src, "type BookingPayment struct")
	assertContains(t, src, "Source")
	assertContains(t, src, "*BookingPayment_SourceUnion")
	assertContains(t, src, "`json:\"source,omitempty\"`")
	assertContains(t, src, "type BookingPayment_SourceUnion struct")
	assertContains(t, src, "Raw json.RawMessage")
	assertNotContains(t, src, "type BookingPayment_Source interface")
	if !hasDiagnosticCode(file.Diagnostics, DiagnosticOptionalConstDiscriminator) {
		t.Fatalf("expected optional discriminator diagnostic, got %#v", file.Diagnostics)
	}
	if !hasDiagnosticCode(file.Diagnostics, DiagnosticValidationKeyword) {
		t.Fatalf("expected validation keyword diagnostic, got %#v", file.Diagnostics)
	}
	assertParsesAndCompiles(t, file.Source)
}

func TestTrainTravelOptionalConstDiscriminatorTypedUnion(t *testing.T) {
	file := renderTrainTravel(t, WithOptionalConstDiscriminatorUnions(true))
	src := string(file.Source)

	assertContains(t, src, "type BookingPayment_Source interface")
	assertContains(t, src, "type BookingPayment_SourceUnion struct")
	assertContains(t, src, "Value BookingPayment_Source")
	assertContains(t, src, "case \"bank_account\":")
	assertContains(t, src, "case \"card\":")
	assertContains(t, src, "func (BookingPayment_Source_Card) isBookingPayment_Source() {}")
	assertContains(t, src, "func (BookingPayment_Source_BankAccount) isBookingPayment_Source() {}")
	if !hasDiagnosticCode(file.Diagnostics, DiagnosticUnevaluatedProperties) {
		t.Fatalf("expected unevaluatedProperties diagnostic, got %#v", file.Diagnostics)
	}
	assertParsesAndCompiles(t, file.Source)
}

func TestRenderSchemaConvenienceAndOptions(t *testing.T) {
	schema := schemaProxyFromYAML(t, `
type: object
required: [id]
properties:
  id:
    type: string
  enabled:
    type: boolean
`)
	src, err := RenderSchema("option probe", schema,
		WithPackageName("custommodels"),
		WithOptionalFieldsAsPointers(false),
		WithOmitEmpty(false),
		WithGenerateYAMLTags(true),
		WithGenerateJSONTags(false),
	)
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, string(src), "package custommodels")
	assertContains(t, string(src), "Enabled bool")
	assertContains(t, string(src), "`yaml:\"enabled\"`")
	assertNotContains(t, string(src), "omitempty")
}

func TestOpenAPICompositionAndUnionPolicies(t *testing.T) {
	tests := map[string]string{
		"raw oneOf": `
oneOf:
  - type: object
    properties:
      a: { type: string }
  - type: object
    properties:
      b: { type: string }
`,
		"raw anyOf": `
anyOf:
  - type: string
  - type: integer
`,
		"nullable union": `
oneOf:
  - type: string
  - type: 'null'
`,
		"allOf": `
allOf:
  - type: object
    required: [id]
    properties:
      id: { type: string }
  - type: object
    properties:
      name: { type: string }
`,
	}
	for name, yml := range tests {
		t.Run(name, func(t *testing.T) {
			src, err := RenderSchema("Sample", schemaProxyFromYAML(t, yml))
			if err != nil {
				t.Fatal(err)
			}
			assertParsesAndCompiles(t, src)
		})
	}
}

func TestTopLevelArrayInlineItemDeclarations(t *testing.T) {
	source, err := RenderSchema("pets", schemaProxyFromYAML(t, `
type: array
items:
  type: object
  required: [name]
  properties:
    name:
      type: string
`))
	if err != nil {
		t.Fatal(err)
	}
	src := strings.Join(strings.Fields(string(source)), " ")
	assertContains(t, src, "type Pets []Pets_Item")
	assertContains(t, src, "type Pets_Item struct")
	assertContains(t, src, "Name string `json:\"name\"`")
	assertParsesAndCompiles(t, source)
}

func TestTopLevelFreeFormObjectRendersMapAlias(t *testing.T) {
	for name, yml := range map[string]string{
		"payload":   "type: object\n",
		"free bool": "type: object\nadditionalProperties: true\n",
	} {
		t.Run(name, func(t *testing.T) {
			source, err := RenderSchema(name, schemaProxyFromYAML(t, yml))
			if err != nil {
				t.Fatal(err)
			}
			typeName := toPublicName(name)
			assertContains(t, strings.Join(strings.Fields(string(source)), " "), "type "+typeName+" map[string]any")
			assertParsesCompilesAndTests(t, source, "package models\n\n"+
				"import (\n"+
				"\t\"encoding/json\"\n"+
				"\t\"testing\"\n"+
				")\n\n"+
				"func TestFreeFormObjectRoundTrip(t *testing.T) {\n"+
				"\tvar model "+typeName+"\n"+
				"\tif err := json.Unmarshal([]byte(`{\"x\":7}`), &model); err != nil {\n"+
				"\t\tt.Fatal(err)\n"+
				"\t}\n"+
				"\tif model[\"x\"].(float64) != 7 {\n"+
				"\t\tt.Fatalf(\"unexpected model: %#v\", model)\n"+
				"\t}\n"+
				"\tout, err := json.Marshal(model)\n"+
				"\tif err != nil {\n"+
				"\t\tt.Fatal(err)\n"+
				"\t}\n"+
				"\tif string(out) != `{\"x\":7}` {\n"+
				"\t\tt.Fatalf(\"unexpected output: %s\", out)\n"+
				"\t}\n"+
				"}\n")
		})
	}
}

func TestNullableAllOfPreservesWrapperMetadata(t *testing.T) {
	source, err := RenderSchema("holder", schemaProxyFromYAML(t, `
type: object
required: [value]
properties:
  value:
    title: Nullable Value
    nullable: true
    allOf:
      - type: object
        required: [id]
        properties:
          id:
            type: string
`))
	if err != nil {
		t.Fatal(err)
	}
	src := strings.Join(strings.Fields(string(source)), " ")
	assertContains(t, src, "Value *Holder_Value `json:\"value\"`")
	assertContains(t, string(source), "// Holder_Value Nullable Value.")
	assertParsesAndCompiles(t, source)
}

func TestSchemaFromPointerRootPreservesNullability(t *testing.T) {
	for name, proxy := range map[string]*highbase.SchemaProxy{
		"type":  mustSchemaFromType(t, reflect.TypeOf((*string)(nil))),
		"value": mustSchemaFromValue(t, (*string)(nil)),
	} {
		schema := proxy.Schema()
		if schema == nil || !schemaTypeContains(schema.Type, "string") || !schemaTypeContains(schema.Type, "null") {
			t.Fatalf("%s pointer root should render as nullable string, got %#v", name, schema)
		}
	}

	for name, set := range map[string]*SchemaSet{
		"types":  mustSchemasFromTypes(t, reflect.TypeOf((*string)(nil))),
		"values": mustSchemasFromValues(t, (*string)(nil)),
	} {
		assertNullableStringSchema(t, name+" root", set.Root)
		assertNullableStringSchema(t, name+" roots entry", set.Roots.GetOrZero("String"))
	}

	type PointerRootModel struct {
		ID string `json:"id"`
	}
	for name, set := range map[string]*SchemaSet{
		"types":  mustSchemasFromTypes(t, reflect.TypeOf((*PointerRootModel)(nil))),
		"values": mustSchemasFromValues(t, (*PointerRootModel)(nil)),
	} {
		assertNullableRef(t, set.Root, "#/components/schemas/PointerRootModel")
		assertNullableRef(t, set.Roots.GetOrZero("PointerRootModel"), "#/components/schemas/PointerRootModel")
		component := componentSchema(t, set, "PointerRootModel")
		if schemaTypeContains(component.Type, "null") || component.Nullable != nil {
			t.Fatalf("%s component should stay non-nullable, got %#v", name, component)
		}
	}
}

func TestSchemaFromTypeReflection(t *testing.T) {
	type Embedded struct {
		TraceID string `json:"trace_id"`
	}
	type Meta map[string]string
	type Pet interface{ pet() }
	type Cat struct {
		Object string `json:"object"`
		Name   string `json:"name"`
	}
	type Sample struct {
		Embedded
		ID        string    `json:"id"`
		Name      *string   `json:"name,omitempty"`
		CreatedAt time.Time `json:"created_at"`
		Labels    []string  `json:"labels,omitempty"`
		Meta      Meta      `json:"meta,omitempty"`
		Ignored   string    `json:"-"`
		Choice    Pet       `json:"choice,omitempty"`
	}

	gen := NewGenerator(
		WithOneOfTypes((*Pet)(nil), Cat{}),
		WithDiscriminatorMapping((*Pet)(nil), "object", map[string]string{
			"cat": "#/components/schemas/Cat",
		}),
	)
	proxy, err := gen.SchemaFromType(reflect.TypeOf(Sample{}))
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := proxy.Render()
	if err != nil {
		t.Fatal(err)
	}
	text := string(rendered)
	assertContains(t, text, "trace_id:")
	assertContains(t, text, "created_at:")
	assertContains(t, text, "format: date-time")
	assertContains(t, text, "oneOf:")
	assertContains(t, text, "discriminator:")
	assertNotContains(t, text, "Ignored")
}

func TestSchemaFromTypeErrorsAndProvider(t *testing.T) {
	if _, err := SchemaFromValue(nil); err == nil {
		t.Fatal("expected nil value error")
	}
	if _, err := SchemaFromType(nil); err == nil {
		t.Fatal("expected nil type error")
	}
	type BadMap struct {
		Values map[int]string `json:"values"`
	}
	if _, err := SchemaFromType(reflect.TypeOf(BadMap{})); !strings.Contains(err.Error(), ErrUnsupportedMapKey.Error()) {
		t.Fatalf("expected map key error, got %v", err)
	}
	type NeedsRegistration interface{ marker() }
	type HasInterface struct {
		Value NeedsRegistration `json:"value"`
	}
	if _, err := SchemaFromType(reflect.TypeOf(HasInterface{})); !strings.Contains(err.Error(), ErrUnsupportedType.Error()) {
		t.Fatalf("expected unsupported interface error, got %v", err)
	}
	proxy, err := SchemaFromType(reflect.TypeOf(Provider{}))
	if err != nil {
		t.Fatal(err)
	}
	if proxy == nil {
		t.Fatal("expected proxy")
	}
	if _, err := SchemaFromType(reflect.TypeOf(BadProvider{})); err == nil {
		t.Fatal("expected bad provider schema error")
	}
}

func (Provider) OpenAPISchema() *highbase.SchemaProxy {
	return highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"string"}})
}

type Provider struct{}

func (BadProvider) OpenAPISchema() *highbase.SchemaProxy {
	return &highbase.SchemaProxy{}
}

type BadProvider struct{}

func TestHelpersAndErrors(t *testing.T) {
	if got := toPublicName("links-self"); got != "LinksSelf" {
		t.Fatalf("unexpected public name %q", got)
	}
	if got := toPublicName("trip_id"); got != "TripID" {
		t.Fatalf("unexpected initialism %q", got)
	}
	if got := toPublicName("123-value"); got != "Value123Value" {
		t.Fatalf("unexpected digit prefix %q", got)
	}
	if got := toPrivateName("HTTPServer"); got != "httpServer" {
		t.Fatalf("unexpected private name %q", got)
	}
	if got := refName("#/components/schemas/Pet"); got != "Pet" {
		t.Fatalf("unexpected ref name %q", got)
	}
	if got := NewGenerator().refTypeName("pet.yaml"); got != "PetYaml" {
		t.Fatalf("unexpected bare external ref name %q", got)
	}
	used := map[string]struct{}{}
	if uniqueName("Pet", used) != "Pet" || uniqueName("Pet", used) != "Pet__2" {
		t.Fatal("uniqueName did not allocate suffix")
	}
	if intString(0) != "0" || intString(42) != "42" {
		t.Fatal("intString failed")
	}
	if err := validatePackageName("type"); err == nil {
		t.Fatal("expected invalid package error")
	}
	if _, err := RenderSchema("Bad", nil); err == nil {
		t.Fatal("expected nil schema error")
	}
	if _, err := NewGenerator(WithPackageName("bad-name")).RenderSchemas(nil); err == nil {
		t.Fatal("expected invalid package error")
	}
}

func TestToOpenAPIPrimitiveAndRefPaths(t *testing.T) {
	gen := NewGenerator()
	values := []*SchemaIR{
		{Kind: KindRef, Ref: "#/components/schemas/Pet"},
		{Kind: KindString, Format: "uuid", Nullable: true},
		{Kind: KindInteger, Format: "int32"},
		{Kind: KindNumber, Format: "float"},
		{Kind: KindBoolean},
		{Kind: KindArray, Items: &SchemaIR{Kind: KindString}},
		{Kind: KindEnum, Enum: []*yaml.Node{stringNode("a")}},
		{Kind: KindUnknown},
		nil,
	}
	for _, value := range values {
		proxy := gen.openapiFromIR(value)
		if proxy == nil {
			t.Fatal("expected proxy")
		}
		if _, err := proxy.Render(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestExplicitDiscriminatorSchema(t *testing.T) {
	spec := []byte(`openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths: {}
components:
  schemas:
    Cat:
      type: object
      properties:
        kind:
          type: string
    Pet:
      discriminator:
        propertyName: kind
        mapping:
          cat: '#/components/schemas/Cat'
      oneOf:
        - $ref: '#/components/schemas/Cat'
`)
	doc, err := libopenapi.NewDocument(spec)
	if err != nil {
		t.Fatal(err)
	}
	model, err := doc.BuildV3Model()
	if err != nil {
		t.Fatal(err)
	}
	schema, ok := model.Model.Components.Schemas.Get("Pet")
	if !ok {
		t.Fatal("missing pet schema")
	}
	src, err := RenderSchema("Pet", schema)
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, string(src), "type Pet interface")
	assertContains(t, string(src), "case \"cat\":")
}

func TestImplicitDiscriminatorSchema(t *testing.T) {
	spec := []byte(`openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths: {}
components:
  schemas:
    cat:
      type: object
      required: [kind, name]
      properties:
        kind:
          type: string
        name:
          type: string
    dog:
      type: object
      required: [kind, bark]
      properties:
        kind:
          type: string
        bark:
          type: string
    Pet:
      discriminator:
        propertyName: kind
      oneOf:
        - $ref: '#/components/schemas/cat'
        - $ref: '#/components/schemas/dog'
`)
	doc, err := libopenapi.NewDocument(spec)
	if err != nil {
		t.Fatal(err)
	}
	model, err := doc.BuildV3Model()
	if err != nil {
		t.Fatal(err)
	}
	file, err := NewGenerator().RenderSchemas(model.Model.Components.Schemas)
	if err != nil {
		t.Fatal(err)
	}
	src := string(file.Source)
	assertContains(t, src, "case \"cat\":")
	assertContains(t, src, "case \"dog\":")
	assertParsesCompilesAndTests(t, file.Source, `package models

import (
	"encoding/json"
	"testing"
)

func TestImplicitDiscriminatorJSON(t *testing.T) {
	var pet PetUnion
	if err := json.Unmarshal([]byte("{\"kind\":\"cat\",\"name\":\"milo\"}"), &pet); err != nil {
		t.Fatal(err)
	}
	cat, ok := pet.Value.(Cat)
	if !ok || cat.Name != "milo" {
		t.Fatalf("unexpected pet value: %#v", pet.Value)
	}
}
	`)
}

func TestRenderSchemasTransformedSiblingRefComponentIsNotPureReference(t *testing.T) {
	spec := []byte(`openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths: {}
components:
  schemas:
    Base:
      type: string
    WithSibling:
      $ref: '#/components/schemas/Base'
      description: constrained base value
      enum: [fast, slow]
`)
	doc, err := libopenapi.NewDocument(spec)
	if err != nil {
		t.Fatal(err)
	}
	model, err := doc.BuildV3Model()
	if err != nil {
		t.Fatal(err)
	}
	withSibling := model.Model.Components.Schemas.GetOrZero("WithSibling")
	if withSibling == nil || !withSibling.IsReference() || !withSibling.IsTransformedRefWithSiblings() {
		t.Fatalf("expected transformed sibling ref component, got %#v", withSibling)
	}

	file, err := NewGenerator(WithEnumConstants(true)).RenderSchemas(model.Model.Components.Schemas)
	if err != nil {
		t.Fatal(err)
	}
	src := string(file.Source)
	assertContains(t, src, "type Base string")
	assertContains(t, src, "type WithSibling ")
	assertContains(t, src, "constrained base value")
	assertContains(t, src, `"fast"`)
	assertContains(t, src, `"slow"`)
	assertParsesAndCompiles(t, file.Source)
}

func TestIRFromOpenAPITransformedSiblingRefBuildError(t *testing.T) {
	proxy := malformedTransformedSiblingRefProxy(t)
	if !proxy.IsTransformedRefWithSiblings() {
		t.Fatal("expected transformed sibling ref")
	}

	_, err := NewGenerator().run().irFromOpenAPI("WithSibling", proxy, "WithSibling")
	if err == nil {
		t.Fatal("expected transformed sibling ref build error")
	}
	if !strings.Contains(err.Error(), "WithSibling") {
		t.Fatalf("expected error path to include component name, got %v", err)
	}
}

func malformedTransformedSiblingRefProxy(t *testing.T) *highbase.SchemaProxy {
	t.Helper()

	var node yaml.Node
	err := yaml.Unmarshal([]byte(`$ref: '#/components/schemas/Base'
dependentRequired:
  bad: nope
`), &node)
	if err != nil {
		t.Fatal(err)
	}

	cfg := index.CreateOpenAPIIndexConfig()
	cfg.SpecInfo = &datamodel.SpecInfo{VersionNumeric: 3.1}
	cfg.TransformSiblingRefs = true
	idx := index.NewSpecIndexWithConfig(node.Content[0], cfg)

	lowProxy := new(lowbase.SchemaProxy)
	if err := lowProxy.Build(context.Background(), nil, node.Content[0], idx); err != nil {
		t.Fatal(err)
	}
	return highbase.NewSchemaProxy(&low.NodeReference[*lowbase.SchemaProxy]{
		Value:     lowProxy,
		ValueNode: node.Content[0],
	})
}

func renderTrainTravel(t *testing.T, opts ...Option) *GeneratedFile {
	t.Helper()
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
	file, err := NewGenerator(opts...).RenderSchemas(model.Model.Components.Schemas)
	if err != nil {
		t.Fatal(err)
	}
	return file
}

func schemaProxyFromYAML(t *testing.T, yml string) *highbase.SchemaProxy {
	t.Helper()
	spec := []byte("openapi: 3.1.0\ninfo:\n  title: Test\n  version: 1.0.0\npaths: {}\ncomponents:\n  schemas:\n    Sample:\n" + indent(yml, "      "))
	doc, err := libopenapi.NewDocument(spec)
	if err != nil {
		t.Fatal(err)
	}
	model, err := doc.BuildV3Model()
	if err != nil {
		t.Fatal(err)
	}
	schema, ok := model.Model.Components.Schemas.Get("Sample")
	if !ok {
		t.Fatal("missing sample schema")
	}
	return schema
}

func indent(in, prefix string) string {
	lines := strings.Split(strings.TrimPrefix(in, "\n"), "\n")
	var b strings.Builder
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			b.WriteByte('\n')
			continue
		}
		b.WriteString(prefix)
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

func assertParsesAndCompiles(t *testing.T, src []byte) {
	t.Helper()
	assertParsesCompilesAndTests(t, src, "package models\n\nimport \"testing\"\n\nfunc TestGeneratedPackage(t *testing.T) {}\n")
}

func assertParsesCompilesAndTests(t *testing.T, src []byte, testSource string) {
	t.Helper()
	assertParsesCompilesAndTestsWithFiles(t, map[string][]byte{"models.go": src}, testSource)
}

func assertParsesCompilesAndTestsWithFiles(t *testing.T, files map[string][]byte, testSource string) {
	t.Helper()
	dir := t.TempDir()
	for name, src := range files {
		if !bytes.Equal(bytes.TrimSpace(src), bytes.TrimSpace(mustFormat(t, src))) {
			t.Fatalf("%s is not gofmt formatted", name)
		}
		if _, err := parser.ParseFile(token.NewFileSet(), name, src, parser.AllErrors); err != nil {
			t.Fatalf("generated source %s does not parse: %v\n%s", name, err, src)
		}
		if err := os.WriteFile(filepath.Join(dir, name), src, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "models_test.go"), []byte(testSource), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "test")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GO111MODULE=off")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated source does not compile: %v\n%s", err, out)
	}
}

func mustFormat(t *testing.T, src []byte) []byte {
	t.Helper()
	out, err := format.Source(src)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func mustSchemaFromType(t *testing.T, typ reflect.Type) *highbase.SchemaProxy {
	t.Helper()
	proxy, err := SchemaFromType(typ)
	if err != nil {
		t.Fatal(err)
	}
	return proxy
}

func mustSchemaFromValue(t *testing.T, value any) *highbase.SchemaProxy {
	t.Helper()
	proxy, err := SchemaFromValue(value)
	if err != nil {
		t.Fatal(err)
	}
	return proxy
}

func mustSchemasFromTypes(t *testing.T, types ...reflect.Type) *SchemaSet {
	t.Helper()
	set, err := SchemasFromTypes(types...)
	if err != nil {
		t.Fatal(err)
	}
	return set
}

func mustSchemasFromValues(t *testing.T, values ...any) *SchemaSet {
	t.Helper()
	set, err := SchemasFromValues(values...)
	if err != nil {
		t.Fatal(err)
	}
	return set
}

func assertNullableStringSchema(t *testing.T, name string, proxy *highbase.SchemaProxy) {
	t.Helper()
	schema := proxy.Schema()
	if schema == nil || !schemaTypeContains(schema.Type, "string") || !schemaTypeContains(schema.Type, "null") {
		t.Fatalf("%s should render as nullable string, got %#v", name, schema)
	}
}

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Fatalf("expected %q in:\n%s", substr, s)
	}
}

func assertNotContains(t *testing.T, s, substr string) {
	t.Helper()
	if strings.Contains(s, substr) {
		t.Fatalf("did not expect %q in:\n%s", substr, s)
	}
}

func TestManualRenderSchemasNilAndFormatMapping(t *testing.T) {
	file, err := NewGenerator(WithFormatMapping("date-time", "time.Time", "time")).RenderSchemas(orderedmap.New[string, *highbase.SchemaProxy]())
	if err != nil {
		t.Fatal(err)
	}
	if string(file.Source) != "package models\n" {
		t.Fatalf("unexpected empty file %q", file.Source)
	}
}
