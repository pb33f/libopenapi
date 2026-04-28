// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"go.yaml.in/yaml/v4"
)

type ReflectConformanceID string

type ReflectConformanceStatus string

type ReflectConformanceEmbedded struct {
	TraceID string `json:"trace_id"`
}

type ReflectConformancePaymentMethod interface {
	reflectConformancePaymentMethod()
}

type ReflectConformanceCard struct {
	Object string `json:"object"`
	Number string `json:"number"`
	CVC    string `json:"cvc,omitempty"`
}

func (ReflectConformanceCard) reflectConformancePaymentMethod() {}

type ReflectConformanceBank struct {
	Object        string  `json:"object"`
	AccountNumber string  `json:"account_number"`
	BankName      *string `json:"bank_name,omitempty"`
}

func (ReflectConformanceBank) reflectConformancePaymentMethod() {}

type ReflectConformanceAddress struct {
	Line1 string  `json:"line1"`
	Line2 *string `json:"line2,omitempty"`
	City  string  `json:"city"`
}

type ReflectConformanceAttribute struct {
	Name  string `json:"name"`
	Value string `json:"value,omitempty"`
}

type ReflectConformanceRoot struct {
	*ReflectConformanceEmbedded `json:",omitempty"`
	ID                          ReflectConformanceID                   `json:"id"`
	Status                      ReflectConformanceStatus               `json:"status"`
	Active                      bool                                   `json:"active"`
	Count                       int                                    `json:"count,string,omitempty"`
	Limit                       int                                    `json:"limit,omitzero"`
	Nickname                    *string                                `json:"nickname,omitempty"`
	Data                        []byte                                 `json:"data,omitempty"`
	Raw                         json.RawMessage                        `json:"raw,omitempty"`
	Tags                        []string                               `json:"tags,omitempty"`
	Labels                      map[string]string                      `json:"labels,omitempty"`
	Attributes                  map[string]ReflectConformanceAttribute `json:"attributes,omitempty"`
	Address                     *ReflectConformanceAddress             `json:"address,omitempty"`
	Payment                     ReflectConformancePaymentMethod        `json:"payment,omitempty"`
	History                     []ReflectConformancePaymentMethod      `json:"history,omitempty"`
	Self                        *ReflectConformanceRoot                `json:"self,omitempty"`
	Skipped                     string                                 `json:"-"`
}

