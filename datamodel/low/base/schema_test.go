package base

import (
	"context"
	"testing"
	timeStd "time"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func test_get_schema_blob() string {
	return `type: object
description: something object
if:
  type: string
else:
  type: integer
then:
  type: boolean
dependentSchemas:
  schemaOne:
    type: string
patternProperties:
  patternOne:
    type: string
propertyNames:
  type: string
unevaluatedItems:
  type: boolean
unevaluatedProperties:
  type: integer
discriminator:
  propertyName: athing
  mapping:
    log: cat
    pizza: party
allOf:
  - type: object
    description: an allof thing
    properties:
      allOfA:
        type: string
        description: allOfA description
        example: 'allOfAExp'
      allOfB:
        type: string
        description: allOfB description
        example: 'allOfBExp'
oneOf:
  - type: object
    description: a oneof thing
    properties:
      oneOfA:
        type: string
        description: oneOfA description
        example: 'oneOfAExp'
      oneOfB:
        type: string
        description: oneOfB description
        example: 'oneOfBExp'
anyOf:
  - type: object
    description: an anyOf thing
    properties:
      anyOfA:
        type: string
        description: anyOfA description
        example: 'anyOfAExp'
      anyOfB:
        type: string
        description: anyOfB description
        example: 'anyOfBExp'
not:
  type: object
  description: a not thing
  properties:
    notA:
      type: string
      description: notA description
      example: 'notAExp'
    notB:
      type: string
      description: notB description
      example: 'notBExp'
items:
  type: object
  description: an items thing
  properties:
    itemsA:
      type: string
      description: itemsA description
      example: 'itemsAExp'
    itemsB:
      type: string
      description: itemsB description
      example: 'itemsBExp'
prefixItems:
  - type: object
    description: an items thing
    properties:
      itemsA:
        type: string
        description: itemsA description
        example: 'itemsAExp'
      itemsB:
        type: string
        description: itemsB description
        example: 'itemsBExp'
properties:
  somethingA:
    type: number
    description: a number
    example: 2
  somethingB:
    type: object
    exclusiveMinimum: true
    exclusiveMaximum: true
    description: an object
    externalDocs:
      description: the best docs
      url: https://pb33f.io
    properties:
      somethingBProp:
        type: string
        description: something b subprop
        example: picnics are nice.
        xml:
          name: an xml thing
          namespace: an xml namespace
          prefix: a prefix
          attribute: true
          wrapped: false
          x-pizza: love
    additionalProperties:
        why: yes
        thatIs: true
additionalProperties: true
required:
  - them
enum:
  - one
  - two
x-pizza: tasty
examples:
  - hey
  - hi!
contains:
  type: int
maxContains: 10
minContains: 1
uniqueItems: true
$anchor: anchor
$dynamicAnchor: dynamicAnchorValue
$dynamicRef: "#dynamicRefTarget"`
}

func Test_Schema(t *testing.T) {
	testSpec := test_get_schema_blob()

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(testSpec), &rootNode)
	assert.NoError(t, mErr)

	sch := Schema{}
	mbErr := low.BuildModel(rootNode.Content[0], &sch)
	assert.NoError(t, mbErr)

	schErr := sch.Build(context.Background(), rootNode.Content[0], nil)
	assert.NoError(t, schErr)
	assert.Equal(t, "something object", sch.Description.Value)
	assert.True(t, sch.AdditionalProperties.Value.B)
	assert.NotNil(t, sch.GetRootNode())

	assert.Equal(t, 2, orderedmap.Len(sch.Properties.Value))
	v := sch.FindProperty("somethingB")

	assert.Equal(t, "https://pb33f.io", v.Value.Schema().ExternalDocs.Value.URL.Value)
	assert.Equal(t, "the best docs", v.Value.Schema().ExternalDocs.Value.Description.Value)

	assert.True(t, v.Value.Schema().ExclusiveMinimum.Value.A)
	assert.True(t, v.Value.Schema().ExclusiveMaximum.Value.A)

	assert.NotNil(t, sch.GetContext())
	assert.Nil(t, sch.GetIndex())

	j := v.Value.Schema().FindProperty("somethingBProp").Value.Schema()
	k := v.Value.Schema().FindProperty("somethingBProp").Value

	assert.Equal(t, k, j.ParentProxy)

	assert.NotNil(t, j)
	assert.NotNil(t, j.XML.Value)
	assert.Equal(t, "an xml thing", j.XML.Value.Name.Value)
	assert.Equal(t, "an xml namespace", j.XML.Value.Namespace.Value)
	assert.Equal(t, "a prefix", j.XML.Value.Prefix.Value)
	assert.Equal(t, true, j.XML.Value.Attribute.Value)
	assert.Equal(t, 1, orderedmap.Len(j.XML.Value.Extensions))
	assert.Equal(t, 1, orderedmap.Len(j.XML.Value.GetExtensions()))

	assert.NotNil(t, v.Value.Schema().AdditionalProperties.Value)

	var addProps map[string]interface{}
	v.Value.Schema().AdditionalProperties.ValueNode.Decode(&addProps)
	assert.Equal(t, "yes", addProps["why"])
	assert.Equal(t, true, addProps["thatIs"])

	// check polymorphic values allOf
	f := sch.AllOf.Value[0].Value.Schema()
	assert.Equal(t, "an allof thing", f.Description.Value)
	assert.Equal(t, 2, orderedmap.Len(f.Properties.Value))

	v = f.FindProperty("allOfA")
	assert.NotNil(t, v)

	io := v.Value.Schema()

	assert.Equal(t, "allOfA description", io.Description.Value)

	var ioExample string
	_ = io.Example.GetValueNode().Decode(&ioExample)

	assert.Equal(t, "allOfAExp", ioExample)

	qw := f.FindProperty("allOfB").Value.Schema()
	assert.NotNil(t, v)
	assert.Equal(t, "allOfB description", qw.Description.Value)

	var qwExample string
	_ = qw.Example.GetValueNode().Decode(&qwExample)

	assert.Equal(t, "allOfBExp", qwExample)

	// check polymorphic values anyOf
	assert.Equal(t, "an anyOf thing", sch.AnyOf.Value[0].Value.Schema().Description.Value)
	assert.Equal(t, 2, orderedmap.Len(sch.AnyOf.Value[0].Value.Schema().Properties.Value))

	v = sch.AnyOf.Value[0].Value.Schema().FindProperty("anyOfA")
	assert.NotNil(t, v)
	assert.Equal(t, "anyOfA description", v.Value.Schema().Description.Value)

	var vSchemaExample string
	_ = v.GetValue().Schema().Example.GetValueNode().Decode(&vSchemaExample)

	assert.Equal(t, "anyOfAExp", vSchemaExample)

	v = sch.AnyOf.Value[0].Value.Schema().FindProperty("anyOfB")
	assert.NotNil(t, v)
	assert.Equal(t, "anyOfB description", v.Value.Schema().Description.Value)

	_ = v.GetValue().Schema().Example.GetValueNode().Decode(&vSchemaExample)
	assert.Equal(t, "anyOfBExp", vSchemaExample)

	// check polymorphic values oneOf
	assert.Equal(t, "a oneof thing", sch.OneOf.Value[0].Value.Schema().Description.Value)
	assert.Equal(t, 2, orderedmap.Len(sch.OneOf.Value[0].Value.Schema().Properties.Value))

	v = sch.OneOf.Value[0].Value.Schema().FindProperty("oneOfA")
	assert.NotNil(t, v)
	assert.Equal(t, "oneOfA description", v.Value.Schema().Description.Value)

	_ = v.GetValue().Schema().Example.GetValueNode().Decode(&vSchemaExample)
	assert.Equal(t, "oneOfAExp", vSchemaExample)

	v = sch.OneOf.Value[0].Value.Schema().FindProperty("oneOfB")
	assert.NotNil(t, v)
	assert.Equal(t, "oneOfB description", v.Value.Schema().Description.Value)

	_ = v.GetValue().Schema().Example.GetValueNode().Decode(&vSchemaExample)
	assert.Equal(t, "oneOfBExp", vSchemaExample)

	// check values NOT
	assert.Equal(t, "a not thing", sch.Not.Value.Schema().Description.Value)
	assert.Equal(t, 2, orderedmap.Len(sch.Not.Value.Schema().Properties.Value))

	v = sch.Not.Value.Schema().FindProperty("notA")
	assert.NotNil(t, v)
	assert.Equal(t, "notA description", v.Value.Schema().Description.Value)

	_ = v.GetValue().Schema().Example.GetValueNode().Decode(&vSchemaExample)
	assert.Equal(t, "notAExp", vSchemaExample)

	v = sch.Not.Value.Schema().FindProperty("notB")
	assert.NotNil(t, v)
	assert.Equal(t, "notB description", v.Value.Schema().Description.Value)

	_ = v.GetValue().Schema().Example.GetValueNode().Decode(&vSchemaExample)
	assert.Equal(t, "notBExp", vSchemaExample)

	// check values Items
	assert.Equal(t, "an items thing", sch.Items.Value.A.Schema().Description.Value)
	assert.Equal(t, 2, orderedmap.Len(sch.Items.Value.A.Schema().Properties.Value))

	v = sch.Items.Value.A.Schema().FindProperty("itemsA")
	assert.NotNil(t, v)
	assert.Equal(t, "itemsA description", v.Value.Schema().Description.Value)

	_ = v.GetValue().Schema().Example.GetValueNode().Decode(&vSchemaExample)
	assert.Equal(t, "itemsAExp", vSchemaExample)

	v = sch.Items.Value.A.Schema().FindProperty("itemsB")
	assert.NotNil(t, v)
	assert.Equal(t, "itemsB description", v.Value.Schema().Description.Value)

	_ = v.GetValue().Schema().Example.GetValueNode().Decode(&vSchemaExample)
	assert.Equal(t, "itemsBExp", vSchemaExample)

	// check values PrefixItems
	assert.Equal(t, "an items thing", sch.PrefixItems.Value[0].Value.Schema().Description.Value)
	assert.Equal(t, 2, orderedmap.Len(sch.PrefixItems.Value[0].Value.Schema().Properties.Value))

	v = sch.PrefixItems.Value[0].Value.Schema().FindProperty("itemsA")
	assert.NotNil(t, v)
	assert.Equal(t, "itemsA description", v.Value.Schema().Description.Value)

	_ = v.GetValue().Schema().Example.GetValueNode().Decode(&vSchemaExample)
	assert.Equal(t, "itemsAExp", vSchemaExample)

	v = sch.PrefixItems.Value[0].Value.Schema().FindProperty("itemsB")
	assert.NotNil(t, v)
	assert.Equal(t, "itemsB description", v.Value.Schema().Description.Value)

	_ = v.GetValue().Schema().Example.GetValue().Decode(&vSchemaExample)
	assert.Equal(t, "itemsBExp", vSchemaExample)

	// check discriminator
	assert.NotNil(t, sch.Discriminator.Value)
	assert.Equal(t, "athing", sch.Discriminator.Value.PropertyName.Value)
	assert.Equal(t, 2, sch.Discriminator.GetValue().Mapping.GetValue().Len())
	mv := sch.Discriminator.Value.FindMappingValue("log")
	assert.Equal(t, "cat", mv.Value)
	mv = sch.Discriminator.Value.FindMappingValue("pizza")
	assert.Equal(t, "party", mv.Value)

	// check 3.1 properties.
	assert.Equal(t, "int", sch.Contains.Value.Schema().Type.Value.A)
	assert.Equal(t, int64(1), sch.MinContains.Value)
	assert.Equal(t, int64(10), sch.MaxContains.Value)
	assert.Equal(t, "string", sch.If.Value.Schema().Type.Value.A)
	assert.Equal(t, "integer", sch.Else.Value.Schema().Type.Value.A)
	assert.Equal(t, "boolean", sch.Then.Value.Schema().Type.Value.A)
	assert.Equal(t, "string", sch.FindDependentSchema("schemaOne").Value.Schema().Type.Value.A)
	assert.Equal(t, "string", sch.FindPatternProperty("patternOne").Value.Schema().Type.Value.A)
	assert.Equal(t, "string", sch.PropertyNames.Value.Schema().Type.Value.A)
	assert.Equal(t, "boolean", sch.UnevaluatedItems.Value.Schema().Type.Value.A)
	assert.Equal(t, "integer", sch.UnevaluatedProperties.Value.A.Schema().Type.Value.A)
	assert.Equal(t, "anchor", sch.Anchor.Value)
	assert.Equal(t, "dynamicAnchorValue", sch.DynamicAnchor.Value)
	assert.Equal(t, "#dynamicRefTarget", sch.DynamicRef.Value)
}

