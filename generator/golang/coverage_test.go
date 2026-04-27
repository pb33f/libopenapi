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
	if tagLiteral("x", false, false, false, false) != "" {
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
	if got := gen.goType(&SchemaIR{Kind: KindObject, Name: "Obj", Properties: orderedmap.New[string, *SchemaIR]()}, true, false); got != "map[string]any" {
		t.Fatalf("empty object type: %s", got)
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

	if len(nonNullVariants([]*SchemaIR{nil, {Kind: KindAny, Nullable: true}, {Kind: KindString}})) != 1 {
		t.Fatal("nonNullVariants failed")
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
	if _, err := gen.irFromReflect(reflect.TypeOf(&Provider{}), "Provider", "Provider"); err != nil {
		t.Fatal(err)
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
}