func TestReflectOpenAPIConformanceSchemaSet(t *testing.T) {
	set := renderReflectConformanceSet(t)
	if set.Root == nil || !set.Root.IsReference() || set.Root.GetReference() != "#/components/schemas/ReflectConformanceRoot" {
		t.Fatalf("unexpected root ref: %#v", set.Root)
	}
	if root, ok := set.Roots.Get("ReflectConformanceRoot"); !ok || !root.IsReference() {
		t.Fatalf("missing root entry: %#v", set.Roots)
	}
	assertComponentKeysSorted(t, set.Components)
	for _, name := range []string{
		"ReflectConformanceAddress",
		"ReflectConformanceAttribute",
		"ReflectConformanceBank",
		"ReflectConformanceCard",
		"ReflectConformanceEmbedded",
		"ReflectConformanceRoot",
		"ReflectConformanceRoot_Attributes",
		"ReflectConformanceRoot_Labels",
		"ReflectConformanceRoot_Payment",
		"ReflectConformanceStatus",
	} {
		if _, ok := set.Components.Get(name); !ok {
			t.Fatalf("missing component %s", name)
		}
	}
	root := componentSchema(t, set, "ReflectConformanceRoot")
	for _, required := range []string{"active", "id", "status"} {
		if !containsString(root.Required, required) {
			t.Fatalf("missing required field %q in %#v", required, root.Required)
		}
	}
	for _, optional := range []string{"trace_id", "count", "limit", "nickname", "payment"} {
		if containsString(root.Required, optional) {
			t.Fatalf("field %q should be optional in %#v", optional, root.Required)
		}
	}
	if _, ok := root.Properties.Get("Skipped"); ok {
		t.Fatal("json:- field should not be exported")
	}

	id := root.Properties.GetOrZero("id").Schema()
	if id == nil || id.Type[0] != "string" || id.Format != "uuid" {
		t.Fatalf("id should use custom uuid schema, got %#v", id)
	}
	status := root.Properties.GetOrZero("status")
	if !status.IsReference() || status.GetReference() != "#/components/schemas/ReflectConformanceStatus" {
		t.Fatalf("status should reference enum component, got %#v", status)
	}
	statusSchema := componentSchema(t, set, "ReflectConformanceStatus")
	if statusSchema.Type[0] != "string" || len(statusSchema.Enum) != 2 {
		t.Fatalf("status enum schema was not preserved: %#v", statusSchema)
	}
	count := root.Properties.GetOrZero("count").Schema()
	if count == nil || count.Type[0] != "string" {
		t.Fatalf("json,string count should render as string, got %#v", count)
	}
	nickname := root.Properties.GetOrZero("nickname").Schema()
	if nickname == nil || !schemaTypeContains(nickname.Type, "null") || nickname.Nullable != nil {
		t.Fatalf("pointer scalar should be nullable, got %#v", nickname)
	}
	data := root.Properties.GetOrZero("data").Schema()
	if data == nil || data.Type[0] != "string" || data.Format != "byte" {
		t.Fatalf("[]byte should render as string byte, got %#v", data)
	}
	raw := root.Properties.GetOrZero("raw").Schema()
	if raw == nil || len(raw.Type) != 0 {
		t.Fatalf("json.RawMessage should be unconstrained, got %#v", raw)
	}
	self := root.Properties.GetOrZero("self")
	assertNullableRef(t, self, "#/components/schemas/ReflectConformanceRoot")
	address := root.Properties.GetOrZero("address")
	assertNullableRef(t, address, "#/components/schemas/ReflectConformanceAddress")
	addressSchema := componentSchema(t, set, "ReflectConformanceAddress")
	if schemaTypeContains(addressSchema.Type, "null") || addressSchema.Nullable != nil {
		t.Fatalf("address component should stay non-null; nullable belongs at ref usage, got %#v", addressSchema)
	}
	if line2 := addressSchema.Properties.GetOrZero("line2").Schema(); line2 == nil || !schemaTypeContains(line2.Type, "null") || line2.Nullable != nil {
		t.Fatalf("pointer scalar in address should be nullable, got %#v", line2)
	}
	embedded := componentSchema(t, set, "ReflectConformanceEmbedded")
	if schemaTypeContains(embedded.Type, "null") || embedded.Nullable != nil {
		t.Fatalf("embedded component should stay non-null; nullable belongs at usage, got %#v", embedded)
	}
	labels := root.Properties.GetOrZero("labels")
	if !labels.IsReference() || labels.GetReference() != "#/components/schemas/ReflectConformanceRoot_Labels" {
		t.Fatalf("labels should reference map component, got %#v", labels)
	}
	labelSchema := componentSchema(t, set, "ReflectConformanceRoot_Labels")
	labelValue := labelSchema.AdditionalProperties.A.Schema()
	if labelValue == nil || labelValue.Type[0] != "string" {
		t.Fatalf("labels additionalProperties should be string, got %#v", labelSchema.AdditionalProperties)
	}
	attributes := componentSchema(t, set, "ReflectConformanceRoot_Attributes")
	if !attributes.AdditionalProperties.A.IsReference() || attributes.AdditionalProperties.A.GetReference() != "#/components/schemas/ReflectConformanceAttribute" {
		t.Fatalf("attributes map should reference attribute component, got %#v", attributes.AdditionalProperties)
	}
	payment := root.Properties.GetOrZero("payment")
	if !payment.IsReference() || payment.GetReference() != "#/components/schemas/ReflectConformanceRoot_Payment" {
		t.Fatalf("payment should reference union component, got %#v", payment)
	}
	paymentSchema := componentSchema(t, set, "ReflectConformanceRoot_Payment")
	if len(paymentSchema.OneOf) != 2 || paymentSchema.Discriminator == nil || paymentSchema.Discriminator.PropertyName != "object" {
		t.Fatalf("payment should be discriminated oneOf, got %#v", paymentSchema)
	}
	history := root.Properties.GetOrZero("history").Schema()
	if history == nil || history.Items == nil || !history.Items.IsA() || !history.Items.A.IsReference() {
		t.Fatalf("history should be array of union refs, got %#v", history)
	}
	if history.Items.A.GetReference() != "#/components/schemas/ReflectConformanceRoot_Payment" {
		t.Fatalf("history item should reference item union, got %q", history.Items.A.GetReference())
	}
	if !hasDiagnosticCode(set.Diagnostics, DiagnosticStringEncoded) {
		t.Fatalf("expected string encoded diagnostic, got %#v", set.Diagnostics)
	}
}