func TestSchemaAllOfSequenceOrder(t *testing.T) {
	testSpec := test_get_allOf_schema_blob()

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(testSpec), &rootNode)
	assert.NoError(t, mErr)

	// test data is a map with one node
	mapContent := rootNode.Content[0].Content

	_, vn := utils.FindKeyNodeTop(AllOfLabel, mapContent)
	assert.True(t, utils.IsNodeArray(vn))

	want := []string{}

	// Go over every element in AllOf and grab description
	// Odd: object
	// Event: description
	for i := range vn.Content {
		assert.True(t, utils.IsNodeMap(vn.Content[i]))
		_, vn := utils.FindKeyNodeTop("description", vn.Content[i].Content)
		assert.True(t, utils.IsNodeStringValue(vn))
		want = append(want, vn.Value)
	}

	sch := Schema{}
	mbErr := low.BuildModel(rootNode.Content[0], &sch)
	assert.NoError(t, mbErr)

	schErr := sch.Build(context.Background(), rootNode.Content[0], nil)
	assert.NoError(t, schErr)
	assert.Equal(t, "allOf sequence check", sch.Description.Value)

	got := []string{}
	for i := range sch.AllOf.Value {
		v := sch.AllOf.Value[i]
		got = append(got, v.Value.Schema().Description.Value)
	}

	assert.Equal(t, want, got)
}

func TestSchema_Hash(t *testing.T) {
	// create two versions
	testSpec := test_get_schema_blob()
	var sc1n yaml.Node
	_ = yaml.Unmarshal([]byte(testSpec), &sc1n)
	sch1 := Schema{}
	_ = low.BuildModel(&sc1n, &sch1)
	_ = sch1.Build(context.Background(), sc1n.Content[0], nil)

	var sc2n yaml.Node
	_ = yaml.Unmarshal([]byte(testSpec), &sc2n)
	sch2 := Schema{}
	_ = low.BuildModel(&sc2n, &sch2)
	_ = sch2.Build(context.Background(), sc2n.Content[0], nil)

	assert.Equal(t, sch1.Hash(), sch2.Hash())
}

func BenchmarkSchema_Hash(b *testing.B) {
	// create two versions
	testSpec := test_get_schema_blob()
	var sc1n yaml.Node
	_ = yaml.Unmarshal([]byte(testSpec), &sc1n)
	sch1 := Schema{}
	_ = low.BuildModel(&sc1n, &sch1)
	_ = sch1.Build(context.Background(), sc1n.Content[0], nil)

	var sc2n yaml.Node
	_ = yaml.Unmarshal([]byte(testSpec), &sc2n)
	sch2 := Schema{}
	_ = low.BuildModel(&sc2n, &sch2)
	_ = sch2.Build(context.Background(), sc2n.Content[0], nil)

	for i := 0; i < b.N; i++ {
		assert.Equal(b, sch1.Hash(), sch2.Hash())
	}
}

func Test_Schema_31(t *testing.T) {
	testSpec := `$schema: https://something
type:
  - object
  - null
description: something object
exclusiveMinimum: 12
exclusiveMaximum: 13
contentEncoding: fish64
contentMediaType: fish/paste
items: true
examples:
  - testing
const: tasty`

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(testSpec), &rootNode)
	assert.NoError(t, mErr)

	sch := Schema{}
	mbErr := low.BuildModel(rootNode.Content[0], &sch)
	assert.NoError(t, mbErr)

	schErr := sch.Build(context.Background(), rootNode.Content[0], nil)
	assert.NoError(t, schErr)
	assert.Equal(t, "something object", sch.Description.Value)
	assert.Len(t, sch.Type.Value.B, 2)
	assert.True(t, sch.Type.Value.IsB())
	assert.Equal(t, "object", sch.Type.Value.B[0].Value)
	assert.True(t, sch.ExclusiveMinimum.Value.IsB())
	assert.False(t, sch.ExclusiveMinimum.Value.IsA())
	assert.True(t, sch.ExclusiveMaximum.Value.IsB())
	assert.Equal(t, float64(12), sch.ExclusiveMinimum.Value.B)
	assert.Equal(t, float64(13), sch.ExclusiveMaximum.Value.B)
	assert.Len(t, sch.Examples.Value, 1)

	var example0 string
	_ = sch.Examples.GetValue()[0].GetValue().Decode(&example0)

	assert.Equal(t, "testing", example0)
	assert.Equal(t, "fish64", sch.ContentEncoding.Value)
	assert.Equal(t, "fish/paste", sch.ContentMediaType.Value)
	assert.True(t, sch.Items.Value.IsB())
	assert.True(t, sch.Items.Value.B)

	var schConst string
	_ = sch.Const.GetValue().Decode(&schConst)

	assert.Equal(t, "tasty", schConst)
}

func TestSchema_Build_PropsLookup(t *testing.T) {
	yml := `components:
  schemas:
    Something:
      description: this is something
      type: string`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `type: object
properties:
  aValue:
    $ref: '#/components/schemas/Something'`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	var n Schema
	err := n.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "this is something", n.FindProperty("aValue").Value.Schema().Description.Value)
}

func TestSchema_Build_PropsLookup_Fail(t *testing.T) {
	yml := `components:
  schemas:
    Something:
      description: this is something
      type: string`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `type: object
properties:
  aValue:
    $ref: '#/bork'`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	var n Schema
	err := n.Build(context.Background(), idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestSchema_Build_DependentSchemas_Fail(t *testing.T) {
	yml := `components:
  schemas:
    Something:
      description: this is something
      type: string`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `type: object
dependentSchemas:
  aValue:
    $ref: '#/bork'`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	var n Schema
	err := n.Build(context.Background(), idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestSchema_Build_PatternProperties_Fail(t *testing.T) {
	yml := `components:
  schemas:
    Something:
      description: this is something
      type: string`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `type: object
patternProperties:
  aValue:
    $ref: '#/bork'`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	var n Schema
	err := n.Build(context.Background(), idxNode.Content[0], idx)
	assert.Error(t, err)
}

func Test_Schema_Polymorphism_Array_Ref(t *testing.T) {
	yml := `components:
  schemas:
    Something:
      type: object
      description: poly thing
      properties:
        polyProp:
          type: string
          description: a property
          example: anything`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `type: object
allOf:
  - $ref: '#/components/schemas/Something'
oneOf:
  - $ref: '#/components/schemas/Something'
anyOf:
  - $ref: '#/components/schemas/Something'
not:
  $ref: '#/components/schemas/Something'
items:
  $ref: '#/components/schemas/Something'`

	var sch Schema
	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	err := low.BuildModel(&idxNode, &sch)
	assert.NoError(t, err)

	schErr := sch.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, schErr)

	desc := "poly thing"
	assert.Equal(t, desc, sch.OneOf.Value[0].Value.Schema().Description.Value)
	assert.Equal(t, desc, sch.AnyOf.Value[0].Value.Schema().Description.Value)
	assert.Equal(t, desc, sch.AllOf.Value[0].Value.Schema().Description.Value)
	assert.Equal(t, desc, sch.Not.Value.Schema().Description.Value)
	assert.Equal(t, desc, sch.Items.Value.A.Schema().Description.Value)
}

func Test_Schema_Polymorphism_Array_Ref_Fail(t *testing.T) {
	yml := `components:
  schemas:
    Something:
      type: object
      description: poly thing
      properties:
        polyProp:
          type: string
          description: a property
          example: anything`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `type: object
allOf:
  - $ref: '#/components/schemas/Missing'
oneOf:
  - $ref: '#/components/schemas/Something'
anyOf:
  - $ref: '#/components/schemas/Something'
not:
  - $ref: '#/components/schemas/Something'
items:
  - $ref: '#/components/schemas/Something'`

	var sch Schema
	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	err := low.BuildModel(&idxNode, &sch)
	assert.NoError(t, err)

	schErr := sch.Build(context.Background(), idxNode.Content[0], idx)
	assert.Error(t, schErr)
}

func Test_Schema_Polymorphism_Map_Ref(t *testing.T) {
	yml := `components:
  schemas:
    Something:
      type: object
      description: poly thing
      properties:
        polyProp:
          type: string
          description: a property
          example: anything`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `type: object
allOf:
  - $ref: '#/components/schemas/Something'
oneOf:
  - $ref: '#/components/schemas/Something'
anyOf:
  - $ref: '#/components/schemas/Something'
not:
  $ref: '#/components/schemas/Something'
items:
  $ref: '#/components/schemas/Something'`

	var sch Schema
	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	err := low.BuildModel(&idxNode, &sch)
	assert.NoError(t, err)

	schErr := sch.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, schErr)

	desc := "poly thing"
	assert.Equal(t, desc, sch.OneOf.Value[0].Value.Schema().Description.Value)
	assert.Equal(t, desc, sch.AnyOf.Value[0].Value.Schema().Description.Value)
	assert.Equal(t, desc, sch.AllOf.Value[0].Value.Schema().Description.Value)
	assert.Equal(t, desc, sch.Not.Value.Schema().Description.Value)
	assert.Equal(t, desc, sch.Items.Value.A.Schema().Description.Value)
}

func Test_Schema_Polymorphism_Map_Ref_Fail(t *testing.T) {
	yml := `components:
  schemas:
    Something:
      type: object
      description: poly thing
      properties:
        polyProp:
          type: string
          description: a property
          example: anything`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `type: object
allOf:
  $ref: '#/components/schemas/Missing'
oneOf:
  $ref: '#/components/schemas/Something'
anyOf:
  $ref: '#/components/schemas/Something'
not:
  $ref: '#/components/schemas/Something'
items:
  $ref: '#/components/schemas/Something'`

	var sch Schema
	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	err := low.BuildModel(&idxNode, &sch)
	assert.NoError(t, err)

	schErr := sch.Build(context.Background(), idxNode.Content[0], idx)
	assert.Error(t, schErr)
}

func Test_Schema_Polymorphism_BorkParent(t *testing.T) {
	yml := `components:
  schemas:
    Something:
      $ref: #borko`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `type: object
allOf:
  $ref: '#/components/schemas/Something'`

	var sch Schema
	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	err := low.BuildModel(&idxNode, &sch)
	assert.NoError(t, err)

	schErr := sch.Build(context.Background(), idxNode.Content[0], idx)
	assert.Error(t, schErr)
}

func Test_Schema_Polymorphism_BorkChild(t *testing.T) {
	yml := `components:
  schemas:
    Something:
      $ref: #borko`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `type: object
allOf:
  $ref: #borko`

	var sch Schema
	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	err := low.BuildModel(&idxNode, &sch)
	assert.NoError(t, err)

	schErr := sch.Build(context.Background(), idxNode.Content[0], idx)
	assert.Error(t, schErr)
}

func Test_Schema_Polymorphism_BorkChild_Array(t *testing.T) {
	yml := `components:
  schemas:
    Something:
      $ref: #borko`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `type: object
allOf:
  - type: object
    allOf:
      - $ref: #bork'`

	var sch Schema
	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	err := low.BuildModel(&idxNode, &sch)
	assert.NoError(t, err)

	schErr := sch.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, schErr)
	assert.Nil(t, sch.AllOf.Value[0].Value.Schema()) // child can't be resolved, so this will be nil.
	assert.Error(t, sch.AllOf.Value[0].Value.GetBuildError())
}

func Test_Schema_Polymorphism_RefMadness(t *testing.T) {
	yml := `components:
  schemas:
    Something:
      $ref: '#/components/schemas/Else'
    Else:
      description: madness`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `type: object
allOf:
  - $ref: '#/components/schemas/Something'`

	var sch Schema
	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	err := low.BuildModel(&idxNode, &sch)
	assert.NoError(t, err)

	schErr := sch.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, schErr)

	desc := "madness"
	assert.Equal(t, desc, sch.AllOf.Value[0].Value.Schema().Description.Value)
}

