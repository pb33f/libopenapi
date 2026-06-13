// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"errors"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

type MetadataTaggedModel struct {
	ID       string            `json:"id" openapi:"format=uuid;description=identifier;readOnly"`
	Secret   string            `json:"secret" openapi:"writeOnly;deprecated=false"`
	Old      string            `json:"old" openapi:"deprecated"`
	Optional *string           `json:"optional,omitempty" openapi:"nullable=false;minLength=3;maxLength=4;pattern=^[a-z]+$"`
	Amount   *float64          `json:"amount,omitempty" openapi:"nullable=false;minimum=1;maximum=10;exclusiveMinimum=0;exclusiveMaximum=11;multipleOf=0.5"`
	Tags     []string          `json:"tags,omitempty" openapi:"minItems=1;maxItems=3;uniqueItems=true"`
	Extras   map[string]string `json:"extras,omitempty" openapi:"minProperties=1;maxProperties=2"`
	Status   string            `json:"status" openapi:"enum=str:pending|str:done|int:7|float:1.5|bool:true|null:"`
	Kind     *string           `json:"kind,omitempty" openapi:"const=str:card;nullable=false"`
}

type MetadataQuotedTagModel struct {
	Value string `json:"value" openapi:"description=quote \"inside\" metadata;pattern=^\"[a-z]+\"$;enum=str:\"red\"|str:blue;const=str:\"red\""`
}

type MetadataFieldOverrideUnion struct {
	Value any `json:"value"`
}

type MetadataFieldOverride struct {
	Source MetadataFieldOverrideUnion `json:"source"`
	Alt    string                     `json:"alt"`
}

type MetadataNullableFieldOverride struct {
	Source *MetadataFieldOverrideUnion `json:"source,omitempty"`
}

type MetadataSchemaProvider struct{}

func (*MetadataSchemaProvider) OpenAPISchema() *highbase.SchemaProxy {
	return highbase.CreateSchemaProxy(&highbase.Schema{
		Type: []string{"object"},
		Properties: orderedmap.ToOrderedMap(map[string]*highbase.SchemaProxy{
			"code": highbase.CreateSchemaProxy(&highbase.Schema{
				Type:   []string{"string"},
				Format: "uuid",
			}),
		}),
		Required: []string{"code"},
	})
}

type MetadataSchemaProviderHolder struct {
	Provider *MetadataSchemaProvider `json:"provider,omitempty"`
}

var metadataCountingSchemaProviderCalls int

type MetadataCountingSchemaProvider struct{}

func (*MetadataCountingSchemaProvider) OpenAPISchema() *highbase.SchemaProxy {
	metadataCountingSchemaProviderCalls++
	return highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"object"}})
}

type MetadataCountingSchemaProviderHolder struct {
	Provider *MetadataCountingSchemaProvider `json:"provider,omitempty"`
}

type MetadataYAMLProvider struct{}

func (*MetadataYAMLProvider) OpenAPISchemaYAML() string {
	return "type: object\nproperties:\n  code:\n    type: string\n    format: uuid\nrequired:\n  - code\n"
}

type MetadataBadYAMLProvider struct{}

func (*MetadataBadYAMLProvider) OpenAPISchemaYAML() string {
	return "type: ["
}

type MetadataTypedProvider struct{}

func (*MetadataTypedProvider) OpenAPISchemaMetadata() any {
	return &providerSchemaMetadata{
		Type:     []string{"object"},
		Required: []string{"code"},
		Properties: []providerNamedSchemaMetadata{{
			Name: "code",
			Schema: &providerSchemaMetadata{
				Type:   []string{"string"},
				Format: "uuid",
			},
		}},
		AdditionalProperties: &providerDynamicSchemaBool{Bool: &providerBool{Value: false}},
	}
}

type MetadataBadTypedProvider struct{}

func (*MetadataBadTypedProvider) OpenAPISchemaMetadata() any {
	return func() {}
}

type MetadataInvalidTypedProvider struct{}

func (*MetadataInvalidTypedProvider) OpenAPISchemaMetadata() any {
	return map[string]any{"Type": 7}
}

type MetadataProviderHolder struct {
	Provider *MetadataYAMLProvider `json:"provider,omitempty"`
}

type MetadataTypedProviderHolder struct {
	Provider *MetadataTypedProvider `json:"provider,omitempty"`
}

func TestOpenAPITagMetadataReflectsIntoSchema(t *testing.T) {
	set, err := SchemasFromTypes(reflect.TypeOf(MetadataTaggedModel{}))
	if err != nil {
		t.Fatal(err)
	}
	root := componentSchema(t, set, "MetadataTaggedModel")
	id := root.Properties.GetOrZero("id").Schema()
	if id.Format != "uuid" || id.Description != "identifier" || id.ReadOnly == nil || !*id.ReadOnly {
		t.Fatalf("id metadata not reflected: %#v", id)
	}
	secret := root.Properties.GetOrZero("secret").Schema()
	if secret.WriteOnly == nil || !*secret.WriteOnly || secret.Deprecated == nil || *secret.Deprecated {
		t.Fatalf("secret metadata not reflected: %#v", secret)
	}
	old := root.Properties.GetOrZero("old").Schema()
	if old.Deprecated == nil || !*old.Deprecated {
		t.Fatalf("deprecated metadata not reflected: %#v", old)
	}
	optional := root.Properties.GetOrZero("optional").Schema()
	if schemaTypeContains(optional.Type, "null") || optional.MinLength == nil || *optional.MinLength != 3 || optional.MaxLength == nil || *optional.MaxLength != 4 || optional.Pattern != "^[a-z]+$" {
		t.Fatalf("optional metadata not reflected: %#v", optional)
	}
	amount := root.Properties.GetOrZero("amount").Schema()
	if schemaTypeContains(amount.Type, "null") || amount.Minimum == nil || *amount.Minimum != 1 || amount.Maximum == nil || *amount.Maximum != 10 {
		t.Fatalf("amount range metadata not reflected: %#v", amount)
	}
	if amount.ExclusiveMinimum == nil || !amount.ExclusiveMinimum.IsB() || amount.ExclusiveMinimum.B != 0 {
		t.Fatalf("exclusive minimum not reflected: %#v", amount.ExclusiveMinimum)
	}
	if amount.ExclusiveMaximum == nil || !amount.ExclusiveMaximum.IsB() || amount.ExclusiveMaximum.B != 11 {
		t.Fatalf("exclusive maximum not reflected: %#v", amount.ExclusiveMaximum)
	}
	if amount.MultipleOf == nil || *amount.MultipleOf != 0.5 {
		t.Fatalf("multipleOf not reflected: %#v", amount.MultipleOf)
	}
	tags := root.Properties.GetOrZero("tags").Schema()
	if tags.MinItems == nil || *tags.MinItems != 1 || tags.MaxItems == nil || *tags.MaxItems != 3 || tags.UniqueItems == nil || !*tags.UniqueItems {
		t.Fatalf("array metadata not reflected: %#v", tags)
	}
	extras := root.Properties.GetOrZero("extras").Schema()
	if extras.MinProperties == nil || *extras.MinProperties != 1 || extras.MaxProperties == nil || *extras.MaxProperties != 2 {
		t.Fatalf("object metadata not reflected: %#v", extras)
	}
	status := root.Properties.GetOrZero("status").Schema()
	if len(status.Enum) != 6 || status.Enum[2].Tag != "!!int" || status.Enum[5].Tag != "!!null" {
		t.Fatalf("enum metadata not reflected: %#v", status.Enum)
	}
	kind := root.Properties.GetOrZero("kind").Schema()
	if schemaTypeContains(kind.Type, "null") || kind.Const == nil || kind.Const.Value != "card" {
		t.Fatalf("const metadata not reflected: %#v", kind)
	}
}