func assertNullableRef(t *testing.T, proxy *highbase.SchemaProxy, ref string) {
	t.Helper()
	if proxy == nil {
		t.Fatalf("nullable ref should render as anyOf wrapper for %s, got nil", ref)
	}
	schema := proxy.Schema()
	if schema == nil || len(schema.AnyOf) != 2 {
		t.Fatalf("nullable ref should render as anyOf wrapper, got %#v", proxy)
	}
	if !schema.AnyOf[0].IsReference() || schema.AnyOf[0].GetReference() != ref {
		t.Fatalf("nullable ref first variant should be %s, got %#v", ref, schema.AnyOf[0])
	}
	nullSchema := schema.AnyOf[1].Schema()
	if nullSchema == nil || !schemaTypeContains(nullSchema.Type, "null") {
		t.Fatalf("nullable ref second variant should be null schema, got %#v", schema.AnyOf[1])
	}
}

func TestReflectOpenAPIConformanceGoldenDocument(t *testing.T) {
	doc := renderReflectConformanceOpenAPIDocument(t, renderReflectConformanceSet(t))
	assertGolden(t, "testdata/reflect_openapi_conformance.golden.yaml", doc)
	parsed, err := libopenapi.NewDocument(doc)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parsed.BuildV3Model(); err != nil {
		t.Fatal(err)
	}
}

func TestReflectOpenAPIConformanceNameResolverCollision(t *testing.T) {
	set, err := SchemasFromTypesWithOptions([]reflect.Type{
		reflect.TypeOf(ReflectConformanceAddress{}),
		reflect.TypeOf(ReflectConformanceAttribute{}),
	}, WithTypeNameResolver(func(name string) string {
		if strings.HasPrefix(name, "ReflectConformance") {
			return "Collision"
		}
		return ""
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !hasDiagnosticCode(set.Diagnostics, DiagnosticRootNameCollision) || !hasDiagnosticCode(set.Diagnostics, DiagnosticComponentNameCollision) {
		t.Fatalf("expected root and component collision diagnostics, got %#v", set.Diagnostics)
	}
}

func renderReflectConformanceSet(t *testing.T) *SchemaSet {
	t.Helper()
	statusSchema := highbase.CreateSchemaProxy(&highbase.Schema{
		Type: []string{"string"},
		Enum: []*yaml.Node{
			stringNode("active"),
			stringNode("paused"),
		},
	})
	idSchema := highbase.CreateSchemaProxy(&highbase.Schema{
		Type:   []string{"string"},
		Format: "uuid",
	})
	set, err := SchemasFromTypesWithOptions([]reflect.Type{reflect.TypeOf(ReflectConformanceRoot{})},
		WithTypeSchema(reflect.TypeOf(ReflectConformanceID("")), idSchema),
		WithTypeSchema(reflect.TypeOf(ReflectConformanceStatus("")), statusSchema),
		WithOneOfTypes((*ReflectConformancePaymentMethod)(nil), ReflectConformanceCard{}, ReflectConformanceBank{}),
		WithDiscriminatorMapping((*ReflectConformancePaymentMethod)(nil), "object", map[string]string{
			"bank_account": "#/components/schemas/ReflectConformanceBank",
			"card":         "#/components/schemas/ReflectConformanceCard",
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	return set
}

func renderReflectConformanceOpenAPIDocument(t *testing.T, set *SchemaSet) []byte {
	t.Helper()
	var b strings.Builder
	b.WriteString("openapi: 3.1.0\n")
	b.WriteString("info:\n")
	b.WriteString("  title: Reflected Go Model Conformance API\n")
	b.WriteString("  version: 1.0.0\n")
	b.WriteString("paths: {}\n")
	b.WriteString("components:\n")
	b.WriteString("  schemas:\n")
	for name, proxy := range set.Components.FromOldest() {
		rendered, err := proxy.Render()
		if err != nil {
			t.Fatal(err)
		}
		b.WriteString("    ")
		b.WriteString(name)
		b.WriteString(":\n")
		b.WriteString(indentString(string(rendered), "      "))
	}
	return []byte(b.String())
}

func indentString(in, prefix string) string {
	lines := strings.Split(strings.TrimSuffix(in, "\n"), "\n")
	var b strings.Builder
	for _, line := range lines {
		b.WriteString(prefix)
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}