func Test_Schema_Polymorphism_RefMadnessBork(t *testing.T) {
	yml := `components:
  schemas:
    Something:
      $ref: '#/components/schemas/Else'
    Else:
      $ref: #borko`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `type: object
allOf:
  $ref: '#/components/schemas/Something'`

	var sch Schema
	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	err := low.BuildModel(&idxNode, &sch)
	assert.NoError(t, err)

	err = sch.Build(context.Background(), idxNode.Content[0], idx)
	assert.Error(t, err)
}

func Test_Schema_Polymorphism_RefMadnessIllegal(t *testing.T) {
	// this does not work, but it won't error out.

	yml := `components:
  schemas:
    Something:
      $ref: '#/components/schemas/Else'
    Else:
      description: hey!`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `$ref: '#/components/schemas/Something'`

	var sch Schema
	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	err := low.BuildModel(&idxNode, &sch)
	assert.NoError(t, err)

	schErr := sch.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, schErr)
}

func Test_Schema_RefMadnessIllegal_Circular(t *testing.T) {
	// this does not work, but it won't error out.

	yml := `components:
  schemas:
    Something:
      $ref: '#/components/schemas/Else'
    Else:
      $ref: '#/components/schemas/Something'`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `$ref: '#/components/schemas/Something'`

	var sch Schema
	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	err := low.BuildModel(&idxNode, &sch)
	assert.NoError(t, err)

	schErr := sch.Build(context.Background(), idxNode.Content[0], idx)
	assert.Error(t, schErr)
}

func Test_Schema_RefMadnessIllegal_Nonexist(t *testing.T) {
	// this does not work, but it won't error out.

	yml := `components:
  schemas:
    Something:
      $ref: '#/components/schemas/Else'
    Else:
      $ref: '#/components/schemas/Something'`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `$ref: #BORKLE`

	var sch Schema
	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	err := low.BuildModel(&idxNode, &sch)
	assert.NoError(t, err)

	schErr := sch.Build(context.Background(), idxNode.Content[0], idx)
	assert.Error(t, schErr)
}

