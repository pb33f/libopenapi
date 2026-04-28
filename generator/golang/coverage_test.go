// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"reflect"
	"strings"
	"testing"

	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

func TestInternalBranchCoverage(t *testing.T) {
	gen := NewGenerator(
		WithNullableAsPointer(false),
		WithEnumConstants(true),
		WithNameResolver(func(name string) string {
			if name == "Custom" {
				return "Resolved"
			}
			return ""
		}),
	)
	if got := gen.publicName("Custom"); got != "Resolved" {
		t.Fatalf("resolver not used: %s", got)
	}
	if got := toPublicName(""); got != "Value" {
		t.Fatalf("empty public name: %s", got)
	}
	if got := toPublicName("type"); got != "Type" {
		t.Fatalf("keyword public name: %s", got)
	}
	if got := gen.nestedTypeName("", "child value"); got != "ChildValue" {
		t.Fatalf("empty parent nested name: %s", got)
	}
	if names := gen.resolveComponentTypeNames(nil); len(names) != 0 {
		t.Fatalf("nil component name map should be empty: %#v", names)
	}
	componentNameProbe := orderedmap.New[string, *highbase.SchemaProxy]()
	componentNameProbe.Set("component", highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"string"}}))
	if names := gen.resolveComponentTypeNames(componentNameProbe); names["component"] != "Component" {
		t.Fatalf("component name map should resolve without an active registry: %#v", names)
	}
	if got := refName("Pet"); got != "Pet" {
		t.Fatalf("plain ref: %s", got)
	}
	if got := refName(""); got != "" {
		t.Fatalf("empty ref: %s", got)
	}
	if got := refName("#/"); got != "#/" {
		t.Fatalf("trailing ref: %s", got)
	}
	if splitCamel("") != nil {
		t.Fatal("empty camel split should be nil")
	}
	if got := uniqueName("", map[string]struct{}{}); got != "Value" {
		t.Fatalf("empty unique name: %s", got)
	}
	if derefType(nil) != nil {
		t.Fatal("nil deref should stay nil")
	}
	if got := typeName(nil); got != "" {
		t.Fatalf("nil type name: %s", got)
	}
	if got := typeName(reflect.TypeOf([]string{})); got != "Slice" {
		t.Fatalf("unnamed type name: %s", got)
	}
	if interfaceKey(nil) != nil || interfaceKey(struct{}{}) != nil {
		t.Fatal("bad interface keys should be nil")
	}
	var iface any
	if interfaceKey(&iface) == nil {
		t.Fatal("interface pointer key should resolve")
	}
	if got := derefType(reflect.TypeOf((**string)(nil))); got.Kind() != reflect.String {
		t.Fatalf("pointer deref failed: %v", got)
	}
	if isRequired(nil, "x") {
		t.Fatal("nil required should be false")
	}

	var bare Generator
	WithFormatMapping("date", "civil.Date", "civil")(&bare)
	if bare.formatMappings["date"].goType != "civil.Date" {
		t.Fatal("format mapping should initialize nil map")
	}
	WithAdditionalPropertiesMethods(false)(&bare)
	if bare.additionalPropertiesMethods {
		t.Fatal("additional properties methods option not applied")
	}
	WithExternalRefTypeResolver(func(ref string) string { return "ResolvedExternal" })(&bare)
	if bare.refTypeName("../common.yaml#/components/schemas/Pet") != "ResolvedExternal" {
		t.Fatal("external ref resolver not applied")
	}
	if bare.refTypeName("") != "" {
		t.Fatal("empty ref type should stay empty")
	}
	bare.externalRefResolver = nil
	if bare.refTypeName("AlreadyNamed") != "AlreadyNamed" {
		t.Fatal("plain ref type should stay unchanged")
	}
	WithTypeSchema(nil, highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"string"}}))(&bare)
	WithTypeSchema(reflect.TypeOf(""), nil)(&bare)
	WithTypeSchema(reflect.TypeOf(""), highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"string"}}))(&bare)
	if bare.typeSchemas[reflect.TypeOf("")] == nil {
		t.Fatal("type schema option should initialize nil map")
	}
	WithOneOfTypes(struct{}{}, nil)(&bare)
	WithDiscriminatorMapping(struct{}{}, "kind", map[string]string{"x": "Y"})(&bare)

	tag := parseJSONTag(reflect.StructField{Name: "Value", Tag: `json:",omitempty"`})
	if tag.name != "Value" || !tag.omitempty {
		t.Fatalf("unexpected empty-name tag: %#v", tag)
	}
	tag = parseJSONTag(reflect.StructField{Name: "Value", Tag: `json:"-"`})
	if !tag.skip {
		t.Fatal("skip tag not parsed")
	}
	if tagLiteral("x", false, false, false, false, "") != "" {
		t.Fatal("expected empty literal")
	}
	if quoted := strconvQuote(`a"b\c`); quoted != `"a\"b\\c"` {
		t.Fatalf("bad quote: %s", quoted)
	}
	if _, err := NewGenerator().RenderSchemas(nil); err != nil {
		t.Fatal(err)
	}
	if _, err := (&Generator{packageName: "bad-name"}).renderFile(nil); err == nil {
		t.Fatal("expected direct renderFile package error")
	}
	if _, err := NewGenerator().renderFile([]*SchemaIR{nil}); err != nil {
		t.Fatal(err)
	}
	if _, err := NewGenerator().SchemaFromValue(nil); err == nil {
		t.Fatal("expected generator nil value error")
	}
	if _, err := NewGenerator().SchemaFromValue("hello"); err != nil {
		t.Fatal(err)
	}
	if _, err := NewGenerator().RenderSchema("Empty", &highbase.SchemaProxy{}); err == nil {
		t.Fatal("expected render schema openapi error")
	}
	badNameGen := NewGenerator(WithNameResolver(func(string) string { return "bad-name" }))
	if _, err := badNameGen.RenderSchema("Bad", highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"object"}})); err == nil {
		t.Fatal("expected formatting error from invalid resolved name")
	}
	nilSchemas := orderedmap.New[string, *highbase.SchemaProxy]()
	nilSchemas.Set("Broken", nil)
	if _, err := NewGenerator().RenderSchemas(nilSchemas); err == nil {
		t.Fatal("expected render schemas error")
	}

	gen.renderDecl(nil)
	gen.renderDecl(&SchemaIR{Kind: KindRef, Name: "Ref", Ref: "#/components/schemas/Ref"})
	if !gen.rememberDecl("Once") || gen.rememberDecl("Once") || gen.rememberDecl("") {
		t.Fatal("rememberDecl branch failed")
	}
	gen.renderAliasDecl(&SchemaIR{Name: "Alias", Kind: KindString})
	gen.renderAliasDecl(&SchemaIR{Name: "Alias", Kind: KindString})
	gen.renderObjectDecl(&SchemaIR{
		Name:                 "MapOnly",
		Kind:                 KindObject,
		AdditionalProperties: &SchemaIR{Kind: KindString},
	})
	gen.renderObjectDecl(&SchemaIR{Name: "MapOnly", Kind: KindObject})
	gen.renderObjectDecl(&SchemaIR{
		Name:       "Embedded",
		Kind:       KindObject,
		Properties: orderedmap.New[string, *SchemaIR](),
		AllOf:      []*SchemaIR{{Kind: KindRef, Ref: "#/components/schemas/Base", Name: "Base"}},
	})
	gen.renderChildren(&SchemaIR{
		Items:                &SchemaIR{Name: "ChildItem", Kind: KindString},
		AdditionalProperties: &SchemaIR{Name: "ChildAdditional", Kind: KindString},
		AllOf:                []*SchemaIR{{Name: "ChildAllOf", Kind: KindString}},
	})
	gen.renderNested(nil)
	gen.renderNested(&SchemaIR{Kind: KindArray, Items: &SchemaIR{Name: "NestedAlias", Kind: KindInteger}})
	gen.renderNested(&SchemaIR{Kind: KindMap, AdditionalProperties: &SchemaIR{Name: "NestedMapAlias", Kind: KindBoolean}})
	gen.renderUnionDecl(&SchemaIR{Name: "BrokenUnion", Kind: KindUnion})
	gen.renderDiscriminatedUnion(&SchemaIR{Name: "BrokenDisc", Kind: KindUnion, Union: &UnionIR{}})
	gen.renderRawUnion(&SchemaIR{Name: "BrokenDisc"})
	gen.renderDiscriminatedUnion(&SchemaIR{
		Name: "DiscWithNilVariant",
		Kind: KindUnion,
		Union: &UnionIR{
			Discriminator: &Discriminator{PropertyName: "kind", Mapping: map[string]string{"x": "X"}},
			Variants:      []*SchemaIR{nil, {Name: "", Kind: KindObject}},
		},
	})
	gen.renderDiscriminatedUnion(&SchemaIR{
		Name: "DiscWithNilVariant",
		Kind: KindUnion,
		Union: &UnionIR{
			Discriminator: &Discriminator{PropertyName: "kind", Mapping: map[string]string{"x": "X"}},
		},
	})

	enum := &SchemaIR{
		Name: "IntEnum",
		Kind: KindEnum,
		Enum: []*yaml.Node{{Kind: yaml.ScalarNode, Tag: "!!int", Value: "1"}},
	}
	gen.renderEnumDecl(enum)
	gen.renderEnumDecl(enum)
	gen.renderEnumDecl(&SchemaIR{
		Name: "StringEnum",
		Kind: KindEnum,
		Enum: []*yaml.Node{stringNode("hello-world")},
	})
	if got := gen.goType(nil, true, false); got != "any" {
		t.Fatalf("nil type: %s", got)
	}
	if got := gen.goType(&SchemaIR{Kind: KindRef, Ref: "#/components/schemas/Pet"}, true, false); got != "Pet" {
		t.Fatalf("ref type: %s", got)
	}
	gen.componentKinds = map[string]Kind{"Choice": KindUnion}
	if got := gen.goType(&SchemaIR{Kind: KindRef, Name: "Choice", Ref: "#/components/schemas/Choice"}, true, false); got != "ChoiceUnion" {
		t.Fatalf("union ref type: %s", got)
	}
	gen.componentKinds = nil
	if got := gen.goType(&SchemaIR{Kind: KindObject, Name: "Obj", Properties: orderedmap.New[string, *SchemaIR]()}, true, false); got != "map[string]any" {
		t.Fatalf("empty object type: %s", got)
	}
	closed := false
	if got := gen.goType(&SchemaIR{Kind: KindObject, Name: "ClosedObj", AdditionalAllowed: &closed}, true, false); got != "ClosedObj" {
		t.Fatalf("closed object type: %s", got)
	}
	props := orderedmap.New[string, *SchemaIR]()
	props.Set("id", &SchemaIR{Kind: KindString})
	if got := gen.goType(&SchemaIR{Kind: KindObject, Name: "Obj", Properties: props}, true, false); got != "Obj" {
		t.Fatalf("object type: %s", got)
	}
	if got := gen.goType(&SchemaIR{Kind: KindObject, AdditionalProperties: &SchemaIR{Kind: KindInteger}}, true, false); got != "map[string]int" {
		t.Fatalf("additional object type: %s", got)
	}
	if got := gen.goType(&SchemaIR{Kind: KindArray, Items: &SchemaIR{Kind: KindString}}, true, false); got != "[]string" {
		t.Fatalf("array type: %s", got)
	}
	if got := gen.goType(&SchemaIR{Kind: KindMap, AdditionalProperties: &SchemaIR{Kind: KindString}}, true, false); got != "map[string]string" {
		t.Fatalf("map type: %s", got)
	}
	if got := gen.goType(&SchemaIR{Kind: KindInteger, Format: "int32"}, true, false); got != "int32" {
		t.Fatalf("int32 type: %s", got)
	}
	if got := gen.goType(&SchemaIR{Kind: KindInteger, Format: "int64"}, true, false); got != "int64" {
		t.Fatalf("int64 type: %s", got)
	}
	if got := gen.goType(&SchemaIR{Kind: KindNumber, Format: "float"}, true, false); got != "float32" {
		t.Fatalf("float type: %s", got)
	}
	if got := gen.goType(&SchemaIR{Kind: KindNumber}, true, false); got != "float64" {
		t.Fatalf("number type: %s", got)
	}
	if got := gen.goType(&SchemaIR{Kind: KindEnum}, true, false); got != "string" {
		t.Fatalf("unnamed enum type: %s", got)
	}
	if got := gen.goType(&SchemaIR{Kind: KindUnion, Name: "Choice"}, true, false); got != "ChoiceUnion" {
		t.Fatalf("union type: %s", got)
	}
	if got := gen.goType(&SchemaIR{Kind: KindUnknown}, true, false); got != "any" {
		t.Fatalf("unknown type: %s", got)
	}
	if got := gen.goType(&SchemaIR{Kind: KindBoolean}, true, true); got != "bool" {
		t.Fatalf("nullable disabled pointer type: %s", got)
	}
	gen.formatMappings["date-time"] = formatMapping{goType: "time.Time", importPath: "time"}
	if got := gen.formatType("date-time", "string"); got != "time.Time" {
		t.Fatalf("mapped format: %s", got)
	}
	if got := gen.formatType("unknown", "string"); got != "string" {
		t.Fatalf("fallback format: %s", got)
	}
	if shouldPointer("[]string", nil, false, true, true) {
		t.Fatal("slices should not be pointered")
	}
	if !shouldPointer("string", &SchemaIR{Nullable: true}, true, true, true) {
		t.Fatal("nullable should pointer")
	}
	var comment strings.Builder
	writeComment(&comment, "Thing", "")
	writeComment(&comment, "Thing", "\n")
	writeComment(&comment, "Thing", "already.")
	writeComment(&comment, "Thing", "missing")
	if !strings.Contains(comment.String(), "already.") || !strings.Contains(comment.String(), "missing.") {
		t.Fatal("comment not written")
	}
	if gen.stringEncodedIR(nil, "nil") != nil {
		t.Fatal("nil string encoded IR should stay nil")
	}
	unsupportedStringEncoded := &SchemaIR{Kind: KindArray}
	if got := gen.stringEncodedIR(unsupportedStringEncoded, "array"); got != unsupportedStringEncoded {
		t.Fatal("unsupported string encoded IR should return original")
	}

	if shape := enumShapeFor(nil); shape.goType != "any" || shape.constants {
		t.Fatalf("nil enum shape should be any without constants: %#v", shape)
	}
	if shape := enumShapeFor([]*yaml.Node{{Kind: yaml.ScalarNode, Tag: "!!int", Value: "1"}, {Kind: yaml.ScalarNode, Tag: "!!float", Value: "1.5"}}); shape.goType != "float64" || !shape.constants {
		t.Fatalf("numeric enum shape should widen to float64: %#v", shape)
	}
	if shape := enumShapeFor([]*yaml.Node{{Kind: yaml.ScalarNode, Tag: "!!binary", Value: "abc"}}); shape.goType != "string" || !shape.constants {
		t.Fatalf("unknown scalar enum shape should fall back to string: %#v", shape)
	}
	if enumLiteral(nil, "string") != "" || enumLiteral(stringNode("x"), "any") != "" {
		t.Fatal("enum literal should skip nil and unsupported bases")
	}
	for typ, kind := range map[string]Kind{
		"string":  KindString,
		"integer": KindInteger,
		"number":  KindNumber,
		"boolean": KindBoolean,
		"array":   KindArray,
		"object":  KindObject,
		"unknown": KindAny,
	} {
		if got := kindForJSONType(typ); got != kind {
			t.Fatalf("kind for %s: %v", typ, got)
		}
	}
	if schemaDeclaresType(nil) {
		t.Fatal("nil schema should not declare a type")
	}
}