func TestOpenAPITagQuotedMetadataRoundTrips(t *testing.T) {
	schema := highbase.CreateSchemaProxy(&highbase.Schema{
		Type:        []string{"object"},
		Description: "quoted tag test",
		Properties: orderedmap.ToOrderedMap(map[string]*highbase.SchemaProxy{
			"value": highbase.CreateSchemaProxy(&highbase.Schema{
				Type:        []string{"string"},
				Description: `quote "inside" metadata`,
				Pattern:     `^"[a-z]+"$`,
				Enum:        []*yaml.Node{stringNode(`"red"`), stringNode("blue")},
				Const:       stringNode(`"red"`),
			}),
		}),
	})
	source, err := RenderSchema("Quoted", schema, WithOpenAPITags(true))
	if err != nil {
		t.Fatal(err)
	}
	src := string(source)
	assertContains(t, src, `description=quote \"inside\" metadata`)
	assertContains(t, src, `pattern=^\"[a-z]+\"$`)
	assertContains(t, src, `enum=str:\"red\"|str:blue`)
	assertContains(t, src, `const=str:\"red\"`)
	if _, err := parser.ParseFile(token.NewFileSet(), "quoted.go", source, parser.AllErrors); err != nil {
		t.Fatalf("quoted tag source should parse: %v\n%s", err, source)
	}
	assertParsesCompilesAndTests(t, source, `package models

import (
	"reflect"
	"strings"
	"testing"
)

func TestGeneratedQuotedOpenAPITag(t *testing.T) {
	field, ok := reflect.TypeOf(Quoted{}).FieldByName("Value")
	if !ok {
		t.Fatal("missing generated field")
	}
	tag := field.Tag.Get("openapi")
	for _, want := range []string{
		"description=quote \"inside\" metadata",
		"pattern=^\"[a-z]+\"$",
		"enum=str:\"red\"|str:blue",
		"const=str:\"red\"",
	} {
		if !strings.Contains(tag, want) {
			t.Fatalf("generated tag missing %q in %q", want, tag)
		}
	}
}
`)

	field, ok := reflect.TypeOf(MetadataQuotedTagModel{}).FieldByName("Value")
	if !ok {
		t.Fatal("missing quoted tag field")
	}
	rawTag := field.Tag.Get("openapi")
	for _, want := range []string{
		`description=quote "inside" metadata`,
		`pattern=^"[a-z]+"$`,
		`enum=str:"red"|str:blue`,
		`const=str:"red"`,
	} {
		if !strings.Contains(rawTag, want) {
			t.Fatalf("reflect.StructTag.Get truncated or corrupted %q in %q", want, rawTag)
		}
	}

	set, err := SchemasFromTypes(reflect.TypeOf(MetadataQuotedTagModel{}))
	if err != nil {
		t.Fatal(err)
	}
	prop := componentSchema(t, set, "MetadataQuotedTagModel").Properties.GetOrZero("value").Schema()
	if prop.Description != `quote "inside" metadata` || prop.Pattern != `^"[a-z]+"$` {
		t.Fatalf("quoted metadata did not reflect: %#v", prop)
	}
	if len(prop.Enum) != 2 || prop.Enum[0].Value != `"red"` || prop.Const == nil || prop.Const.Value != `"red"` {
		t.Fatalf("quoted enum/const metadata did not reflect: %#v", prop)
	}
}

func TestApplyOpenAPIMetadataKeepsEquivalentSourceYAMLNodes(t *testing.T) {
	enum := []*yaml.Node{{Kind: yaml.ScalarNode, Tag: "!!str", Value: "bam"}}
	constValue := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "card"}
	schema := &highbase.Schema{Enum: enum, Const: constValue}
	ir := &SchemaIR{Enum: enum, Const: constValue, SourceSchema: schema}

	NewGenerator().applyOpenAPIMetadata(ir, openAPIMetadata{
		Present: true,
		Enum:    []*yaml.Node{{Kind: yaml.ScalarNode, Value: "bam"}},
		Const:   &yaml.Node{Kind: yaml.ScalarNode, Value: "card"},
	})

	if ir.Enum[0].Tag != "!!str" || schema.Enum[0].Tag != "!!str" {
		t.Fatalf("equivalent enum metadata stripped source tag: %#v", ir.Enum[0])
	}
	if ir.Const.Tag != "!!str" || schema.Const.Tag != "!!str" {
		t.Fatalf("equivalent const metadata stripped source tag: %#v", ir.Const)
	}

	NewGenerator().applyOpenAPIMetadata(ir, openAPIMetadata{
		Present: true,
		Enum:    []*yaml.Node{{Kind: yaml.ScalarNode, Tag: "!!str", Value: "bgn"}},
		Const:   &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "bank_account"},
	})

	if ir.Enum[0].Value != "bgn" || schema.Enum[0].Value != "bgn" {
		t.Fatalf("changed enum metadata was not applied: %#v", ir.Enum[0])
	}
	if ir.Const.Value != "bank_account" || schema.Const.Value != "bank_account" {
		t.Fatalf("changed const metadata was not applied: %#v", ir.Const)
	}
}