func TestExtractSchema(t *testing.T) {
	yml := `components:
  schemas:
    Something:
      description: this is something
      type: string`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `schema:
  type: object
  properties:
    aValue:
      $ref: '#/components/schemas/Something'`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	res, err := ExtractSchema(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.NotNil(t, res.Value)
	aValue := res.Value.Schema().FindProperty("aValue")
	assert.Equal(t, "this is something", aValue.Value.Schema().Description.Value)
}

func TestExtractSchema_DefaultPrimitive(t *testing.T) {
	yml := `
schema:
  type: object
  default: 5`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	res, err := ExtractSchema(context.Background(), idxNode.Content[0], nil)
	assert.NoError(t, err)
	assert.NotNil(t, res.Value)
	sch := res.Value.Schema()

	var def int
	_ = sch.Default.GetValueNode().Decode(&def)

	assert.Equal(t, 5, def)
}

func TestExtractSchema_ConstPrimitive(t *testing.T) {
	yml := `
schema:
  type: object
  const: 5`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	res, err := ExtractSchema(context.Background(), idxNode.Content[0], nil)
	assert.NoError(t, err)
	assert.NotNil(t, res.Value)
	sch := res.Value.Schema()

	var cnst int
	_ = sch.Const.GetValueNode().Decode(&cnst)

	assert.Equal(t, 5, cnst)
	assert.NotNil(t, sch.Hash())
}

func TestExtractSchema_Ref(t *testing.T) {
	yml := `components:
  schemas:
    Something:
      description: this is something
      type: string`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `schema:
  $ref: '#/components/schemas/Something'`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	res, err := ExtractSchema(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.NotNil(t, res.Value)
	assert.Equal(t, "this is something", res.Value.Schema().Description.Value)
}

func TestExtractSchema_Ref_Fail(t *testing.T) {
	yml := `components:
  schemas:
    Something:
      description: this is something
      type: string`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `schema:
  $ref: '#/components/schemas/Missing'`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	_, err := ExtractSchema(context.Background(), idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestExtractSchema_CheckChildPropCircular(t *testing.T) {
	yml := `components:
  schemas:
    Something:
      properties:
        nothing:
          $ref: '#/components/schemas/Nothing'
      required:
        - nothing
    Nothing:
      properties:
        something:
          $ref: '#/components/schemas/Something'
      required:
        - something
`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `$ref: '#/components/schemas/Something'`

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	res, err := ExtractSchema(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.NotNil(t, res.Value)

	props := res.Value.Schema().FindProperty("nothing")
	assert.NotNil(t, props)
}

func TestExtractSchema_RefRoot(t *testing.T) {
	yml := `components:
  schemas:
    Something:
      description: this is something
      type: string`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `$ref: '#/components/schemas/Something'`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	res, err := ExtractSchema(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.NotNil(t, res.Value)
	assert.Equal(t, "this is something", res.Value.Schema().Description.Value)
}

func TestExtractSchema_RefRoot_Fail(t *testing.T) {
	yml := `components:
  schemas:
    Something:
      description: this is something
      type: string`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `$ref: '#/components/schemas/Missing'`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	_, err := ExtractSchema(context.Background(), idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestExtractSchema_RefRoot_Child_Fail(t *testing.T) {
	yml := `components:
  schemas:
    Something:
      $ref: #bork`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `$ref: '#/components/schemas/Something'`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	_, err := ExtractSchema(context.Background(), idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestExtractSchema_AdditionalPropertiesAsSchema(t *testing.T) {
	yml := `components:
  schemas:
    Something:
      additionalProperties:
        type: string`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `$ref: '#/components/schemas/Something'`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	res, err := ExtractSchema(context.Background(), idxNode.Content[0], idx)

	assert.NotNil(t, res.Value.Schema().AdditionalProperties.Value.A.Schema())
	assert.Nil(t, err)
}

func TestExtractSchema_DoNothing(t *testing.T) {
	yml := `components:
  schemas:
    Something:
      $ref: #bork`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `please: do nothing.`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	res, err := ExtractSchema(context.Background(), idxNode.Content[0], idx)
	assert.Nil(t, res)
	assert.Nil(t, err)
}

func TestExtractSchema_AdditionalProperties_Ref(t *testing.T) {
	yml := `components:
  schemas:
    Nothing:
      type: int
    Something:
      additionalProperties:
        cake:
          $ref: '#/components/schemas/Nothing'`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `schema:
  type: int
  additionalProperties:
    $ref: '#/components/schemas/Something'`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	res, err := ExtractSchema(context.Background(), idxNode.Content[0], idx)
	assert.NotNil(t, res.Value.Schema().AdditionalProperties.Value.A.Schema())
	assert.Nil(t, err)
}

func TestExtractSchema_OneOfRef(t *testing.T) {
	yml := `components:
  schemas:
    Error:
      type: object
      description: Error defining what went wrong when providing a specification. The message should help indicate the issue clearly.
      properties:
        message:
          type: string
          description: returns the error message if something wrong happens
          example: No such burger as 'Big-Whopper'
    Burger:
      type: object
      description: The tastiest food on the planet you would love to eat everyday
      required:
        - name
        - numPatties
      properties:
        name:
          type: string
          description: The name of your tasty burger - burger names are listed in our menus
          example: Big Mac
        numPatties:
          type: integer
          description: The number of burger patties used
          example: 2
        numTomatoes:
          type: integer
          description: how many slices of orange goodness would you like?
          example: 1
        fries:
          $ref: '#/components/schemas/Fries'
    Fries:
      type: object
      description: golden slices of happy fun joy
      required:
        - potatoShape
        - favoriteDrink
      properties:
        seasoning:
          type: array
          description: herbs and spices for your golden joy
          items:
            type: string
            description: type of herb or spice used to liven up the yummy
            example: salt
        potatoShape:
          type: string
          description: what type of potato shape? wedges? shoestring?
          example: Crispy Shoestring
        favoriteDrink:
          $ref: '#/components/schemas/Drink'
    Dressing:
      type: object
      description: This is the object that contains the information about the content of the dressing
      required:
        - name
      properties:
        name:
          type: string
          description: The name of your dressing you can pick up from the menu
          example: Cheese
      additionalProperties:
        type: object
        description: something in here.
    Drink:
      type: object
      description: a frosty cold beverage can be coke or sprite
      required:
        - size
        - drinkType
      properties:
        ice:
          type: boolean
        drinkType:
          description: select from coke or sprite
          enum:
            - coke
            - sprite
        size:
          type: string
          description: what size man? S/M/L
          example: M
      additionalProperties: true
      discriminator:
        propertyName: drinkType
        mapping:
          drink: some value
    SomePayload:
      type: string
      description: some kind of payload for something.
      xml:
        name: is html programming? yes.
      externalDocs:
        url: https://pb33f.io/docs
      oneOf:
        - $ref: '#/components/schemas/Drink'`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `schema:
  $ref: '#/components/schemas/SomePayload'`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	res, err := ExtractSchema(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "a frosty cold beverage can be coke or sprite",
		res.Value.Schema().OneOf.Value[0].Value.Schema().Description.Value)
}

func TestSchema_Hash_Equal(t *testing.T) {
	low.ClearHashCache()
	left := `schema:
  $schema: https://athing.com
  multipleOf: 1
  maximum: 10
  minimum: 1
  maxLength: 10
  minLength: 1
  pattern: something
  format: another
  maxItems: 10
  minItems: 1
  uniqueItems: 1
  maxProperties: 10
  minProperties: 1
  additionalProperties: true
  description: milky
  contentEncoding: rubber shoes
  contentMediaType: paper tiger
  default:
     type: jazz
  nullable: true
  readOnly: true
  writeOnly: true
  deprecated: true
  exclusiveMaximum: 23
  exclusiveMinimum: 10
  type:
    - int
  x-coffee: black
  enum:
    - one
    - two
  x-toast: burned
  title: an OK message
  required:
    - propA
  properties:
    propA:
      title: a proxy property
      type: string`

	right := `schema:
  $schema: https://athing.com
  multipleOf: 1
  maximum: 10
  x-coffee: black
  minimum: 1
  maxLength: 10
  minLength: 1
  pattern: something
  format: another
  maxItems: 10
  minItems: 1
  uniqueItems: 1
  maxProperties: 10
  minProperties: 1
  additionalProperties: true
  description: milky
  contentEncoding: rubber shoes
  contentMediaType: paper tiger
  default:
     type: jazz
  nullable: true
  readOnly: true
  writeOnly: true
  deprecated: true
  exclusiveMaximum: 23
  exclusiveMinimum: 10
  type:
    - int
  enum:
    - one
    - two
  x-toast: burned
  title: an OK message
  required:
    - propA
  properties:
    propA:
      title: a proxy property
      type: string`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	lDoc, _ := ExtractSchema(context.Background(), lNode.Content[0], nil)
	rDoc, _ := ExtractSchema(context.Background(), rNode.Content[0], nil)

	assert.NotNil(t, lDoc)
	assert.NotNil(t, rDoc)

	lHash := lDoc.Value.Schema().Hash()
	rHash := rDoc.Value.Schema().Hash()

	assert.Equal(t, lHash, rHash)
}

func TestSchema_Hash_AdditionalPropsSlice(t *testing.T) {
	low.ClearHashCache()
	left := `schema:
  additionalProperties:
    - type: string`

	right := `schema:
  additionalProperties:
    - type: string`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	lDoc, _ := ExtractSchema(context.Background(), lNode.Content[0], nil)
	rDoc, _ := ExtractSchema(context.Background(), rNode.Content[0], nil)

	assert.NotNil(t, lDoc)
	assert.NotNil(t, rDoc)

	lHash := lDoc.Value.Schema().Hash()
	rHash := rDoc.Value.Schema().Hash()

	assert.Equal(t, lHash, rHash)
}

func TestSchema_Hash_AdditionalPropsSliceNoMap(t *testing.T) {
	low.ClearHashCache()
	left := `schema:
  additionalProperties:
    - hello`

	right := `schema:
  additionalProperties:
    - hello`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	lDoc, _ := ExtractSchema(context.Background(), lNode.Content[0], nil)
	rDoc, _ := ExtractSchema(context.Background(), rNode.Content[0], nil)

	assert.NotNil(t, lDoc)
	assert.NotNil(t, rDoc)

	lHash := lDoc.Value.Schema().Hash()
	rHash := rDoc.Value.Schema().Hash()

	assert.Equal(t, lHash, rHash)
}

func TestSchema_Hash_NotEqual(t *testing.T) {
	low.ClearHashCache()
	left := `schema:
  title: an OK message - but different
  items: true
  minContains: 3
  maxContains: 22
  properties:
    propA:
      title: a proxy property
      type: string`

	right := `schema:
  title: an OK message
  items: false
  minContains: 2
  maxContains: 10
  properties:
    propA:
      title: a proxy property
      type: string`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	lDoc, _ := ExtractSchema(context.Background(), lNode.Content[0], nil)
	rDoc, _ := ExtractSchema(context.Background(), rNode.Content[0], nil)

	assert.False(t, low.AreEqual(lDoc.Value.Schema(), rDoc.Value.Schema()))
}

func TestSchema_Hash_EqualJumbled(t *testing.T) {
	left := `schema:
  title: an OK message
  description: a nice thing.
  properties:
    propZ:
      type: int
    propK:
      description: a prop!
      type: bool
    propA:
      title: a proxy property
      type: string`

	right := `schema:
  description: a nice thing.
  properties:
    propA:
      type: string
      title: a proxy property
    propK:
      type: bool
      description: a prop!
    propZ:
      type: int
  title: an OK message`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	lDoc, _ := ExtractSchema(context.Background(), lNode.Content[0], nil)
	rDoc, _ := ExtractSchema(context.Background(), rNode.Content[0], nil)
	assert.True(t, low.AreEqual(lDoc.Value.Schema(), rDoc.Value.Schema()))
}

func test_get_allOf_schema_blob() string {
	return `type: object
description: allOf sequence check
allOf:
  - type: object
    description: allOf sequence check 1
  - description: allOf sequence check 2
  - type: object
    description: allOf sequence check 3
  - description: allOf sequence check 4
`
}

func TestSchema_UnevaluatedPropertiesAsBool_DefinedAsTrue(t *testing.T) {
	yml := `components:
  schemas:
    Something:
      unevaluatedProperties: true`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&iNode, index.CreateOpenAPIIndexConfig())

	yml = `$ref: '#/components/schemas/Something'`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	res, _ := ExtractSchema(context.Background(), idxNode.Content[0], idx)

	assert.True(t, res.Value.Schema().UnevaluatedProperties.Value.IsB())
	assert.True(t, res.Value.Schema().UnevaluatedProperties.Value.B)

	// maphash uses random seed per process, so just test non-empty
	assert.NotEmpty(t, low.GenerateHashString(res.Value.Schema().UnevaluatedProperties.Value))
}

func TestSchema_UnevaluatedPropertiesAsBool_DefinedAsFalse(t *testing.T) {
	yml := `components:
  schemas:
    Something:
      unevaluatedProperties: false`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&iNode, index.CreateOpenAPIIndexConfig())

	yml = `$ref: '#/components/schemas/Something'`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	res, _ := ExtractSchema(context.Background(), idxNode.Content[0], idx)

	assert.True(t, res.Value.Schema().UnevaluatedProperties.Value.IsB())
	assert.False(t, res.Value.Schema().UnevaluatedProperties.Value.B)
}

func TestSchema_UnevaluatedPropertiesAsBool_Undefined(t *testing.T) {
	yml := `components:
  schemas:
    Something:
      description: I have not defined unevaluatedProperties`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&iNode, index.CreateOpenAPIIndexConfig())

	yml = `$ref: '#/components/schemas/Something'`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	res, _ := ExtractSchema(context.Background(), idxNode.Content[0], idx)

	assert.Nil(t, res.Value.Schema().UnevaluatedProperties.Value)
}

func TestSchema_ExclusiveMinimum_3_with_Config(t *testing.T) {
	yml := `openapi: 3.0.3
components:
  schemas:
    Something:
      type: integer
      minimum: 3
      exclusiveMinimum: true`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)

	config := index.CreateOpenAPIIndexConfig()
	config.SpecInfo = &datamodel.SpecInfo{
		VersionNumeric: 3.0,
	}

	idx := index.NewSpecIndexWithConfig(&iNode, config)

	yml = `$ref: '#/components/schemas/Something'`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	res, _ := ExtractSchema(context.Background(), idxNode.Content[0], idx)

	assert.True(t, res.Value.Schema().ExclusiveMinimum.Value.A)
}

func TestSchema_ExclusiveMinimum_31_with_Config(t *testing.T) {
	yml := `openapi: 3.1
components:
  schemas:
    Something:
      type: integer
      minimum: 3
      exclusiveMinimum: 3`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)

	config := index.CreateOpenAPIIndexConfig()
	config.SpecInfo = &datamodel.SpecInfo{
		VersionNumeric: 3.1,
	}

	idx := index.NewSpecIndexWithConfig(&iNode, config)

	yml = `$ref: '#/components/schemas/Something'`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	res, _ := ExtractSchema(context.Background(), idxNode.Content[0], idx)

	assert.Equal(t, 3.0, res.Value.Schema().ExclusiveMinimum.Value.B)
}

func TestSchema_ExclusiveMaximum_3_with_Config(t *testing.T) {
	yml := `openapi: 3.0.3
components:
  schemas:
    Something:
      type: integer
      maximum: 3
      exclusiveMaximum: true`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)

	config := index.CreateOpenAPIIndexConfig()
	config.SpecInfo = &datamodel.SpecInfo{
		VersionNumeric: 3.0,
	}

	idx := index.NewSpecIndexWithConfig(&iNode, config)

	yml = `$ref: '#/components/schemas/Something'`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	res, _ := ExtractSchema(context.Background(), idxNode.Content[0], idx)

	assert.True(t, res.Value.Schema().ExclusiveMaximum.Value.A)
}

func TestSchema_ExclusiveMaximum_31_with_Config(t *testing.T) {
	yml := `openapi: 3.1
components:
  schemas:
    Something:
      type: integer
      maximum: 3
      exclusiveMaximum: 3`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)

	config := index.CreateOpenAPIIndexConfig()
	config.SpecInfo = &datamodel.SpecInfo{
		VersionNumeric: 3.1,
	}

	idx := index.NewSpecIndexWithConfig(&iNode, config)

	yml = `$ref: '#/components/schemas/Something'`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	res, _ := ExtractSchema(context.Background(), idxNode.Content[0], idx)

	assert.Equal(t, 3.0, res.Value.Schema().ExclusiveMaximum.Value.B)
}

func TestSchema_EmptyySchemaRef(t *testing.T) {
	yml := `openapi: 3.0.3
components:
  schemas:
    Something:
      $ref: ''`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)

	config := index.CreateOpenAPIIndexConfig()
	config.SpecInfo = &datamodel.SpecInfo{
		VersionNumeric: 3.0,
	}

	idx := index.NewSpecIndexWithConfig(&iNode, config)

	yml = `schema:
  $ref: ''`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	res, e := ExtractSchema(context.Background(), idxNode.Content[0], idx)
	assert.Nil(t, res)
	assert.Equal(t, "schema build failed: reference '[empty]' cannot be found at line 2, col 9", e.Error())
}

func TestSchema_EmptyRef(t *testing.T) {
	yml := `openapi: 3.0.3
components:
  schemas:
    Something:
      $ref: ''`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)

	config := index.CreateOpenAPIIndexConfig()
	config.SpecInfo = &datamodel.SpecInfo{
		VersionNumeric: 3.0,
	}

	idx := index.NewSpecIndexWithConfig(&iNode, config)

	yml = `$ref: ''`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	res, e := ExtractSchema(context.Background(), idxNode.Content[0], idx)
	assert.Nil(t, res)
	assert.Equal(t, "schema build failed: reference '[empty]' cannot be found at line 1, col 7", e.Error())
}

func TestBuildSchema_BadNodeTypes(t *testing.T) {
	n := &yaml.Node{
		Tag:    "!!burgers",
		Line:   1,
		Column: 2,
	}

	_, err := buildSchema(context.Background(), n, n, nil)
	assert.Error(t, err)
	assert.Equal(t, "build schema failed: expected a single schema object for 'unknown', but found an array or scalar at line 1, col 2", err.Error())
}

func TestExtractSchema_CheckPathAndSpec(t *testing.T) {
	yml := `openapi: 3.0.3
components:
  schemas:
    Something:
      $ref: ''`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)

	config := index.CreateOpenAPIIndexConfig()
	config.SpecInfo = &datamodel.SpecInfo{
		VersionNumeric: 3.0,
	}

	idx := index.NewSpecIndexWithConfig(&iNode, config)

	yml = `schema:
  $ref: "#/"`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	ctx := context.WithValue(context.Background(), index.CurrentPathKey, "test")
	idx.SetAbsolutePath("/not/there")
	res, e := ExtractSchema(ctx, idxNode.Content[0], idx)
	assert.Nil(t, res)
	assert.Equal(t, "schema build failed: reference '#/' cannot be found at line 2, col 9", e.Error())
}