func TestOpenAPIBranchCoverage(t *testing.T) {
	gen := NewGenerator()
	ref := highbase.CreateSchemaProxyRef("#/components/schemas/Pet")
	if ir, err := gen.irFromOpenAPI("Pet", ref, "Pet"); err != nil || ir.Kind != KindRef {
		t.Fatalf("ref ir failed: %#v %v", ir, err)
	}
	if ir, err := gen.irFromOpenAPI("Pet", ref, "Pet"); err != nil || ir.Kind != KindRef {
		t.Fatalf("cached ref ir failed: %#v %v", ir, err)
	}
	if _, err := gen.irFromOpenAPI("Nil", nil, "Nil"); err == nil {
		t.Fatal("expected nil openapi schema error")
	}
	if _, err := gen.irFromOpenAPI("Empty", &highbase.SchemaProxy{}, "Empty"); err == nil {
		t.Fatal("expected empty proxy schema error")
	}
	nullable := true
	if ir, err := gen.irFromOpenAPI("Nullable", highbase.CreateSchemaProxy(&highbase.Schema{
		Type:     []string{"string"},
		Nullable: &nullable,
	}), "Nullable"); err != nil || !ir.Nullable {
		t.Fatalf("nullable schema failed: %#v %v", ir, err)
	}

	schemas := map[string]string{
		"String":      "type: string\n",
		"Integer":     "type: integer\n",
		"Number":      "type: number\n",
		"Boolean":     "type: boolean\n",
		"Array":       "type: array\nitems: true\n",
		"ArraySchema": "type: array\nitems:\n  type: string\n",
		"Free":        "type: object\nadditionalProperties: true\n",
		"Closed":      "type: object\nadditionalProperties: false\n",
		"MapSchema":   "type: object\nadditionalProperties:\n  type: string\n",
		"InferObject": "properties:\n  id:\n    type: string\n",
		"InferMap":    "additionalProperties: true\n",
	}
	for name, yml := range schemas {
		if _, err := gen.irFromOpenAPI(name, schemaProxyFromYAML(t, yml), name); err != nil {
			t.Fatalf("%s failed: %v", name, err)
		}
	}
	unknownIR := &SchemaIR{Name: "Unknown", Required: make(map[string]struct{})}
	gen.populateSchemaShape(unknownIR, &highbase.Schema{Type: []string{"unknown"}}, "unknown")
	if unknownIR.Kind != KindAny {
		t.Fatalf("unknown explicit type should render as any: %#v", unknownIR)
	}
	unknownObjectIR := &SchemaIR{Name: "UnknownObject", Required: make(map[string]struct{})}
	unknownObjectProps := orderedmap.New[string, *highbase.SchemaProxy]()
	unknownObjectProps.Set("id", highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"string"}}))
	gen.populateSchemaShape(unknownObjectIR, &highbase.Schema{Type: []string{"unknown"}, Properties: unknownObjectProps}, "unknownObject")
	if unknownObjectIR.Kind != KindObject {
		t.Fatalf("unknown explicit type with properties should render as object: %#v", unknownObjectIR)
	}
	unknownMapIR := &SchemaIR{Name: "UnknownMap", Required: make(map[string]struct{})}
	gen.populateSchemaShape(unknownMapIR, &highbase.Schema{
		Type: []string{"unknown"},
		AdditionalProperties: &highbase.DynamicValue[*highbase.SchemaProxy, bool]{
			N: 1,
			B: true,
		},
	}, "unknownMap")
	if unknownMapIR.Kind != KindObject {
		t.Fatalf("unknown explicit type with additionalProperties should render as object: %#v", unknownMapIR)
	}

	merged := &SchemaIR{
		Name: "Merged",
		AllOf: []*SchemaIR{
			nil,
			{Kind: KindRef, Ref: "#/components/schemas/Base", Name: "Base"},
			{Kind: KindString, Name: "Ignored"},
			{Kind: KindObject, Properties: orderedmap.New[string, *SchemaIR](), Required: map[string]struct{}{"id": {}}},
		},
	}
	merged.AllOf[3].Properties.Set("id", &SchemaIR{Kind: KindString})
	gen.mergeAllOf(merged)
	if merged.Properties.Len() != 1 || len(merged.AllOf) != 2 {
		t.Fatalf("bad allOf merge: %#v", merged)
	}

	nullableAny := true
	if len(nonNullVariants([]*SchemaIR{
		nil,
		{Kind: KindAny, Nullable: true, SourceSchema: &highbase.Schema{Nullable: &nullableAny}},
		{Kind: KindAny, Nullable: true, SourceSchema: &highbase.Schema{Type: []string{"null"}}},
		{Kind: KindAny, Const: nullNode()},
		{Kind: KindEnum, Enum: []*yaml.Node{nullNode()}},
		{Kind: KindString},
	})) != 2 {
		t.Fatal("nonNullVariants should only remove null-only variants")
	}
	if isNullOnlyIR(nil) || schemaOnlyAllowsNull(nil) {
		t.Fatal("nil null-only checks should be false")
	}
	if isNullOnlyIR(&SchemaIR{Const: stringNode("x")}) {
		t.Fatal("non-null const should not be null-only")
	}
	if schemaOnlyAllowsNull(&highbase.Schema{Enum: []*yaml.Node{stringNode("x")}}) {
		t.Fatal("non-null enum should not be null-only")
	}
	if schemaNeedsNullAlternative(nil) || schemaNeedsNullAlternative(&highbase.Schema{Type: []string{"string"}}) {
		t.Fatal("plain schemas should not need a null alternative wrapper")
	}
	if inferConstDiscriminator(nil) != nil {
		t.Fatal("nil variants should not infer discriminator")
	}
	if inferConstDiscriminator([]*SchemaIR{{Kind: KindObject}}) != nil {
		t.Fatal("missing properties should not infer")
	}
	props := orderedmap.New[string, *SchemaIR]()
	props.Set("kind", &SchemaIR{Const: stringNode("same")})
	if inferConstDiscriminator([]*SchemaIR{
		{Kind: KindObject, Name: "A", Properties: props},
		{Kind: KindObject, Name: "B", Properties: props},
	}) != nil {
		t.Fatal("duplicate discriminator values should not infer")
	}
	propsA := orderedmap.New[string, *SchemaIR]()
	propsA.Set("kind", &SchemaIR{Const: &yaml.Node{Kind: yaml.SequenceNode}})
	if inferConstDiscriminator([]*SchemaIR{{Kind: KindObject, Name: "A", Properties: propsA}}) != nil {
		t.Fatal("non-scalar const should not infer")
	}
	propsB := orderedmap.New[string, *SchemaIR]()
	propsB.Set("other", &SchemaIR{Const: stringNode("b")})
	if inferConstDiscriminator([]*SchemaIR{
		{Kind: KindObject, Name: "A", Properties: props},
		{Kind: KindObject, Name: "B", Properties: propsB},
	}) != nil {
		t.Fatal("missing discriminator on later variant should not infer")
	}
}