func TestFieldSchemaOverridesAndSchemaYAMLProvider(t *testing.T) {
	sourceSchema := schemaProxyFromYAML(t, `
oneOf:
  - type: object
    properties:
      object:
        type: string
        const: card
    required:
      - object
`)
	altSchema := highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"string"}, Format: "uuid"})
	set, err := NewGenerator(
		WithFieldSchema(reflect.TypeOf(MetadataFieldOverride{}), "Source", sourceSchema),
		WithFieldSchemaByJSONName(reflect.TypeOf(MetadataFieldOverride{}), "alt", altSchema),
	).SchemasFromTypes(reflect.TypeOf(MetadataFieldOverride{}))
	if err != nil {
		t.Fatal(err)
	}
	root := componentSchema(t, set, "MetadataFieldOverride")
	source := root.Properties.GetOrZero("source").Schema()
	if len(source.OneOf) != 1 {
		t.Fatalf("field schema override did not preserve oneOf: %#v", source)
	}
	alt := root.Properties.GetOrZero("alt").Schema()
	if alt.Format != "uuid" {
		t.Fatalf("json-name field schema override did not apply: %#v", alt)
	}
	nullableSet, err := NewGenerator(
		WithFieldSchema(reflect.TypeOf(MetadataNullableFieldOverride{}), "Source", sourceSchema),
	).SchemasFromTypes(reflect.TypeOf(MetadataNullableFieldOverride{}))
	if err != nil {
		t.Fatal(err)
	}
	nullableRoot := componentSchema(t, nullableSet, "MetadataNullableFieldOverride")
	nullableSource := nullableRoot.Properties.GetOrZero("source").Schema()
	if nullableSource == nil || len(nullableSource.AnyOf) != 2 {
		t.Fatalf("nullable composed field should render as anyOf original/null, got %#v", nullableSource)
	}
	if original := nullableSource.AnyOf[0].Schema(); original == nil || len(original.OneOf) != 1 {
		t.Fatalf("nullable composed field should preserve original oneOf branch, got %#v", original)
	}
	if nullSchema := nullableSource.AnyOf[1].Schema(); nullSchema == nil || !schemaTypeContains(nullSchema.Type, "null") {
		t.Fatalf("nullable composed field should include native null branch, got %#v", nullSchema)
	}

	providerSet, err := SchemasFromTypes(reflect.TypeOf(MetadataYAMLProvider{}))
	if err != nil {
		t.Fatal(err)
	}
	provider := componentSchema(t, providerSet, "MetadataYamlProvider")
	code := provider.Properties.GetOrZero("code").Schema()
	if code == nil || code.Format != "uuid" || !containsString(provider.Required, "code") {
		t.Fatalf("schema YAML provider did not reflect: %#v", provider)
	}
	if proxy, err := schemaProxyFromProviderYAML("", "type: string\n"); err != nil || proxy.Schema().Type[0] != "string" {
		t.Fatalf("provider yaml helper failed: %#v %v", proxy, err)
	}
	metadataSet, err := SchemasFromTypes(reflect.TypeOf(MetadataTypedProvider{}))
	if err != nil {
		t.Fatal(err)
	}
	metadataProvider := componentSchema(t, metadataSet, "MetadataTypedProvider")
	metadataCode := metadataProvider.Properties.GetOrZero("code").Schema()
	if metadataCode == nil || metadataCode.Format != "uuid" || !containsString(metadataProvider.Required, "code") {
		t.Fatalf("schema metadata provider did not reflect: %#v", metadataProvider)
	}
	metadataHolderSet, err := SchemasFromTypes(reflect.TypeOf(MetadataTypedProviderHolder{}))
	if err != nil {
		t.Fatal(err)
	}
	metadataHolder := componentSchema(t, metadataHolderSet, "MetadataTypedProviderHolder")
	if prop := metadataHolder.Properties.GetOrZero("provider").Schema(); prop == nil || len(prop.AnyOf) != 2 {
		t.Fatalf("nullable schema metadata provider should render as anyOf ref/null: %#v", metadataHolder.Properties.GetOrZero("provider"))
	}
	holderSet, err := SchemasFromTypes(reflect.TypeOf(MetadataProviderHolder{}))
	if err != nil {
		t.Fatal(err)
	}
	holder := componentSchema(t, holderSet, "MetadataProviderHolder")
	if prop := holder.Properties.GetOrZero("provider").Schema(); prop == nil || len(prop.AnyOf) != 2 {
		t.Fatalf("nullable schema yaml provider should render as anyOf ref/null: %#v", holder.Properties.GetOrZero("provider"))
	}
	if _, err := SchemasFromTypes(reflect.TypeOf(MetadataBadYAMLProvider{})); err == nil {
		t.Fatal("invalid schema yaml provider should return an error")
	}
	if _, err := schemaProxyFromProviderYAML("Broken", "type: ["); err == nil {
		t.Fatal("invalid provider yaml helper should fail")
	}
	if _, err := SchemasFromTypes(reflect.TypeOf(MetadataBadTypedProvider{})); err == nil {
		t.Fatal("bad schema metadata provider should return an error")
	}
	if _, err := SchemasFromTypes(reflect.TypeOf(MetadataInvalidTypedProvider{})); err == nil {
		t.Fatal("invalid schema metadata provider should return an error")
	}
}

func TestProviderSchemasReuseCanonicalNamesAcrossRootsAndFields(t *testing.T) {
	cases := []struct {
		holderName   string
		holderType   reflect.Type
		providerName string
		providerType reflect.Type
	}{
		{
			holderName:   "MetadataSchemaProviderHolder",
			holderType:   reflect.TypeOf(MetadataSchemaProviderHolder{}),
			providerName: "MetadataSchemaProvider",
			providerType: reflect.TypeOf(MetadataSchemaProvider{}),
		},
		{
			holderName:   "MetadataProviderHolder",
			holderType:   reflect.TypeOf(MetadataProviderHolder{}),
			providerName: "MetadataYamlProvider",
			providerType: reflect.TypeOf(MetadataYAMLProvider{}),
		},
		{
			holderName:   "MetadataTypedProviderHolder",
			holderType:   reflect.TypeOf(MetadataTypedProviderHolder{}),
			providerName: "MetadataTypedProvider",
			providerType: reflect.TypeOf(MetadataTypedProvider{}),
		},
	}
	for _, tc := range cases {
		t.Run(tc.providerName, func(t *testing.T) {
			set, err := SchemasFromTypes(tc.holderType, tc.providerType)
			if err != nil {
				t.Fatal(err)
			}
			if root, ok := set.Roots.Get(tc.providerName); !ok || !root.IsReference() || root.GetReference() != "#/components/schemas/"+tc.providerName {
				t.Fatalf("provider root should use canonical provider component name, got %#v", root)
			}
			if _, ok := set.Components.Get(tc.providerName); !ok {
				t.Fatalf("provider component %q missing from %#v", tc.providerName, set.Components)
			}
			fieldDerivedName := NewGenerator().nestedTypeName(tc.holderName, "provider")
			if _, ok := set.Components.Get(fieldDerivedName); ok {
				t.Fatalf("field-derived provider component %q should not be emitted", fieldDerivedName)
			}
			holder := componentSchema(t, set, tc.holderName)
			assertNullableRef(t, holder.Properties.GetOrZero("provider"), "#/components/schemas/"+tc.providerName)
		})
	}

	metadataCountingSchemaProviderCalls = 0
	if _, err := SchemasFromTypes(reflect.TypeOf(MetadataCountingSchemaProvider{}), reflect.TypeOf(MetadataCountingSchemaProviderHolder{})); err != nil {
		t.Fatal(err)
	}
	if metadataCountingSchemaProviderCalls != 1 {
		t.Fatalf("cached provider schema should be reused for repeated provider type, got %d calls", metadataCountingSchemaProviderCalls)
	}
}