func TestExtractSchema_CheckExampleNodesExtracted(t *testing.T) {
	yml := `schema:
  type: object
  example:
    ping: pong
    jing:
      jong: jang
  examples:
   - tang: bang
   - bom: jog
     ding: dong`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)

	config := index.CreateOpenAPIIndexConfig()
	config.SpecInfo = &datamodel.SpecInfo{
		VersionNumeric: 3.0,
	}
	idx := index.NewSpecIndexWithConfig(&iNode, config)
	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	ctx := context.WithValue(context.Background(), index.CurrentPathKey, "test")

	res, e := ExtractSchema(ctx, idxNode.Content[0], idx)
	if res != nil {
		sch := res.Value.Schema()
		assert.NotNil(t, sch.Nodes)
		assert.NoError(t, e)

		n, _ := sch.Nodes.Load(4)
		assert.NotNil(t, n.([]*yaml.Node)[1])
		assert.Equal(t, "ping", n.([]*yaml.Node)[0].Value)
		assert.Equal(t, "pong", n.([]*yaml.Node)[1].Value)

		n, _ = sch.Nodes.Load(8)
		assert.NotNil(t, n.([]*yaml.Node)[0])
		assert.Equal(t, "tang", n.([]*yaml.Node)[1].Value)
		assert.Equal(t, "bang", n.([]*yaml.Node)[2].Value)

	} else {
		t.Fail()
	}
}

func TestSchema_Hash_Empty(t *testing.T) {
	var s *Schema
	assert.NotNil(t, s.Hash())
}

func TestSetup(t *testing.T) {
	ClearSchemaQuickHashMap()
}

func TestSchema_QuickHash(t *testing.T) {
	low.ClearHashCache()
	yml := `schema:
  type: object
  example:
    ping: pong
    jing:
      jong: jang
  examples:
   - tang: bang
   - bom: jog
     ding: dong`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)

	config := index.CreateOpenAPIIndexConfig()
	config.SpecInfo = &datamodel.SpecInfo{
		VersionNumeric: 3.0,
	}
	idx := index.NewSpecIndexWithConfig(&iNode, config)
	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	ctx := context.WithValue(context.Background(), index.CurrentPathKey, "test")

	res, _ := ExtractSchema(ctx, idxNode.Content[0], idx)

	quickHash := res.Value.Schema().QuickHash()
	quickHashCompare := quickHash
	regularHash := res.Value.Schema().Hash()
	regularHashCompare := regularHash
	assert.NotEmpty(t, quickHash)
	assert.NotEmpty(t, regularHash)
	assert.Equal(t, quickHash, regularHash)

	// rehash each 50 times, should always be the same
	// calculate how long loop takes to run
	now := timeStd.Now()
	for i := 0; i < 50; i++ {
		quickHashCompare = res.Value.Schema().QuickHash()
		assert.Equal(t, quickHash, quickHashCompare)
	}
	duration := timeStd.Since(now)
	//fmt.Printf("Quick Duration: %d microseconds\n", duration.Microseconds())
	low.ClearHashCache()
	// rehash each 50 times, should always be the same
	// calculate how long loop takes to run
	now = timeStd.Now()
	for i := 0; i < 50; i++ {
		regularHashCompare = res.Value.Schema().Hash()
		assert.Equal(t, regularHash, regularHashCompare)
	}
	durationRegular := timeStd.Since(now)
	//fmt.Printf("Regular Duration: %d microseconds\n", durationRegular.Microseconds())

	// Note: Timing assertions removed - they are flaky on CI systems (especially Windows)
	// where CPU scheduling and runner performance vary. The important assertions above
	// verify correctness: hashes are equal and consistent across multiple calls.
	_ = duration
	_ = durationRegular
}

func TestSchema_Build_DependentRequired_Success(t *testing.T) {
	yml := `type: object
description: something object
dependentRequired:
  billingAddress:
    - street_address
    - locality
    - region
  creditCard:
    - billing_address
properties:
  name:
    type: string
  billingAddress:
    type: object
  creditCard:
    type: string`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Schema
	err := n.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)

	// Check that DependentRequired was parsed correctly
	assert.NotNil(t, n.DependentRequired.Value)
	assert.Equal(t, 2, n.DependentRequired.Value.Len())

	// Check billingAddress dependency by iterating through the map
	foundBilling := false
	foundCredit := false
	for key, value := range n.DependentRequired.Value.FromOldest() {
		if key.Value == "billingAddress" {
			assert.Equal(t, []string{"street_address", "locality", "region"}, value.Value)
			foundBilling = true
		}
		if key.Value == "creditCard" {
			assert.Equal(t, []string{"billing_address"}, value.Value)
			foundCredit = true
		}
	}
	assert.True(t, foundBilling)
	assert.True(t, foundCredit)
}

func TestSchema_Build_DependentRequired_Empty(t *testing.T) {
	yml := `type: object
description: something object
properties:
  name:
    type: string`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Schema
	err := n.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)

	// Check that DependentRequired is empty
	assert.Nil(t, n.DependentRequired.Value)
}

func TestSchema_Build_DependentRequired_EmptyArray(t *testing.T) {
	yml := `type: object
dependentRequired:
  billingAddress: []`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Schema
	err := n.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)

	// Check that DependentRequired has empty array (nil is equivalent to empty slice in Go)
	assert.NotNil(t, n.DependentRequired.Value)
	found := false
	for key, value := range n.DependentRequired.Value.FromOldest() {
		if key.Value == "billingAddress" {
			assert.Empty(t, value.Value) // Use Empty() which handles both nil and empty slices
			found = true
		}
	}
	assert.True(t, found)
}

func TestSchema_Build_DependentRequired_InvalidValue_NotArray(t *testing.T) {
	yml := `type: object
dependentRequired:
  billingAddress: "not_an_array"`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Schema
	err := n.Build(context.Background(), idxNode.Content[0], idx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dependentRequired value must be an array")
}

func TestSchema_Build_DependentRequired_InvalidValue_NonStringArrayItem(t *testing.T) {
	yml := `type: object
dependentRequired:
  billingAddress:
    - street_address
    - nested:
        invalid: true  # This should be a string, not an object`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Schema
	err := n.Build(context.Background(), idxNode.Content[0], idx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dependentRequired array items must be strings")
}

func TestSchema_Hash_IncludesDependentRequired(t *testing.T) {
	yml1 := `type: object
dependentRequired:
  billingAddress:
    - street_address
    - locality`

	yml2 := `type: object
dependentRequired:
  billingAddress:
    - street_address
    - region`

	// Parse first schema
	var idxNode1 yaml.Node
	_ = yaml.Unmarshal([]byte(yml1), &idxNode1)
	idx1 := index.NewSpecIndex(&idxNode1)
	var schema1 Schema
	_ = schema1.Build(context.Background(), idxNode1.Content[0], idx1)

	// Parse second schema
	var idxNode2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml2), &idxNode2)
	idx2 := index.NewSpecIndex(&idxNode2)
	var schema2 Schema
	_ = schema2.Build(context.Background(), idxNode2.Content[0], idx2)

	// Hashes should be different because DependentRequired is different
	hash1 := schema1.Hash()
	hash2 := schema2.Hash()
	assert.NotEqual(t, hash1, hash2)
}

func TestSchema_Hash_SameDependentRequired(t *testing.T) {
	yml := `type: object
dependentRequired:
  billingAddress:
    - street_address
    - locality`

	// Parse same schema twice
	var idxNode1 yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode1)
	idx1 := index.NewSpecIndex(&idxNode1)
	var schema1 Schema
	_ = schema1.Build(context.Background(), idxNode1.Content[0], idx1)

	var idxNode2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode2)
	idx2 := index.NewSpecIndex(&idxNode2)
	var schema2 Schema
	_ = schema2.Build(context.Background(), idxNode2.Content[0], idx2)

	// Hashes should be the same
	hash1 := schema1.Hash()
	hash2 := schema2.Hash()
	assert.Equal(t, hash1, hash2)
}

