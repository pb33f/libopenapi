// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"bytes"
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
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
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
	assertContains(t, src, "*BookingPaymentSourceUnion")
	assertContains(t, src, "`json:\"source,omitempty\"`")
	assertContains(t, src, "type BookingPaymentSourceUnion struct")
	assertContains(t, src, "Raw json.RawMessage")
	assertNotContains(t, src, "type BookingPaymentSource interface")
	if len(file.Diagnostics) != 2 {
		t.Fatalf("expected two diagnostics, got %d: %#v", len(file.Diagnostics), file.Diagnostics)
	}
	assertContains(t, file.Diagnostics[0].Message+file.Diagnostics[1].Message, "optional")
	assertParsesAndCompiles(t, file.Source)
}

func TestTrainTravelOptionalConstDiscriminatorTypedUnion(t *testing.T) {
	file := renderTrainTravel(t, WithOptionalConstDiscriminatorUnions(true))
	src := string(file.Source)

	assertContains(t, src, "type BookingPaymentSource interface")
	assertContains(t, src, "type BookingPaymentSourceUnion struct")
	assertContains(t, src, "Value BookingPaymentSource")
	assertContains(t, src, "case \"bank_account\":")
	assertContains(t, src, "case \"card\":")
	assertContains(t, src, "func (BookingPaymentSourceCard) isBookingPaymentSource() {}")
	assertContains(t, src, "func (BookingPaymentSourceBankAccount) isBookingPaymentSource() {}")
	if len(file.Diagnostics) != 1 {
		t.Fatalf("expected one diagnostic, got %#v", file.Diagnostics)
	}
	assertContains(t, file.Diagnostics[0].Message, "unevaluatedProperties")
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
}

func (Provider) OpenAPISchema() *highbase.SchemaProxy {
	return highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"string"}})
}

type Provider struct{}

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
	used := map[string]struct{}{}
	if uniqueName("Pet", used) != "Pet" || uniqueName("Pet", used) != "Pet2" {
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

func renderTrainTravel(t *testing.T, opts ...Option) *File {
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
	if !bytes.Equal(bytes.TrimSpace(src), bytes.TrimSpace(mustFormat(t, src))) {
		t.Fatal("source is not gofmt formatted")
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "models.go", src, parser.AllErrors); err != nil {
		t.Fatalf("generated source does not parse: %v\n%s", err, src)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "models.go"), src, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "models_test.go"), []byte("package models\n\nimport \"testing\"\n\nfunc TestGeneratedPackage(t *testing.T) {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "test")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GO111MODULE=off")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated source does not compile: %v\n%s\n%s", err, out, src)
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