func TestGeneratedOpenAPITagsAndProviderMethods(t *testing.T) {
	file := renderTrainTravel(t,
		WithOpenAPITags(true),
		WithSchemaMetadataSidecar(true),
		WithOptionalConstDiscriminatorUnions(true),
	)
	source := string(file.Source)
	assertContains(t, source, `openapi:"format=uuid"`)
	assertContains(t, source, `openapi:"writeOnly;minLength=3;maxLength=4"`)
	assertNotContains(t, source, "var openAPISchemas = map[string]*openAPISchemaMetadata")
	assertNotContains(t, source, "OpenAPISchemaMetadata")
	if file.SchemaMetadata == nil {
		t.Fatal("expected schema metadata sidecar")
	}
	if file.SchemaMetadata.Name != SchemaMetadataFileName {
		t.Fatalf("unexpected schema metadata file name %q", file.SchemaMetadata.Name)
	}
	metadataSource := string(file.SchemaMetadata.Source)
	assertContains(t, metadataSource, "var openAPISchemas = map[string]*openAPISchemaMetadata")
	assertContains(t, metadataSource, "func (Station) OpenAPISchemaMetadata() any")
	assertContains(t, metadataSource, "func (BookingPayment_SourceUnion) OpenAPISchemaMetadata() any")
	if strings.Contains(metadataSource, "OpenAPISchemaYAML") {
		t.Fatal("generated sidecar should not emit yaml provider methods")
	}
	assertParsesCompilesAndTestsWithFiles(t, map[string][]byte{
		"models.go":              file.Source,
		file.SchemaMetadata.Name: file.SchemaMetadata.Source,
	}, "package models\n\nimport \"testing\"\n\nfunc TestGeneratedPackage(t *testing.T) {}\n")

	withoutSidecar := renderTrainTravel(t, WithOpenAPITags(true))
	if withoutSidecar.SchemaMetadata != nil {
		t.Fatal("schema metadata sidecar should be disabled unless requested")
	}
	if strings.Contains(string(withoutSidecar.Source), "OpenAPISchemaMetadata") || strings.Contains(string(withoutSidecar.Source), "openAPISchemas") {
		t.Fatal("schema metadata sidecar should be disabled unless requested")
	}
}

func TestSchemaMetadataSidecarFileHeaderAndRenderError(t *testing.T) {
	schemas := orderedmap.New[string, *highbase.SchemaProxy]()
	schemas.Set("Sample", highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"string"}}))
	file, err := NewGenerator(
		WithHeaderComment("Schema metadata header."),
		WithSchemaMetadataSidecar(true),
	).RenderSchemas(schemas)
	if err != nil {
		t.Fatal(err)
	}
	if file.SchemaMetadata == nil {
		t.Fatal("expected schema metadata sidecar")
	}
	assertContains(t, string(file.SchemaMetadata.Source), "// Schema metadata header.")

	_, err = NewGenerator(
		WithSchemaMetadataSidecar(true),
		WithTypeNameResolver(func(string) string { return "Bad Type" }),
	).RenderSchemas(schemas)
	if err == nil {
		t.Fatal("expected invalid sidecar source to return an error")
	}

	oldFormatSource := formatSource
	formatSource = func(src []byte) ([]byte, error) {
		if strings.Contains(string(src), "openAPISchemas") {
			return nil, errors.New("sidecar format failed")
		}
		return oldFormatSource(src)
	}
	defer func() {
		formatSource = oldFormatSource
	}()
	_, err = NewGenerator(WithSchemaMetadataSidecar(true)).RenderSchemas(schemas)
	if err == nil {
		t.Fatal("expected sidecar formatting error")
	}
}