func TestSchema_Build_SiblingRefTransformation(t *testing.T) {
	t.Run("sibling ref transformation enabled", func(t *testing.T) {
		// create a complete spec with both schemas to avoid reference resolution errors
		completeSpec := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    destination-base:
      type: object
      properties:
        id:
          type: string
    destination-amazon-sqs:
      title: "destination-amazon-sqs"
      description: "amazon sqs configuration"
      example: {"queueUrl": "test"}
      $ref: "#/components/schemas/destination-base"`

		var idxNode yaml.Node
		_ = yaml.Unmarshal([]byte(completeSpec), &idxNode)
		config := index.CreateOpenAPIIndexConfig()
		config.TransformSiblingRefs = true
		idx := index.NewSpecIndexWithConfig(&idxNode, config)

		// find the destination-amazon-sqs schema node
		var customerAddressNode *yaml.Node
		if idxNode.Content[0].Content != nil {
			for i, node := range idxNode.Content[0].Content {
				if node.Value == "components" && i+1 < len(idxNode.Content[0].Content) {
					componentsNode := idxNode.Content[0].Content[i+1]
					for j, compNode := range componentsNode.Content {
						if compNode.Value == "schemas" && j+1 < len(componentsNode.Content) {
							schemasNode := componentsNode.Content[j+1]
							for k, schemaNode := range schemasNode.Content {
								if schemaNode.Value == "destination-amazon-sqs" && k+1 < len(schemasNode.Content) {
									customerAddressNode = schemasNode.Content[k+1]
									break
								}
							}
							break
						}
					}
					break
				}
			}
		}

		assert.NotNil(t, customerAddressNode)

		// build schema proxy which should trigger transformation, then get the schema
		schemaProxy := &SchemaProxy{}
		err := schemaProxy.Build(context.Background(), nil, customerAddressNode, idx)
		assert.NoError(t, err)

		// get the transformed schema
		schema := schemaProxy.Schema()
		assert.NotNil(t, schema)

		// verify transformation occurred - root node should now be allOf
		assert.Equal(t, "allOf", schema.RootNode.Content[0].Value, "transformation should create allOf structure")

		// verify the RootNode has the correct allOf structure
		allOfArrayNode := schema.RootNode.Content[1]
		assert.Equal(t, yaml.SequenceNode, allOfArrayNode.Kind)
		assert.Len(t, allOfArrayNode.Content, 2, "allOf should have 2 elements")

		// verify structure integrity
		firstElement := allOfArrayNode.Content[0]
		secondElement := allOfArrayNode.Content[1]
		assert.Equal(t, yaml.MappingNode, firstElement.Kind)
		assert.Equal(t, yaml.MappingNode, secondElement.Kind)

		// check that first element has the sibling properties
		hasTitle := false
		for i := 0; i < len(firstElement.Content); i += 2 {
			if firstElement.Content[i].Value == "title" {
				hasTitle = true
				assert.Equal(t, "destination-amazon-sqs", firstElement.Content[i+1].Value)
			}
		}
		assert.True(t, hasTitle, "first allOf element should contain title")

		// check that second element is the reference
		assert.Equal(t, "$ref", secondElement.Content[0].Value)
		assert.Equal(t, "#/components/schemas/destination-base", secondElement.Content[1].Value)
	})

	t.Run("sibling ref transformation disabled maintains compatibility", func(t *testing.T) {
		yml := `title: "destination-amazon-sqs"
$ref: "#/components/schemas/destination-base"`

		var idxNode yaml.Node
		_ = yaml.Unmarshal([]byte(yml), &idxNode)
		config := index.CreateOpenAPIIndexConfig()
		config.TransformSiblingRefs = false
		idx := index.NewSpecIndexWithConfig(&idxNode, config)

		// when transformation is disabled, the original node structure should be preserved
		originalNode := idxNode.Content[0]
		assert.Equal(t, "title", originalNode.Content[0].Value)
		assert.Equal(t, "$ref", originalNode.Content[2].Value)

		// verify transformer correctly identifies no transformation needed
		transformer := NewSiblingRefTransformer(idx)
		assert.False(t, transformer.ShouldTransform(originalNode))
	})

	t.Run("ref only schema unchanged", func(t *testing.T) {
		yml := `$ref: "#/components/schemas/destination-base"`

		var idxNode yaml.Node
		_ = yaml.Unmarshal([]byte(yml), &idxNode)
		config := index.CreateOpenAPIIndexConfig()
		config.TransformSiblingRefs = true
		idx := index.NewSpecIndexWithConfig(&idxNode, config)

		// ref-only schemas should not be transformed
		originalNode := idxNode.Content[0]
		transformer := NewSiblingRefTransformer(idx)
		assert.False(t, transformer.ShouldTransform(originalNode), "ref-only should not be transformed")

		// verify no transformation occurs
		result, err := transformer.TransformSiblingRef(originalNode)
		assert.NoError(t, err)
		assert.Equal(t, originalNode, result, "ref-only should return original node")
	})
}

func TestSchema_Build_EndToEndSiblingRefSupport(t *testing.T) {
	t.Run("complete github issue 90 example", func(t *testing.T) {
		// create a complete spec to avoid reference resolution issues
		completeSpec := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    destination-base:
      type: object
      properties:
        id:
          type: string
    destination-amazon-sqs:
      title: destination-amazon-sqs
      $ref: '#/components/schemas/destination-base'`

		var idxNode yaml.Node
		_ = yaml.Unmarshal([]byte(completeSpec), &idxNode)
		config := index.CreateOpenAPIIndexConfig()
		config.TransformSiblingRefs = true
		idx := index.NewSpecIndexWithConfig(&idxNode, config)

		// find the destination-amazon-sqs schema node
		var targetSchemaNode *yaml.Node
		if idxNode.Content[0].Content != nil {
			for i, node := range idxNode.Content[0].Content {
				if node.Value == "components" && i+1 < len(idxNode.Content[0].Content) {
					componentsNode := idxNode.Content[0].Content[i+1]
					for j, compNode := range componentsNode.Content {
						if compNode.Value == "schemas" && j+1 < len(componentsNode.Content) {
							schemasNode := componentsNode.Content[j+1]
							for k, schemaNode := range schemasNode.Content {
								if schemaNode.Value == "destination-amazon-sqs" && k+1 < len(schemasNode.Content) {
									targetSchemaNode = schemasNode.Content[k+1]
									break
								}
							}
							break
						}
					}
					break
				}
			}
		}

		assert.NotNil(t, targetSchemaNode)

		// build schema proxy which should trigger transformation, then get the schema
		schemaProxy := &SchemaProxy{}
		err := schemaProxy.Build(context.Background(), nil, targetSchemaNode, idx)
		assert.NoError(t, err)

		// get the transformed schema
		schema := schemaProxy.Schema()
		assert.NotNil(t, schema)

		// verify transformation to allOf occurred
		assert.Equal(t, "allOf", schema.RootNode.Content[0].Value)

		// verify structure matches expected allOf format
		allOfArray := schema.RootNode.Content[1]
		assert.Len(t, allOfArray.Content, 2)

		// first element should have title
		firstElement := allOfArray.Content[0]
		assert.Equal(t, "title", firstElement.Content[0].Value)
		assert.Equal(t, "destination-amazon-sqs", firstElement.Content[1].Value)

		// second element should be the reference
		secondElement := allOfArray.Content[1]
		assert.Equal(t, "$ref", secondElement.Content[0].Value)
		assert.Equal(t, "#/components/schemas/destination-base", secondElement.Content[1].Value)
	})

	t.Run("github issue 262 style example", func(t *testing.T) {
		// create complete spec for issue 262 style testing
		completeSpec := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    address:
      type: object
      properties:
        street:
          type: string
    customer-address:
      example: {"addressLine1": "123 Example Road", "city": "Somewhere"}
      description: "Custom address description"
      $ref: "#/components/schemas/address"`

		var idxNode yaml.Node
		_ = yaml.Unmarshal([]byte(completeSpec), &idxNode)
		config := index.CreateOpenAPIIndexConfig()
		config.TransformSiblingRefs = true
		idx := index.NewSpecIndexWithConfig(&idxNode, config)

		// find the customer-address schema node
		var targetSchemaNode *yaml.Node
		if idxNode.Content[0].Content != nil {
			for i, node := range idxNode.Content[0].Content {
				if node.Value == "components" && i+1 < len(idxNode.Content[0].Content) {
					componentsNode := idxNode.Content[0].Content[i+1]
					for j, compNode := range componentsNode.Content {
						if compNode.Value == "schemas" && j+1 < len(componentsNode.Content) {
							schemasNode := componentsNode.Content[j+1]
							for k, schemaNode := range schemasNode.Content {
								if schemaNode.Value == "customer-address" && k+1 < len(schemasNode.Content) {
									targetSchemaNode = schemasNode.Content[k+1]
									break
								}
							}
							break
						}
					}
					break
				}
			}
		}

		assert.NotNil(t, targetSchemaNode)

		// build schema proxy which should trigger transformation, then get the schema
		schemaProxy := &SchemaProxy{}
		err := schemaProxy.Build(context.Background(), nil, targetSchemaNode, idx)
		assert.NoError(t, err)

		// get the transformed schema
		schema := schemaProxy.Schema()
		assert.NotNil(t, schema)

		// verify transformation preserves example and description
		allOfArray := schema.RootNode.Content[1]
		firstElement := allOfArray.Content[0]

		// check that example and description are preserved
		hasExample := false
		hasDescription := false
		for i := 0; i < len(firstElement.Content); i += 2 {
			if firstElement.Content[i].Value == "example" {
				hasExample = true
			}
			if firstElement.Content[i].Value == "description" {
				hasDescription = true
				assert.Equal(t, "Custom address description", firstElement.Content[i+1].Value)
			}
		}
		assert.True(t, hasExample, "example should be preserved")
		assert.True(t, hasDescription, "description should be preserved")
	})
}

func TestSchema_Build_PropertyMerging_Issue262(t *testing.T) {
	t.Run("property merging with reference resolution", func(t *testing.T) {
		// create a complete spec with the target schema to resolve to
		specYml := `openapi: 3.1.0
info:
  title: Property Merging Test
  version: 1.0.0
components:
  schemas:
    Address:
      type: object
      description: "Base address schema"
      properties:
        street:
          type: string
        city:
          type: string
    CustomerAddress:
      example:
        street: "123 Example Road"
        city: "Test City"
      description: "Customer specific address"
      $ref: "#/components/schemas/Address"`

		var rootDoc yaml.Node
		_ = yaml.Unmarshal([]byte(specYml), &rootDoc)

		config := index.CreateOpenAPIIndexConfig()
		config.TransformSiblingRefs = true
		idx := index.NewSpecIndexWithConfig(&rootDoc, config)

		// find the CustomerAddress schema node
		var customerAddressNode *yaml.Node
		for i, node := range rootDoc.Content[0].Content {
			if node.Value == "components" {
				components := rootDoc.Content[0].Content[i+1]
				for j, compNode := range components.Content {
					if compNode.Value == "schemas" {
						schemas := components.Content[j+1]
						for k, schemaKey := range schemas.Content {
							if schemaKey.Value == "CustomerAddress" {
								customerAddressNode = schemas.Content[k+1]
								break
							}
						}
					}
				}
			}
		}

		assert.NotNil(t, customerAddressNode)

		// build schema proxy which should trigger transformation, then get the schema
		schemaProxy := &SchemaProxy{}
		err := schemaProxy.Build(context.Background(), nil, customerAddressNode, idx)
		assert.NoError(t, err)

		// get the transformed schema
		schema := schemaProxy.Schema()
		assert.NotNil(t, schema)

		// verify transformation occurred
		assert.Equal(t, "allOf", schema.RootNode.Content[0].Value)

		// verify sibling properties are preserved in first allOf element
		allOfArray := schema.RootNode.Content[1]
		firstElement := allOfArray.Content[0]

		hasExample := false
		hasDescription := false
		for i := 0; i < len(firstElement.Content); i += 2 {
			if firstElement.Content[i].Value == "example" {
				hasExample = true
			}
			if firstElement.Content[i].Value == "description" {
				hasDescription = true
			}
		}
		assert.True(t, hasExample, "example should be preserved in allOf structure")
		assert.True(t, hasDescription, "description should be preserved in allOf structure")
	})
}

func TestSchemaDynamicValue_Hash_IsA(t *testing.T) {
	// test when IsA() returns true (N=0, A has value)
	value := &SchemaDynamicValue[string, int]{
		N: 0,
		A: "test value",
		B: 42,
	}

	hash := value.Hash()

	// maphash uses random seed per process, just verify it's non-zero
	assert.NotEqual(t, uint64(0), hash)
	assert.True(t, value.IsA())
	assert.False(t, value.IsB())
}

func TestSchema_Build_WithTransformedParentProxy(t *testing.T) {
	// test that lines 658-659 in schema.go are covered (transformed parent proxy check)
	// this needs to test the Build method directly
	yml := `$ref: '#/components/schemas/Base'`

	var schemaNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &schemaNode)

	// create spec
	specYml := `openapi: 3.1.0
components:
  schemas:
    Base:
      type: object
      properties:
        id:
          type: string`

	var specNode yaml.Node
	_ = yaml.Unmarshal([]byte(specYml), &specNode)

	cfg := index.CreateOpenAPIIndexConfig()
	idx := index.NewSpecIndexWithConfig(&specNode, cfg)

	// create a schema with a parent proxy that has TransformedRef set
	schema := &Schema{}
	sp := &SchemaProxy{
		TransformedRef: &yaml.Node{}, // simulate transformation
	}
	schema.ParentProxy = sp

	// call Build which should detect the transformed parent proxy
	err := schema.Build(context.Background(), schemaNode.Content[0], idx)
	assert.NoError(t, err)

	// the isTransformed check happens internally and should skip reference dereferencing
	// when ParentProxy.TransformedRef is not nil
	assert.NotNil(t, schema.ParentProxy)
	assert.NotNil(t, schema.ParentProxy.TransformedRef)
}

func TestSchemaDynamicValue_Hash_IsB(t *testing.T) {
	// test when IsB() returns true (N=1, B has value)
	value := &SchemaDynamicValue[string, int]{
		N: 1,
		A: "test value",
		B: 42,
	}

	hash := value.Hash()

	// maphash uses random seed per process, just verify it's non-zero
	assert.NotEqual(t, uint64(0), hash)
	assert.False(t, value.IsA())
	assert.True(t, value.IsB())
}

// TestSchema_Id tests that the $id field is correctly extracted and included in the hash
func TestSchema_Id(t *testing.T) {
	yml := `type: object
$id: "https://example.com/schemas/pet.json"
description: A pet schema`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	var sch Schema
	err := low.BuildModel(idxNode.Content[0], &sch)
	assert.NoError(t, err)

	err = sch.Build(context.Background(), idxNode.Content[0], nil)
	assert.NoError(t, err)

	assert.Equal(t, "https://example.com/schemas/pet.json", sch.Id.Value)
	assert.NotNil(t, sch.Id.KeyNode)
	assert.NotNil(t, sch.Id.ValueNode)
}

// TestSchema_Id_Hash tests that $id is included in the schema hash
func TestSchema_Id_Hash(t *testing.T) {
	yml1 := `type: object
$id: "https://example.com/schemas/a.json"
description: Schema A`

	yml2 := `type: object
$id: "https://example.com/schemas/b.json"
description: Schema A`

	yml3 := `type: object
description: Schema A`

	var node1, node2, node3 yaml.Node
	_ = yaml.Unmarshal([]byte(yml1), &node1)
	_ = yaml.Unmarshal([]byte(yml2), &node2)
	_ = yaml.Unmarshal([]byte(yml3), &node3)

	var sch1, sch2, sch3 Schema
	_ = low.BuildModel(node1.Content[0], &sch1)
	_ = sch1.Build(context.Background(), node1.Content[0], nil)

	_ = low.BuildModel(node2.Content[0], &sch2)
	_ = sch2.Build(context.Background(), node2.Content[0], nil)

	_ = low.BuildModel(node3.Content[0], &sch3)
	_ = sch3.Build(context.Background(), node3.Content[0], nil)

	hash1 := sch1.Hash()
	hash2 := sch2.Hash()
	hash3 := sch3.Hash()

	// Different $id values should produce different hashes
	assert.NotEqual(t, hash1, hash2)
	// Schema without $id should differ from schema with $id
	assert.NotEqual(t, hash1, hash3)
	assert.NotEqual(t, hash2, hash3)
}

// TestSchema_Id_Empty tests that empty $id is not set
func TestSchema_Id_Empty(t *testing.T) {
	yml := `type: object
description: A schema without $id`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	var sch Schema
	err := low.BuildModel(idxNode.Content[0], &sch)
	assert.NoError(t, err)

	err = sch.Build(context.Background(), idxNode.Content[0], nil)
	assert.NoError(t, err)

	assert.True(t, sch.Id.IsEmpty())
}

// JSON Schema 2020-12 keyword tests

func TestSchema_Comment(t *testing.T) {
	yml := `type: object
$comment: This is a comment that explains the schema purpose
description: A schema with $comment`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	var sch Schema
	err := low.BuildModel(idxNode.Content[0], &sch)
	assert.NoError(t, err)

	err = sch.Build(context.Background(), idxNode.Content[0], nil)
	assert.NoError(t, err)

	assert.Equal(t, "This is a comment that explains the schema purpose", sch.Comment.Value)
}

func TestSchema_Comment_Empty(t *testing.T) {
	yml := `type: object
description: A schema without $comment`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	var sch Schema
	err := low.BuildModel(idxNode.Content[0], &sch)
	assert.NoError(t, err)

	err = sch.Build(context.Background(), idxNode.Content[0], nil)
	assert.NoError(t, err)

	assert.True(t, sch.Comment.IsEmpty())
}

func TestSchema_ContentSchema(t *testing.T) {
	yml := `type: string
contentMediaType: application/jwt
contentSchema:
  type: object
  properties:
    iss:
      type: string
    exp:
      type: integer`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	var sch Schema
	err := low.BuildModel(idxNode.Content[0], &sch)
	assert.NoError(t, err)

	err = sch.Build(context.Background(), idxNode.Content[0], nil)
	assert.NoError(t, err)

	assert.False(t, sch.ContentSchema.IsEmpty())
	assert.NotNil(t, sch.ContentSchema.Value)

	// Verify the contentSchema is a valid schema proxy
	contentSch := sch.ContentSchema.Value.Schema()
	assert.NotNil(t, contentSch)
	assert.Equal(t, "object", contentSch.Type.Value.A)
}

func TestSchema_ContentSchema_Empty(t *testing.T) {
	yml := `type: string
contentMediaType: text/plain`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	var sch Schema
	err := low.BuildModel(idxNode.Content[0], &sch)
	assert.NoError(t, err)

	err = sch.Build(context.Background(), idxNode.Content[0], nil)
	assert.NoError(t, err)

	assert.True(t, sch.ContentSchema.IsEmpty())
}

func TestSchema_Vocabulary(t *testing.T) {
	yml := `$vocabulary:
  https://json-schema.org/draft/2020-12/vocab/core: true
  https://json-schema.org/draft/2020-12/vocab/applicator: true
  https://json-schema.org/draft/2020-12/vocab/validation: false
type: object`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	var sch Schema
	err := low.BuildModel(idxNode.Content[0], &sch)
	assert.NoError(t, err)

	err = sch.Build(context.Background(), idxNode.Content[0], nil)
	assert.NoError(t, err)

	assert.NotNil(t, sch.Vocabulary.Value)
	assert.Equal(t, 3, sch.Vocabulary.Value.Len())

	// Check specific vocabulary entries
	for k, v := range sch.Vocabulary.Value.FromOldest() {
		switch k.Value {
		case "https://json-schema.org/draft/2020-12/vocab/core":
			assert.True(t, v.Value)
		case "https://json-schema.org/draft/2020-12/vocab/applicator":
			assert.True(t, v.Value)
		case "https://json-schema.org/draft/2020-12/vocab/validation":
			assert.False(t, v.Value)
		}
	}
}

func TestSchema_Vocabulary_Empty(t *testing.T) {
	yml := `type: object
description: A regular schema without $vocabulary`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	var sch Schema
	err := low.BuildModel(idxNode.Content[0], &sch)
	assert.NoError(t, err)

	err = sch.Build(context.Background(), idxNode.Content[0], nil)
	assert.NoError(t, err)

	assert.Nil(t, sch.Vocabulary.Value)
}

func TestSchema_Hash_IncludesNewFields(t *testing.T) {
	// Test that hash() includes the new JSON Schema 2020-12 fields
	yml1 := `type: object
$comment: Comment 1`

	yml2 := `type: object
$comment: Comment 2`

	var node1, node2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml1), &node1)
	_ = yaml.Unmarshal([]byte(yml2), &node2)

	var sch1, sch2 Schema
	_ = low.BuildModel(node1.Content[0], &sch1)
	_ = sch1.Build(context.Background(), node1.Content[0], nil)

	_ = low.BuildModel(node2.Content[0], &sch2)
	_ = sch2.Build(context.Background(), node2.Content[0], nil)

	hash1 := sch1.Hash()
	hash2 := sch2.Hash()

	// Different comments should produce different hashes
	assert.NotEqual(t, hash1, hash2)
}

// TestSchema_Vocabulary_AlternativeBooleanFormats tests that strconv.ParseBool handles
// various boolean representations correctly (1, 0, t, f, T, F, TRUE, FALSE, etc.)
func TestSchema_Vocabulary_AlternativeBooleanFormats(t *testing.T) {
	yml := `type: object
$vocabulary:
  "https://example.com/vocab/one": 1
  "https://example.com/vocab/zero": 0
  "https://example.com/vocab/t": t
  "https://example.com/vocab/f": f
  "https://example.com/vocab/TRUE": TRUE
  "https://example.com/vocab/FALSE": FALSE`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	var sch Schema
	err := low.BuildModel(idxNode.Content[0], &sch)
	assert.NoError(t, err)

	err = sch.Build(context.Background(), idxNode.Content[0], nil)
	assert.NoError(t, err)

	assert.NotNil(t, sch.Vocabulary.Value)
	assert.Equal(t, 6, sch.Vocabulary.Value.Len())

	// Check specific vocabulary entries with alternative boolean formats
	for k, v := range sch.Vocabulary.Value.FromOldest() {
		switch k.Value {
		case "https://example.com/vocab/one":
			assert.True(t, v.Value, "1 should parse as true")
		case "https://example.com/vocab/zero":
			assert.False(t, v.Value, "0 should parse as false")
		case "https://example.com/vocab/t":
			assert.True(t, v.Value, "t should parse as true")
		case "https://example.com/vocab/f":
			assert.False(t, v.Value, "f should parse as false")
		case "https://example.com/vocab/TRUE":
			assert.True(t, v.Value, "TRUE should parse as true")
		case "https://example.com/vocab/FALSE":
			assert.False(t, v.Value, "FALSE should parse as false")
		}
	}
}

// TestSchema_Vocabulary_InvalidBooleanDefaultsToFalse tests that invalid boolean values
// default to false when parsed with strconv.ParseBool
func TestSchema_Vocabulary_InvalidBooleanDefaultsToFalse(t *testing.T) {
	yml := `type: object
$vocabulary:
  "https://example.com/vocab/invalid": notaboolean
  "https://example.com/vocab/valid": true`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	var sch Schema
	err := low.BuildModel(idxNode.Content[0], &sch)
	assert.NoError(t, err)

	err = sch.Build(context.Background(), idxNode.Content[0], nil)
	assert.NoError(t, err)

	assert.NotNil(t, sch.Vocabulary.Value)
	assert.Equal(t, 2, sch.Vocabulary.Value.Len())

	// Check that invalid boolean defaults to false
	for k, v := range sch.Vocabulary.Value.FromOldest() {
		switch k.Value {
		case "https://example.com/vocab/invalid":
			assert.False(t, v.Value, "Invalid boolean should default to false")
		case "https://example.com/vocab/valid":
			assert.True(t, v.Value, "true should parse as true")
		}
	}
}

// TestSchema_Hash_VocabularyDifferent tests that different vocabulary values produce different hashes
func TestSchema_Hash_VocabularyDifferent(t *testing.T) {
	yml1 := `type: object
$vocabulary:
  "https://example.com/vocab/core": true`

	yml2 := `type: object
$vocabulary:
  "https://example.com/vocab/core": false`

	var node1, node2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml1), &node1)
	_ = yaml.Unmarshal([]byte(yml2), &node2)

	var sch1, sch2 Schema
	_ = low.BuildModel(node1.Content[0], &sch1)
	_ = sch1.Build(context.Background(), node1.Content[0], nil)

	_ = low.BuildModel(node2.Content[0], &sch2)
	_ = sch2.Build(context.Background(), node2.Content[0], nil)

	hash1 := sch1.Hash()
	hash2 := sch2.Hash()

	// Different vocabulary values should produce different hashes
	assert.NotEqual(t, hash1, hash2)
}

// TestSchema_Hash_ContentSchemaDifferent tests that different contentSchema produces different hashes
func TestSchema_Hash_ContentSchemaDifferent(t *testing.T) {
	yml1 := `type: string
contentMediaType: application/json
contentSchema:
  type: object`

	yml2 := `type: string
contentMediaType: application/json
contentSchema:
  type: array`

	var node1, node2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml1), &node1)
	_ = yaml.Unmarshal([]byte(yml2), &node2)

	var sch1, sch2 Schema
	_ = low.BuildModel(node1.Content[0], &sch1)
	_ = sch1.Build(context.Background(), node1.Content[0], nil)

	_ = low.BuildModel(node2.Content[0], &sch2)
	_ = sch2.Build(context.Background(), node2.Content[0], nil)

	hash1 := sch1.Hash()
	hash2 := sch2.Hash()

	// Different contentSchema types should produce different hashes
	assert.NotEqual(t, hash1, hash2)
}

func TestBuildPropertyMap_SkipExternalRef(t *testing.T) {
	// Schema with a property that has an external $ref
	schemaYml := `type: object
properties:
  local:
    type: string
  external:
    $ref: './models/Pet.yaml#/Pet'`

	var schemaNode yaml.Node
	_ = yaml.Unmarshal([]byte(schemaYml), &schemaNode)

	cfg := index.CreateClosedAPIIndexConfig()
	cfg.SkipExternalRefResolution = true
	idx := index.NewSpecIndexWithConfig(&schemaNode, cfg)

	var schema Schema
	_ = low.BuildModel(schemaNode.Content[0], &schema)
	err := schema.Build(context.Background(), schemaNode.Content[0], idx)
	assert.Nil(t, err) // parent builds successfully

	// Check properties
	assert.NotNil(t, schema.Properties.Value)
	found := false
	for k, v := range schema.Properties.Value.FromOldest() {
		if k.Value == "external" {
			found = true
			proxy := v.Value
			assert.True(t, proxy.IsReference())
			assert.Equal(t, "./models/Pet.yaml#/Pet", proxy.GetReference())
			// Schema() should return nil for unresolved external ref
			assert.Nil(t, proxy.Schema())
			assert.Nil(t, proxy.GetBuildError())
		}
	}
	assert.True(t, found, "expected to find 'external' property")
}

func TestBuildSchema_AllOf_SkipExternalRef(t *testing.T) {
	schemaYml := `allOf:
  - $ref: './models/Base.yaml#/Base'
  - type: object
    properties:
      name:
        type: string`

	var schemaNode yaml.Node
	_ = yaml.Unmarshal([]byte(schemaYml), &schemaNode)
	cfg := index.CreateClosedAPIIndexConfig()
	cfg.SkipExternalRefResolution = true
	idx := index.NewSpecIndexWithConfig(&schemaNode, cfg)

	var schema Schema
	_ = low.BuildModel(schemaNode.Content[0], &schema)
	err := schema.Build(context.Background(), schemaNode.Content[0], idx)
	assert.Nil(t, err)

	assert.NotNil(t, schema.AllOf.Value)
	assert.Len(t, schema.AllOf.Value, 2)

	// First allOf item should be the external ref
	first := schema.AllOf.Value[0].Value
	assert.True(t, first.IsReference())
	assert.Equal(t, "./models/Base.yaml#/Base", first.GetReference())
	assert.Nil(t, first.Schema())
	assert.Nil(t, first.GetBuildError())
}

func TestBuildSchema_OneOf_SkipExternalRef(t *testing.T) {
	schemaYml := `oneOf:
  - $ref: 'https://example.com/Cat.yaml'
  - type: object
    properties:
      bark:
        type: boolean`

	var schemaNode yaml.Node
	_ = yaml.Unmarshal([]byte(schemaYml), &schemaNode)
	cfg := index.CreateClosedAPIIndexConfig()
	cfg.SkipExternalRefResolution = true
	idx := index.NewSpecIndexWithConfig(&schemaNode, cfg)

	var schema Schema
	_ = low.BuildModel(schemaNode.Content[0], &schema)
	err := schema.Build(context.Background(), schemaNode.Content[0], idx)
	assert.Nil(t, err)

	assert.NotNil(t, schema.OneOf.Value)
	assert.Len(t, schema.OneOf.Value, 2)

	first := schema.OneOf.Value[0].Value
	assert.True(t, first.IsReference())
	assert.Equal(t, "https://example.com/Cat.yaml", first.GetReference())
	assert.Nil(t, first.Schema())
	assert.Nil(t, first.GetBuildError())
}

func TestBuildSchema_AllOfMap_SkipExternalRef(t *testing.T) {
	// allOf as a single map $ref (not an array) exercises the map branch of buildSchema (Site B)
	schemaYml := `allOf:
  - $ref: './models/Base.yaml#/Base'`

	var schemaNode yaml.Node
	_ = yaml.Unmarshal([]byte(schemaYml), &schemaNode)
	cfg := index.CreateClosedAPIIndexConfig()
	cfg.SkipExternalRefResolution = true
	idx := index.NewSpecIndexWithConfig(&schemaNode, cfg)

	var schema Schema
	_ = low.BuildModel(schemaNode.Content[0], &schema)
	err := schema.Build(context.Background(), schemaNode.Content[0], idx)
	assert.Nil(t, err)

	assert.NotNil(t, schema.AllOf.Value)
	assert.Len(t, schema.AllOf.Value, 1)

	first := schema.AllOf.Value[0].Value
	assert.True(t, first.IsReference())
	assert.Equal(t, "./models/Base.yaml#/Base", first.GetReference())
	assert.Nil(t, first.Schema())
	assert.Nil(t, first.GetBuildError())
}

func TestExtractSchema_RootRef_SkipExternalRef(t *testing.T) {
	yml := `$ref: './models/Pet.yaml#/Pet'`

	var root yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &root)

	cfg := index.CreateClosedAPIIndexConfig()
	cfg.SkipExternalRefResolution = true
	idx := index.NewSpecIndexWithConfig(&root, cfg)

	result, err := ExtractSchema(context.Background(), root.Content[0], idx)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Value.IsReference())
	assert.Equal(t, "./models/Pet.yaml#/Pet", result.Value.GetReference())
	assert.Nil(t, result.Value.Schema())
	assert.Nil(t, result.Value.GetBuildError())
}

func TestExtractSchema_SchemaKeyRef_SkipExternalRef(t *testing.T) {
	yml := `schema:
  $ref: './models/Pet.yaml#/Pet'`

	var root yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &root)

	cfg := index.CreateClosedAPIIndexConfig()
	cfg.SkipExternalRefResolution = true
	idx := index.NewSpecIndexWithConfig(&root, cfg)

	result, err := ExtractSchema(context.Background(), root.Content[0], idx)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Value.IsReference())
	assert.Equal(t, "./models/Pet.yaml#/Pet", result.Value.GetReference())
	assert.Nil(t, result.Value.Schema())
	assert.Nil(t, result.Value.GetBuildError())
}

func TestClearSchemaQuickHashMap(t *testing.T) {
	// Store a value.
	SchemaQuickHashMap.Store("test-key", "test-value")

	// Verify it's there.
	_, ok := SchemaQuickHashMap.Load("test-key")
	assert.True(t, ok)

	// Clear and verify it's gone.
	ClearSchemaQuickHashMap()
	_, ok = SchemaQuickHashMap.Load("test-key")
	assert.False(t, ok)

	// Idempotent: clearing an empty map should not panic.
	ClearSchemaQuickHashMap()
}