func TestReflectBranchCoverage(t *testing.T) {
	gen := NewGenerator()
	if _, err := gen.irFromReflect(nil, "", "nil"); err == nil {
		t.Fatal("expected nil reflect type error")
	}
	type Recursive struct {
		Next *Recursive `json:"next,omitempty"`
	}
	if _, err := gen.irFromReflect(reflect.TypeOf(Recursive{}), "Recursive", "Recursive"); err != nil {
		t.Fatal(err)
	}
	type Numbers struct {
		Uint    uint    `json:"uint"`
		Uint8   uint8   `json:"uint8"`
		Bool    bool    `json:"bool"`
		Int32   int32   `json:"int32"`
		Int64   int64   `json:"int64"`
		Float32 float32 `json:"float32"`
		Float64 float64 `json:"float64"`
		Bytes   []byte  `json:"bytes"`
	}
	if _, err := gen.irFromReflect(reflect.TypeOf(Numbers{}), "Numbers", "Numbers"); err != nil {
		t.Fatal(err)
	}
	if _, err := gen.irFromReflect(reflect.TypeOf(&Numbers{}), "Numbers", "Numbers"); err != nil {
		t.Fatal(err)
	}
	providerIR, err := gen.irFromReflect(reflect.TypeOf(&Provider{}), "Provider", "Provider")
	if err != nil {
		t.Fatal(err)
	}
	if !providerIR.Nullable {
		t.Fatal("pointer schema provider should preserve nullable")
	}
	type WithPrivate struct {
		name string
		ID   string `json:"id"`
	}
	if _, err := gen.irFromReflect(reflect.TypeOf(WithPrivate{}), "WithPrivate", "WithPrivate"); err != nil {
		t.Fatal(err)
	}
	type BrokenInterface interface{ broken() }
	brokenGen := NewGenerator(WithOneOfTypes((*BrokenInterface)(nil), make(chan string)))
	if _, err := brokenGen.irFromReflect(reflect.TypeOf((*BrokenInterface)(nil)).Elem(), "BrokenInterface", "BrokenInterface"); err == nil {
		t.Fatal("expected broken interface variant error")
	}
	type ProviderLikeInterface interface {
		OpenAPISchema() *highbase.SchemaProxy
	}
	if _, err := gen.irFromReflect(reflect.TypeOf((*ProviderLikeInterface)(nil)).Elem(), "ProviderLikeInterface", "ProviderLikeInterface"); err == nil {
		t.Fatal("provider-like interfaces should fail cleanly unless registered as oneOf")
	}
	type BadSlice []chan string
	if _, err := gen.irFromReflect(reflect.TypeOf(BadSlice{}), "BadSlice", "BadSlice"); err == nil {
		t.Fatal("expected bad slice error")
	}
	type BadMap map[string]chan string
	if _, err := gen.irFromReflect(reflect.TypeOf(BadMap{}), "BadMap", "BadMap"); err == nil {
		t.Fatal("expected bad map value error")
	}
	if _, err := gen.irFromReflect(reflect.TypeOf(make(chan string)), "Chan", "Chan"); err == nil {
		t.Fatal("expected unsupported chan")
	}
}