func TestSchemaMetadataSidecarTypedDataCoversJSONSchemaKeywords(t *testing.T) {
	zeroFloat := float64(0)
	oneFloat := float64(1)
	tenFloat := float64(10)
	zeroInt := int64(0)
	oneInt := int64(1)
	twoInt := int64(2)
	trueValue := true
	falseValue := false
	discriminatorMapping := orderedmap.New[string, string]()
	discriminatorMapping.Set("card", "#/components/schemas/Card")
	vocabulary := orderedmap.New[string, bool]()
	vocabulary.Set("https://json-schema.org/draft/2020-12/vocab/core", true)
	properties := orderedmap.New[string, *highbase.SchemaProxy]()
	properties.Set("id", highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"string"}, Format: "uuid"}))
	patternProperties := orderedmap.New[string, *highbase.SchemaProxy]()
	patternProperties.Set("^x-", highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"string"}}))
	dependentSchemas := orderedmap.New[string, *highbase.SchemaProxy]()
	dependentSchemas.Set("id", highbase.CreateSchemaProxy(&highbase.Schema{Required: []string{"kind"}}))
	dependentRequired := orderedmap.New[string, []string]()
	dependentRequired.Set("id", []string{"kind"})
	extensions := orderedmap.New[string, *yaml.Node]()
	extensions.Set("x-test", stringNode("extension"))
	defaultNode := &yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
		Content: []*yaml.Node{
			stringNode("enabled"),
			{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"},
		},
	}
	aliasTarget := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "alias-target", Anchor: "target"}
	schema := &highbase.Schema{
		SchemaTypeRef:    "https://json-schema.org/draft/2020-12/schema",
		ExclusiveMaximum: &highbase.DynamicValue[bool, float64]{A: true},
		ExclusiveMinimum: &highbase.DynamicValue[bool, float64]{N: 1, B: zeroFloat},
		Type:             []string{"object"},
		AllOf:            []*highbase.SchemaProxy{highbase.CreateSchemaProxyRef("#/components/schemas/Base")},
		OneOf:            []*highbase.SchemaProxy{highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"string"}})},
		AnyOf:            []*highbase.SchemaProxy{highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"null"}})},
		Discriminator: &highbase.Discriminator{
			PropertyName:   "kind",
			Mapping:        discriminatorMapping,
			DefaultMapping: "#/components/schemas/Fallback",
		},
		Examples:              []*yaml.Node{stringNode("example")},
		PrefixItems:           []*highbase.SchemaProxy{highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"integer"}})},
		Contains:              highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"string"}}),
		MinContains:           &zeroInt,
		MaxContains:           &twoInt,
		If:                    highbase.CreateSchemaProxy(&highbase.Schema{Required: []string{"kind"}}),
		Else:                  highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"object"}}),
		Then:                  highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"object"}}),
		DependentSchemas:      dependentSchemas,
		DependentRequired:     dependentRequired,
		PatternProperties:     patternProperties,
		PropertyNames:         highbase.CreateSchemaProxy(&highbase.Schema{Pattern: "^[a-z]+$"}),
		UnevaluatedItems:      highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"boolean"}}),
		UnevaluatedProperties: &highbase.DynamicValue[*highbase.SchemaProxy, bool]{N: 1, B: false},
		Items:                 &highbase.DynamicValue[*highbase.SchemaProxy, bool]{A: highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"string"}})},
		Id:                    "https://example.com/schema",
		Anchor:                "root",
		DynamicAnchor:         "node",
		DynamicRef:            "#node",
		Comment:               "comment",
		ContentSchema:         highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"object"}}),
		Vocabulary:            vocabulary,
		Not:                   highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"null"}}),
		Properties:            properties,
		Title:                 "Typed Metadata",
		MultipleOf:            &oneFloat,
		Maximum:               &tenFloat,
		Minimum:               &zeroFloat,
		MaxLength:             &twoInt,
		MinLength:             &zeroInt,
		Pattern:               "^[a-z]+$",
		Format:                "uuid",
		MaxItems:              &twoInt,
		MinItems:              &zeroInt,
		UniqueItems:           &trueValue,
		MaxProperties:         &twoInt,
		MinProperties:         &oneInt,
		Required:              []string{"id"},
		Enum:                  []*yaml.Node{stringNode("a"), nullNode()},
		AdditionalProperties:  &highbase.DynamicValue[*highbase.SchemaProxy, bool]{A: highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"number"}})},
		Description:           "description",
		ContentEncoding:       "base64",
		ContentMediaType:      "application/json",
		Default:               defaultNode,
		Const:                 stringNode("constant"),
		Nullable:              &falseValue,
		ReadOnly:              &trueValue,
		WriteOnly:             &falseValue,
		Example:               &yaml.Node{Kind: yaml.AliasNode, Alias: aliasTarget},
		Deprecated:            &trueValue,
		Extensions:            extensions,
	}

	gen := NewGenerator(WithSchemaMetadataSidecar(true))
	gen.recordSchemaMetadata("Sample", schema)
	sidecar := gen.renderSchemaMetadataSidecarDecl()
	if sidecar == "" {
		t.Fatal("expected metadata sidecar declaration")
	}
	for _, want := range []string{"SchemaTypeRef", "ExclusiveMaximum", "DependentRequired", "UnevaluatedProperties", "ContentMediaType", "Extensions"} {
		assertContains(t, sidecar, want)
	}

	proxy, err := schemaProxyFromProviderMetadata(&providerSchemaMetadata{
		SchemaTypeRef:    "https://json-schema.org/draft/2020-12/schema",
		Type:             []string{"object"},
		AllOf:            []*providerSchemaMetadata{{Ref: "#/components/schemas/Base"}},
		OneOf:            []*providerSchemaMetadata{{Type: []string{"string"}}},
		AnyOf:            []*providerSchemaMetadata{{Type: []string{"null"}}},
		Discriminator:    &providerDiscriminatorMetadata{PropertyName: "kind", Mapping: []providerStringString{{Name: "card", Value: "#/components/schemas/Card"}}, DefaultMapping: "#/components/schemas/Fallback"},
		Examples:         []*providerYAMLNode{{Kind: "sequence", Tag: "!!seq", Content: []*providerYAMLNode{{Kind: "scalar", Tag: "!!str", Value: "example"}}}},
		PrefixItems:      []*providerSchemaMetadata{{Type: []string{"integer"}}},
		Contains:         &providerSchemaMetadata{Type: []string{"string"}},
		MinContains:      &providerInt{Value: 0},
		MaxContains:      &providerInt{Value: 1},
		If:               &providerSchemaMetadata{Required: []string{"kind"}},
		Else:             &providerSchemaMetadata{Type: []string{"object"}},
		Then:             &providerSchemaMetadata{Type: []string{"object"}},
		DependentSchemas: []providerNamedSchemaMetadata{{Name: "id", Schema: &providerSchemaMetadata{Required: []string{"kind"}}}},
		DependentRequired: []providerStringList{{
			Name:   "id",
			Values: []string{"kind"},
		}},
		PatternProperties: []providerNamedSchemaMetadata{{Name: "^x-", Schema: &providerSchemaMetadata{Type: []string{"string"}}}},
		PropertyNames:     &providerSchemaMetadata{Pattern: "^[a-z]+$"},
		UnevaluatedItems:  &providerSchemaMetadata{Type: []string{"boolean"}},
		Properties: []providerNamedSchemaMetadata{{
			Name: "id",
			Schema: &providerSchemaMetadata{
				Type:   []string{"string"},
				Format: "uuid",
			},
		}},
		Required:              []string{"id"},
		AdditionalProperties:  &providerDynamicSchemaBool{Bool: &providerBool{Value: false}},
		UnevaluatedProperties: &providerDynamicSchemaBool{Schema: &providerSchemaMetadata{Type: []string{"string"}}},
		ExclusiveMaximum:      &providerDynamicBoolNumber{Bool: &providerBool{Value: true}},
		ExclusiveMinimum:      &providerDynamicBoolNumber{Number: &providerFloat{Value: 0}},
		ID:                    "https://example.com/schema",
		Anchor:                "root",
		DynamicAnchor:         "node",
		DynamicRef:            "#node",
		Comment:               "comment",
		ContentSchema:         &providerSchemaMetadata{Type: []string{"object"}},
		Vocabulary:            []providerStringBool{{Name: "https://json-schema.org/draft/2020-12/vocab/core", Value: true}},
		Not:                   &providerSchemaMetadata{Type: []string{"null"}},
		Title:                 "Typed Metadata",
		MultipleOf:            &providerFloat{Value: 1},
		Maximum:               &providerFloat{Value: 10},
		Minimum:               &providerFloat{Value: 0},
		MinLength:             &providerInt{Value: 0},
		MaxLength:             &providerInt{Value: 2},
		Pattern:               "^[a-z]+$",
		MaxItems:              &providerInt{Value: 2},
		MinItems:              &providerInt{Value: 0},
		UniqueItems:           &providerBool{Value: true},
		MaxProperties:         &providerInt{Value: 2},
		MinProperties:         &providerInt{Value: 1},
		Enum:                  []*providerYAMLNode{{Kind: "scalar", Tag: "!!str", Value: "a"}},
		Description:           "description",
		ContentEncoding:       "base64",
		ContentMediaType:      "application/json",
		Default: &providerYAMLNode{
			Kind:  "mapping",
			Tag:   "!!map",
			Value: "",
			Content: []*providerYAMLNode{
				{Kind: "scalar", Tag: "!!str", Value: "enabled"},
				{Kind: "scalar", Tag: "!!bool", Value: "true"},
			},
		},
		Const:      &providerYAMLNode{Kind: "document", Content: []*providerYAMLNode{{Kind: "scalar", Tag: "!!str", Value: "constant"}}},
		Nullable:   &providerBool{Value: false},
		ReadOnly:   &providerBool{Value: true},
		WriteOnly:  &providerBool{Value: false},
		Example:    &providerYAMLNode{Kind: "alias", Alias: &providerYAMLNode{Kind: "scalar", Tag: "!!str", Value: "alias-target", Anchor: "target"}},
		Deprecated: &providerBool{Value: true},
		Extensions: []providerNamedYAMLNode{{
			Name:  "x-test",
			Value: &providerYAMLNode{Kind: "scalar", Tag: "!!str", Value: "extension"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	roundTrip := proxy.Schema()
	if roundTrip.Properties.GetOrZero("id").Schema().Format != "uuid" || roundTrip.AdditionalProperties == nil || !roundTrip.AdditionalProperties.IsB() || roundTrip.MinLength == nil || *roundTrip.MinLength != 0 {
		t.Fatalf("typed provider metadata did not convert: %#v", roundTrip)
	}
	refProxy := schemaProxyFromMetadata(&providerSchemaMetadata{
		Ref:         "#/components/schemas/Target",
		Description: "sibling",
	})
	if !refProxy.IsReference() || refProxy.Schema().Description != "sibling" {
		t.Fatalf("ref sibling metadata did not convert: %#v", refProxy)
	}
}

func TestMetadataHelpersCoverage(t *testing.T) {
	meta := parseOpenAPITag(`;format=uuid;title=Example;description=a\;b\=c\|d;nullable=maybe;minimum;maximum=bad;minLength;maxLength=bad;uniqueItems=maybe;readOnly=false;unknown=value;`)
	if !meta.Present || meta.NullableSet || meta.MinimumSet || meta.MaximumSet || meta.MinLengthSet || meta.MaxLengthSet || meta.UniqueItemsSet {
		t.Fatalf("invalid tag values should be ignored: %#v", meta)
	}
	enumPipe := parseOpenAPITag(`enum=str:a\|b|str:c`)
	if len(enumPipe.Enum) != 2 || enumPipe.Enum[0].Value != "a|b" || enumPipe.Enum[1].Value != "c" {
		t.Fatalf("escaped enum separator should survive splitting: %#v", enumPipe.Enum)
	}
	if got := unescapeOpenAPITagValue(`trailing\`); got != `trailing\` {
		t.Fatalf("trailing escape not preserved: %q", got)
	}
	gen := NewGenerator()
	if tag := gen.openAPITagLiteral(nil, "string"); tag != "" {
		t.Fatalf("nil ir should not render openapi tag: %q", tag)
	}
	if tag := gen.openAPITagLiteral(&SchemaIR{Kind: KindString}, "string"); tag != "" {
		t.Fatalf("disabled openapi tags should not render: %q", tag)
	}
	gen = NewGenerator(WithOpenAPITags(true))
	tag := gen.openAPITagLiteral(&SchemaIR{
		Kind:        KindString,
		Title:       "bad`title",
		Description: "",
	}, "string")
	if strings.Contains(tag, "bad`title") {
		t.Fatalf("unsafe backtick tag value should be skipped: %q", tag)
	}
	min := float64(1)
	max := float64(10)
	multiple := float64(0.5)
	minLen := int64(2)
	maxLen := int64(8)
	minItems := int64(1)
	maxItems := int64(3)
	unique := true
	minProps := int64(1)
	maxProps := int64(4)
	tag = gen.openAPITagLiteral(&SchemaIR{
		Kind:        KindString,
		Nullable:    true,
		Format:      "uuid",
		Title:       "Title",
		Description: "Description",
		ReadOnly:    true,
		WriteOnly:   true,
		Deprecated:  true,
		Enum:        []*yaml.Node{stringNode("a")},
		Const:       stringNode("a"),
		SourceSchema: &highbase.Schema{
			Minimum:       &min,
			Maximum:       &max,
			MultipleOf:    &multiple,
			MinLength:     &minLen,
			MaxLength:     &maxLen,
			Pattern:       "^[a]$",
			MinItems:      &minItems,
			MaxItems:      &maxItems,
			UniqueItems:   &unique,
			MinProperties: &minProps,
			MaxProperties: &maxProps,
			ExclusiveMinimum: &highbase.DynamicValue[bool, float64]{
				N: 1,
				B: min,
			},
			ExclusiveMaximum: &highbase.DynamicValue[bool, float64]{
				N: 1,
				B: max,
			},
		},
	}, "string")
	for _, want := range []string{"nullable=true", "title=Title", "description=Description", "readOnly", "writeOnly", "deprecated", "enum=str:a", "const=str:a", "minimum=1", "exclusiveMaximum=10", "maxProperties=4"} {
		if !strings.Contains(tag, want) {
			t.Fatalf("tag %q missing %q", tag, want)
		}
	}
	tag = gen.openAPITagLiteral(&SchemaIR{
		Kind:  KindString,
		Enum:  []*yaml.Node{stringNode("a|b"), stringNode("safe"), stringNode("bad`tick")},
		Const: stringNode("bad`tick"),
	}, "string")
	if !strings.Contains(tag, `enum=str:a\|b|str:safe`) || strings.Contains(tag, "bad`tick") || strings.Contains(tag, "const=") {
		t.Fatalf("tag should keep escaped safe enum values and skip unsafe enum/const values: %q", tag)
	}
	parsedSafe := parseOpenAPITag(tag)
	if len(parsedSafe.Enum) != 2 || parsedSafe.Enum[0].Value != "a|b" || parsedSafe.Enum[1].Value != "safe" || parsedSafe.Const != nil {
		t.Fatalf("safe enum tag should parse after unsafe values are skipped: %#v", parsedSafe)
	}
	if node := parseTagNode("bare"); node == nil || node.Value != "bare" {
		t.Fatalf("bare tag node not parsed: %#v", node)
	}
	if node := parseTagNode("weird:value"); node == nil || node.Value != "weird:value" {
		t.Fatalf("unknown tag node kind should round-trip as string: %#v", node)
	}
	if key, value, ok := cutEscaped(`novalue`, '='); ok || key != "novalue" || value != "" {
		t.Fatalf("cut without separator failed: %q %q %v", key, value, ok)
	}
	if encoded := encodeTagNodes([]*yaml.Node{
		{Kind: yaml.ScalarNode, Tag: "!!int", Value: "1"},
		{Kind: yaml.ScalarNode, Tag: "!!float", Value: "1.5"},
		{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"},
		{Kind: yaml.ScalarNode, Tag: "!!unknown", Value: "fallback"},
		nullNode(),
	}); encoded != "int:1|float:1.5|bool:true|str:fallback|null:" {
		t.Fatalf("typed tag nodes not encoded: %q", encoded)
	}
	if encoded := encodeTagNode(nil); encoded != "" {
		t.Fatalf("nil tag node should not encode: %q", encoded)
	}
	if encoded := encodeTagNode(stringNode("bad`tick")); encoded != "" {
		t.Fatalf("unsafe tag node should not encode: %q", encoded)
	}
	if key, value, ok := cutEscaped(`a\=b=c`, '='); !ok || key != `a\=b` || value != "c" {
		t.Fatalf("escaped cut failed: %q %q %v", key, value, ok)
	}
	if key, value, ok := cutEscaped(`a=b`, '='); !ok || key != "a" || value != "b" {
		t.Fatalf("plain cut failed: %q %q %v", key, value, ok)
	}
	var schema highbase.Schema
	applyIRBooleans(nil, nil)
	applyIRBooleans(&schema, &SchemaIR{ReadOnly: true, WriteOnly: true, Deprecated: true})
	if schema.ReadOnly == nil || schema.WriteOnly == nil || schema.Deprecated == nil {
		t.Fatalf("ir booleans not applied: %#v", schema)
	}
	NewGenerator().applyOpenAPIMetadata(nil, openAPIMetadata{Present: true})
	NewGenerator().applyOpenAPIMetadata(&SchemaIR{}, openAPIMetadata{})
	var tagged SchemaIR
	NewGenerator().applyOpenAPIMetadata(&tagged, openAPIMetadata{
		Present:             true,
		FormatSet:           true,
		Format:              "uuid",
		TitleSet:            true,
		Title:               "Title",
		DescriptionSet:      true,
		Description:         "Description",
		NullableSet:         true,
		Nullable:            true,
		ReadOnlySet:         true,
		WriteOnlySet:        true,
		DeprecatedSet:       true,
		MinimumSet:          true,
		Minimum:             1,
		MaximumSet:          true,
		Maximum:             10,
		ExclusiveMinimumSet: true,
		ExclusiveMinimum:    1,
		ExclusiveMaximumSet: true,
		ExclusiveMaximum:    10,
		MultipleOfSet:       true,
		MultipleOf:          0.5,
		MinLengthSet:        true,
		MinLength:           2,
		MaxLengthSet:        true,
		MaxLength:           8,
		PatternSet:          true,
		Pattern:             "^[a]$",
		MinItemsSet:         true,
		MinItems:            1,
		MaxItemsSet:         true,
		MaxItems:            3,
		UniqueItemsSet:      true,
		UniqueItems:         true,
		MinPropertiesSet:    true,
		MinProperties:       1,
		MaxPropertiesSet:    true,
		MaxProperties:       4,
		Enum:                []*yaml.Node{stringNode("a")},
		Const:               stringNode("a"),
	})
	if !tagged.Nullable || tagged.Format != "uuid" || tagged.SourceSchema == nil || tagged.SourceSchema.Minimum == nil {
		t.Fatalf("full metadata was not applied: %#v", tagged)
	}
	var falseTagged SchemaIR
	NewGenerator().applyOpenAPIMetadata(&falseTagged, openAPIMetadata{Present: true, ReadOnlySet: true, WriteOnlySet: true, DeprecatedSet: true})
	if falseTagged.ReadOnly || falseTagged.WriteOnly || falseTagged.Deprecated {
		t.Fatalf("false boolean metadata should stay false: %#v", tagged)
	}
	if cloneIR(nil) != nil {
		t.Fatal("nil clone should stay nil")
	}
	cloned := cloneIR(&SchemaIR{SourceSchema: &highbase.Schema{Format: "uuid"}})
	if cloned.SourceSchema == nil || cloned.SourceSchema.Format != "uuid" {
		t.Fatalf("source schema clone failed: %#v", cloned)
	}
	var emptySchemaGen Generator
	if schema := emptySchemaGen.fieldSchema(reflect.TypeOf(MetadataFieldOverride{}), reflect.StructField{Name: "Missing"}, "missing"); schema != nil {
		t.Fatalf("empty field schema registry should miss: %#v", schema)
	}
	emptySchemaGen.fieldSchemas = map[fieldSchemaKey]*highbase.SchemaProxy{}
	if schema := emptySchemaGen.fieldSchema(reflect.TypeOf(MetadataFieldOverride{}), reflect.StructField{Name: "Missing"}, "missing"); schema != nil {
		t.Fatalf("field schema registry without json registry should miss: %#v", schema)
	}
	if _, err := NewGenerator().irFromFieldSchema(reflect.TypeOf((*string)(nil)), "Broken", "Broken", nil); err == nil {
		t.Fatal("nil field schema should fail")
	}
	if ir, err := NewGenerator().irFromFieldSchema(reflect.TypeOf((*string)(nil)), "String", "String", highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"string"}})); err != nil || !ir.Nullable {
		t.Fatalf("pointer field schema should be nullable: %#v %v", ir, err)
	}
	var bare Generator
	WithOpenAPITags(true)(&bare)
	WithSchemaMetadataSidecar(true)(&bare)
	WithFieldSchema(nil, "Field", highbase.CreateSchemaProxy(&highbase.Schema{}))(&bare)
	WithFieldSchema(reflect.TypeOf(MetadataFieldOverride{}), "", highbase.CreateSchemaProxy(&highbase.Schema{}))(&bare)
	WithFieldSchema(reflect.TypeOf(MetadataFieldOverride{}), "Field", nil)(&bare)
	WithFieldSchema(reflect.TypeOf(MetadataFieldOverride{}), "Field", highbase.CreateSchemaProxy(&highbase.Schema{}))(&bare)
	WithFieldSchemaByJSONName(nil, "field", highbase.CreateSchemaProxy(&highbase.Schema{}))(&bare)
	WithFieldSchemaByJSONName(reflect.TypeOf(MetadataFieldOverride{}), "", highbase.CreateSchemaProxy(&highbase.Schema{}))(&bare)
	WithFieldSchemaByJSONName(reflect.TypeOf(MetadataFieldOverride{}), "field", nil)(&bare)
	WithFieldSchemaByJSONName(reflect.TypeOf(MetadataFieldOverride{}), "field", highbase.CreateSchemaProxy(&highbase.Schema{}))(&bare)
	if !bare.openapiTags || !bare.schemaMetadataSidecar || bare.fieldSchemas == nil || bare.jsonSchemas == nil {
		t.Fatalf("metadata options did not initialize bare generator: %#v", bare)
	}
	providerGen := NewGenerator()
	providerGen.recordSchemaMetadata("", &highbase.Schema{Type: []string{"string"}})
	providerGen.schemaMetadataSidecar = true
	providerGen.recordSchemaMetadata("NoSchema", nil)
	providerGen.recordSchemaMetadata("Sample", &highbase.Schema{Type: []string{"string"}})
	providerGen.recordSchemaMetadata("Sample", &highbase.Schema{Type: []string{"string"}})
	sidecarDecl := providerGen.renderSchemaMetadataSidecarDecl()
	if sidecarDecl == "" {
		t.Fatal("expected metadata sidecar declaration")
	}
	if !strings.Contains(sidecarDecl, "OpenAPISchemaMetadata") {
		t.Fatalf("metadata sidecar was not rendered: %s", sidecarDecl)
	}
	emptySidecar := NewGenerator(WithSchemaMetadataSidecar(true))
	if emptySidecar.renderSchemaMetadataSidecarDecl() != "" {
		t.Fatal("empty metadata sidecar should not render")
	}
	if _, err := schemaProxyFromProviderMetadata(nil); err == nil {
		t.Fatal("nil schema metadata should fail")
	}
	if schemaProxyFromMetadata(nil) != nil || schemaFromMetadata(nil) != nil {
		t.Fatal("nil schema metadata should stay nil")
	}
	if schemaMetadataHasSiblings(nil) || !schemaMetadataEmpty(nil) {
		t.Fatal("nil schema metadata helper mismatch")
	}
	if pureRef := schemaProxyFromMetadata(&providerSchemaMetadata{Ref: "#/components/schemas/Target"}); pureRef == nil || !pureRef.IsReference() || pureRef.Schema() != nil {
		t.Fatalf("pure ref metadata should not create siblings: %#v", pureRef)
	}
	if dynamicBoolNumberFromMetadata(&providerDynamicBoolNumber{}) != nil {
		t.Fatal("empty dynamic bool/number should stay nil")
	}
	if dynamicSchemaBoolFromMetadata(nil) != nil {
		t.Fatal("nil dynamic schema/bool should stay nil")
	}
	if yamlKindFromMetadata("unknown") != yaml.ScalarNode || metadataYAMLKind(yaml.Kind(99)) != "scalar" {
		t.Fatal("unknown yaml kinds should default to scalar")
	}
	for _, kind := range []yaml.Kind{yaml.DocumentNode, yaml.SequenceNode, yaml.MappingNode, yaml.AliasNode} {
		if metadataYAMLKind(kind) == "scalar" {
			t.Fatalf("yaml kind %v should not render as scalar", kind)
		}
	}
	if metadataIndent(0) != "" {
		t.Fatal("zero metadata indent should be empty")
	}
	writerGen := NewGenerator()
	if got := writerGen.schemaMetadataLiteral(nil, 0); got != "nil" {
		t.Fatalf("nil schema literal mismatch: %q", got)
	}
	if got := writerGen.schemaProxyMetadataLiteral(nil, 0); got != "nil" {
		t.Fatalf("nil schema proxy literal mismatch: %q", got)
	}
	refWithSibling := highbase.CreateSchemaProxyRefWithSchema("#/components/schemas/Target", &highbase.Schema{
		Description: "sibling",
		ReadOnly:    boolPtr(true),
	})
	refSiblingLiteral := writerGen.schemaProxyMetadataLiteral(refWithSibling, 0)
	for _, want := range []string{`Ref: "#/components/schemas/Target"`, `Description: "sibling"`, `ReadOnly: &openAPIBool{Value: true}`} {
		if !strings.Contains(refSiblingLiteral, want) {
			t.Fatalf("ref sibling metadata literal missing %q in %s", want, refSiblingLiteral)
		}
	}
	if schema := referenceSiblingMetadataSchema(highbase.CreateSchemaProxyRef("#/components/schemas/Target")); schema != nil {
		t.Fatalf("pure programmatic ref should not expose sibling metadata: %#v", schema)
	}
	if referenceSiblingMetadataSchema(nil) != nil || referenceSiblingMetadataSchema(highbase.CreateSchemaProxy(&highbase.Schema{})) != nil {
		t.Fatal("nil and non-reference proxies should not expose sibling metadata")
	}
	if schemaFromReferenceSiblingNode(nil) != nil {
		t.Fatal("nil reference sibling node should not render metadata")
	}
	lowRefWithSibling := schemaProxyFromRefDocumentYAML(t, "$ref: '#/components/schemas/Target'\ndescription: low sibling\n")
	if schema := referenceSiblingMetadataSchema(lowRefWithSibling); schema == nil || schema.Description != "low sibling" {
		t.Fatalf("low-level ref sibling metadata should be detected: %#v", schema)
	}
	plainLowRef := schemaProxyFromRefDocumentYAML(t, "$ref: '#/components/schemas/Target'\n")
	if schema := referenceSiblingMetadataSchema(plainLowRef); schema != nil {
		t.Fatalf("plain low-level ref should not expose sibling metadata: %#v", schema)
	}
	if got := writerGen.schemaSliceMetadataLiteral([]*highbase.SchemaProxy{nil}, 0); !strings.Contains(got, "nil") {
		t.Fatalf("nil schema slice literal mismatch: %q", got)
	}
	nilMap := orderedmap.New[string, *highbase.SchemaProxy]()
	nilMap.Set("nil", nil)
	if got := writerGen.schemaMapMetadataLiteral(nilMap, 0); !strings.Contains(got, "nil") {
		t.Fatalf("nil schema map literal mismatch: %q", got)
	}
	if metadataStringStringMapLiteral(nil, 0) != "" || metadataPlainIntLiteral(1) != "1" {
		t.Fatal("metadata literal helper fallback mismatch")
	}
	if stringStringMapFromMetadata(nil) != nil {
		t.Fatal("nil string-string metadata map should stay nil")
	}
	if got := metadataYAMLNodeLiteral(nil, 0); got != "nil" {
		t.Fatalf("nil yaml node literal mismatch: %q", got)
	}
	if got := metadataYAMLNodeLiteral(&yaml.Node{Kind: yaml.ScalarNode, Value: "1"}, 0); !strings.Contains(got, `Tag: "!!int"`) {
		t.Fatalf("empty integer scalar tag should infer numeric metadata literal: %s", got)
	}
	if got := metadataYAMLNodeLiteral(&yaml.Node{Kind: yaml.ScalarNode, Value: "true"}, 0); !strings.Contains(got, `Tag: "!!bool"`) {
		t.Fatalf("empty boolean scalar tag should infer boolean metadata literal: %s", got)
	}
	if got := metadataYAMLNodeLiteral(&yaml.Node{Kind: yaml.ScalarNode, Style: yaml.SingleQuotedStyle, Value: "1"}, 0); !strings.Contains(got, `Tag: "!!str"`) {
		t.Fatalf("styled scalar tag should stay string in metadata literal: %s", got)
	}
	if got := metadataYAMLNodeLiteral(&yaml.Node{Kind: yaml.MappingNode}, 0); !strings.Contains(got, `Tag: "!!map"`) {
		t.Fatalf("empty mapping tag should normalize to default in metadata literal: %s", got)
	}
}

func schemaProxyFromRefDocumentYAML(t *testing.T, sampleYAML string) *highbase.SchemaProxy {
	t.Helper()
	spec := []byte("openapi: 3.1.0\ninfo:\n  title: Test\n  version: 1.0.0\npaths: {}\ncomponents:\n  schemas:\n    Target:\n      type: string\n    Sample:\n" + indent(sampleYAML, "      "))
	config := datamodel.NewDocumentConfiguration()
	config.TransformSiblingRefs = false
	doc, err := libopenapi.NewDocumentWithConfiguration(spec, config)
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