func TestToOpenAPIBranchCoverage(t *testing.T) {
	gen := NewGenerator()
	applySchemaFidelity(nil, nil)
	fidelitySchema := &highbase.Schema{}
	applySchemaFidelity(fidelitySchema, &SchemaIR{SourceSchema: &highbase.Schema{DynamicRef: "#dynamic"}})
	if fidelitySchema.DynamicRef != "#dynamic" {
		t.Fatalf("dynamic ref fidelity not applied: %#v", fidelitySchema)
	}
	gen.populateOpenAPIUnion(&highbase.Schema{}, &SchemaIR{})
	falseValue := false
	obj := &SchemaIR{
		Kind:              KindObject,
		AdditionalAllowed: &falseValue,
	}
	if out := gen.openapiFromIR(obj); out == nil {
		t.Fatal("expected object schema")
	}
	union := &SchemaIR{
		Kind: KindUnion,
		Union: &UnionIR{
			Kind: UnionAnyOf,
			Variants: []*SchemaIR{
				{Kind: KindString},
				{Kind: KindInteger},
			},
		},
	}
	if out := gen.openapiFromIR(union); out == nil {
		t.Fatal("expected union schema")
	}
	emptyUnion := &SchemaIR{Kind: KindUnion, Union: &UnionIR{}}
	if out := gen.openapiFromIR(emptyUnion); out == nil {
		t.Fatal("expected empty union schema")
	}
	nullableDynamic := gen.openapiFromIR(&SchemaIR{
		Kind:       KindRef,
		Ref:        "#dynamic",
		DynamicRef: true,
		Nullable:   true,
		SourceSchema: &highbase.Schema{
			DynamicRef: "#dynamic",
			Comment:    "dynamic nullable ref",
		},
	}).Schema()
	if nullableDynamic == nil || len(nullableDynamic.AnyOf) != 2 || nullableDynamic.AnyOf[0].Schema().DynamicRef != "#dynamic" {
		t.Fatalf("expected nullable dynamic ref anyOf, got %#v", nullableDynamic)
	}
	nullableEnum := gen.openapiFromIR(&SchemaIR{
		Kind:     KindEnum,
		Nullable: true,
		Enum:     []*yaml.Node{stringNode("active")},
	}).Schema()
	if nullableEnum == nil || !schemaTypeContains(nullableEnum.Type, "null") || !enumHasNull(nullableEnum.Enum) {
		t.Fatalf("expected nullable enum to include null type and enum value, got %#v", nullableEnum)
	}
	for _, enumIR := range []*SchemaIR{
		{Kind: KindEnum, Enum: []*yaml.Node{{Kind: yaml.ScalarNode, Tag: "!!int", Value: "1"}}},
		{Kind: KindEnum, Enum: []*yaml.Node{{Kind: yaml.ScalarNode, Tag: "!!float", Value: "1.5"}}},
		{Kind: KindEnum, Enum: []*yaml.Node{{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"}}},
		{Kind: KindEnum, Enum: []*yaml.Node{{Kind: yaml.ScalarNode, Tag: "!!str", Value: "x"}, {Kind: yaml.ScalarNode, Tag: "!!int", Value: "1"}}},
	} {
		if out := gen.openapiFromIR(enumIR); out == nil {
			t.Fatalf("expected enum schema for %#v", enumIR)
		}
	}
}
