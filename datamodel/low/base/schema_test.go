package base

import (
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/resolver"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
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
  type: object
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
  type: object
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
$anchor: anchor`
}

func Test_Schema(t *testing.T) {
	testSpec := test_get_schema_blob()

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(testSpec), &rootNode)
	assert.NoError(t, mErr)

	sch := Schema{}
	mbErr := low.BuildModel(rootNode.Content[0], &sch)
	assert.NoError(t, mbErr)

	schErr := sch.Build(rootNode.Content[0], nil)
	assert.NoError(t, schErr)
	assert.Equal(t, "something object", sch.Description.Value)
	assert.True(t, sch.AdditionalProperties.Value.(bool))

	assert.Len(t, sch.Properties.Value, 2)
	v := sch.FindProperty("somethingB")

	assert.Equal(t, "https://pb33f.io", v.Value.Schema().ExternalDocs.Value.URL.Value)
	assert.Equal(t, "the best docs", v.Value.Schema().ExternalDocs.Value.Description.Value)

	assert.True(t, v.Value.Schema().ExclusiveMinimum.Value.A)
	assert.True(t, v.Value.Schema().ExclusiveMaximum.Value.A)

	j := v.Value.Schema().FindProperty("somethingBProp").Value.Schema()
	k := v.Value.Schema().FindProperty("somethingBProp").Value

	assert.Equal(t, k, j.ParentProxy)

	assert.NotNil(t, j)
	assert.NotNil(t, j.XML.Value)
	assert.Equal(t, "an xml thing", j.XML.Value.Name.Value)
	assert.Equal(t, "an xml namespace", j.XML.Value.Namespace.Value)
	assert.Equal(t, "a prefix", j.XML.Value.Prefix.Value)
	assert.Equal(t, true, j.XML.Value.Attribute.Value)
	assert.Len(t, j.XML.Value.Extensions, 1)
	assert.Len(t, j.XML.Value.GetExtensions(), 1)

	assert.NotNil(t, v.Value.Schema().AdditionalProperties.Value)

	var addProps map[string]interface{}
	v.Value.Schema().AdditionalProperties.ValueNode.Decode(&addProps)
	assert.Equal(t, "yes", addProps["why"])
	assert.Equal(t, true, addProps["thatIs"])

	// check polymorphic values allOf
	f := sch.AllOf.Value[0].Value.Schema()
	assert.Equal(t, "an allof thing", f.Description.Value)
	assert.Len(t, f.Properties.Value, 2)

	v = f.FindProperty("allOfA")
	assert.NotNil(t, v)

	io := v.Value.Schema()

	assert.Equal(t, "allOfA description", io.Description.Value)
	assert.Equal(t, "allOfAExp", io.Example.Value)

	qw := f.FindProperty("allOfB").Value.Schema()
	assert.NotNil(t, v)
	assert.Equal(t, "allOfB description", qw.Description.Value)
	assert.Equal(t, "allOfBExp", qw.Example.Value)

	// check polymorphic values anyOf
	assert.Equal(t, "an anyOf thing", sch.AnyOf.Value[0].Value.Schema().Description.Value)
	assert.Len(t, sch.AnyOf.Value[0].Value.Schema().Properties.Value, 2)

	v = sch.AnyOf.Value[0].Value.Schema().FindProperty("anyOfA")
	assert.NotNil(t, v)
	assert.Equal(t, "anyOfA description", v.Value.Schema().Description.Value)
	assert.Equal(t, "anyOfAExp", v.Value.Schema().Example.Value)

	v = sch.AnyOf.Value[0].Value.Schema().FindProperty("anyOfB")
	assert.NotNil(t, v)
	assert.Equal(t, "anyOfB description", v.Value.Schema().Description.Value)
	assert.Equal(t, "anyOfBExp", v.Value.Schema().Example.Value)

	// check polymorphic values oneOf
	assert.Equal(t, "a oneof thing", sch.OneOf.Value[0].Value.Schema().Description.Value)
	assert.Len(t, sch.OneOf.Value[0].Value.Schema().Properties.Value, 2)

	v = sch.OneOf.Value[0].Value.Schema().FindProperty("oneOfA")
	assert.NotNil(t, v)
	assert.Equal(t, "oneOfA description", v.Value.Schema().Description.Value)
	assert.Equal(t, "oneOfAExp", v.Value.Schema().Example.Value)

	v = sch.OneOf.Value[0].Value.Schema().FindProperty("oneOfB")
	assert.NotNil(t, v)
	assert.Equal(t, "oneOfB description", v.Value.Schema().Description.Value)
	assert.Equal(t, "oneOfBExp", v.Value.Schema().Example.Value)

	// check values NOT
	assert.Equal(t, "a not thing", sch.Not.Value.Schema().Description.Value)
	assert.Len(t, sch.Not.Value.Schema().Properties.Value, 2)

	v = sch.Not.Value.Schema().FindProperty("notA")
	assert.NotNil(t, v)
	assert.Equal(t, "notA description", v.Value.Schema().Description.Value)
	assert.Equal(t, "notAExp", v.Value.Schema().Example.Value)

	v = sch.Not.Value.Schema().FindProperty("notB")
	assert.NotNil(t, v)
	assert.Equal(t, "notB description", v.Value.Schema().Description.Value)
	assert.Equal(t, "notBExp", v.Value.Schema().Example.Value)

	// check values Items
	assert.Equal(t, "an items thing", sch.Items.Value.A.Schema().Description.Value)
	assert.Len(t, sch.Items.Value.A.Schema().Properties.Value, 2)

	v = sch.Items.Value.A.Schema().FindProperty("itemsA")
	assert.NotNil(t, v)
	assert.Equal(t, "itemsA description", v.Value.Schema().Description.Value)
	assert.Equal(t, "itemsAExp", v.Value.Schema().Example.Value)

	v = sch.Items.Value.A.Schema().FindProperty("itemsB")
	assert.NotNil(t, v)
	assert.Equal(t, "itemsB description", v.Value.Schema().Description.Value)
	assert.Equal(t, "itemsBExp", v.Value.Schema().Example.Value)

	// check values PrefixItems
	assert.Equal(t, "an items thing", sch.PrefixItems.Value[0].Value.Schema().Description.Value)
	assert.Len(t, sch.PrefixItems.Value[0].Value.Schema().Properties.Value, 2)

	v = sch.PrefixItems.Value[0].Value.Schema().FindProperty("itemsA")
	assert.NotNil(t, v)
	assert.Equal(t, "itemsA description", v.Value.Schema().Description.Value)
	assert.Equal(t, "itemsAExp", v.Value.Schema().Example.Value)

	v = sch.PrefixItems.Value[0].Value.Schema().FindProperty("itemsB")
	assert.NotNil(t, v)
	assert.Equal(t, "itemsB description", v.Value.Schema().Description.Value)
	assert.Equal(t, "itemsBExp", v.Value.Schema().Example.Value)

	// check discriminator
	assert.NotNil(t, sch.Discriminator.Value)
	assert.Equal(t, "athing", sch.Discriminator.Value.PropertyName.Value)
	assert.Len(t, sch.Discriminator.Value.Mapping.Value, 2)
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

	schErr := sch.Build(rootNode.Content[0], nil)
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
	_ = sch1.Build(sc1n.Content[0], nil)

	var sc2n yaml.Node
	_ = yaml.Unmarshal([]byte(testSpec), &sc2n)
	sch2 := Schema{}
	_ = low.BuildModel(&sc2n, &sch2)
	_ = sch2.Build(sc2n.Content[0], nil)

	assert.Equal(t, sch1.Hash(), sch2.Hash())
}

func BenchmarkSchema_Hash(b *testing.B) {
	// create two versions
	testSpec := test_get_schema_blob()
	var sc1n yaml.Node
	_ = yaml.Unmarshal([]byte(testSpec), &sc1n)
	sch1 := Schema{}
	_ = low.BuildModel(&sc1n, &sch1)
	_ = sch1.Build(sc1n.Content[0], nil)

	var sc2n yaml.Node
	_ = yaml.Unmarshal([]byte(testSpec), &sc2n)
	sch2 := Schema{}
	_ = low.BuildModel(&sc2n, &sch2)
	_ = sch2.Build(sc2n.Content[0], nil)

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
  - testing`

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(testSpec), &rootNode)
	assert.NoError(t, mErr)

	sch := Schema{}
	mbErr := low.BuildModel(rootNode.Content[0], &sch)
	assert.NoError(t, mbErr)

	schErr := sch.Build(rootNode.Content[0], nil)
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
	assert.Equal(t, "testing", sch.Examples.Value[0].Value)
	assert.Equal(t, "fish64", sch.ContentEncoding.Value)
	assert.Equal(t, "fish/paste", sch.ContentMediaType.Value)
	assert.True(t, sch.Items.Value.IsB())
	assert.True(t, sch.Items.Value.B)
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
	err := n.Build(idxNode.Content[0], idx)
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
	err := n.Build(idxNode.Content[0], idx)
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
	err := n.Build(idxNode.Content[0], idx)
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
	err := n.Build(idxNode.Content[0], idx)
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
  - $ref: '#/components/schemas/Something'
items:
  - $ref: '#/components/schemas/Something'`

	var sch Schema
	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	err := low.BuildModel(&idxNode, &sch)
	assert.NoError(t, err)

	schErr := sch.Build(idxNode.Content[0], idx)
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

	schErr := sch.Build(idxNode.Content[0], idx)
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
  $ref: '#/components/schemas/Something'
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

	schErr := sch.Build(idxNode.Content[0], idx)
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

	schErr := sch.Build(idxNode.Content[0], idx)
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

	schErr := sch.Build(idxNode.Content[0], idx)
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

	schErr := sch.Build(idxNode.Content[0], idx)
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

	schErr := sch.Build(idxNode.Content[0], idx)
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
  $ref: '#/components/schemas/Something'`

	var sch Schema
	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	err := low.BuildModel(&idxNode, &sch)
	assert.NoError(t, err)

	schErr := sch.Build(idxNode.Content[0], idx)
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

	err = sch.Build(idxNode.Content[0], idx)
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

	schErr := sch.Build(idxNode.Content[0], idx)
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

	resolve := resolver.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	err := low.BuildModel(&idxNode, &sch)
	assert.NoError(t, err)

	schErr := sch.Build(idxNode.Content[0], idx)
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

	resolve := resolver.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	err := low.BuildModel(&idxNode, &sch)
	assert.NoError(t, err)

	schErr := sch.Build(idxNode.Content[0], idx)
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

	res, err := ExtractSchema(idxNode.Content[0], idx)
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

	res, err := ExtractSchema(idxNode.Content[0], nil)
	assert.NoError(t, err)
	assert.NotNil(t, res.Value)
	sch := res.Value.Schema()
	assert.Equal(t, 5, sch.Default.Value)
}

func TestExtractSchema_ConstPrimitive(t *testing.T) {
	yml := `
schema: 
  type: object
  const: 5`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	res, err := ExtractSchema(idxNode.Content[0], nil)
	assert.NoError(t, err)
	assert.NotNil(t, res.Value)
	sch := res.Value.Schema()
	assert.Equal(t, 5, sch.Const.Value)
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

	res, err := ExtractSchema(idxNode.Content[0], idx)
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

	_, err := ExtractSchema(idxNode.Content[0], idx)
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

	resolve := resolver.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	res, err := ExtractSchema(idxNode.Content[0], idx)
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

	res, err := ExtractSchema(idxNode.Content[0], idx)
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

	_, err := ExtractSchema(idxNode.Content[0], idx)
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

	_, err := ExtractSchema(idxNode.Content[0], idx)
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

	res, err := ExtractSchema(idxNode.Content[0], idx)

	assert.NotNil(t, res.Value.Schema().AdditionalProperties.Value.(*SchemaProxy).Schema())
	assert.Nil(t, err)
}

func TestExtractSchema_AdditionalPropertiesAsSchemaSlice(t *testing.T) {
	yml := `components:
  schemas:
    Something:
      additionalProperties:
        - nice: rice`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `$ref: '#/components/schemas/Something'`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	res, err := ExtractSchema(idxNode.Content[0], idx)

	assert.NotNil(t, res.Value.Schema().AdditionalProperties.Value.([]low.ValueReference[interface{}]))
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

	res, err := ExtractSchema(idxNode.Content[0], idx)
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

	res, err := ExtractSchema(idxNode.Content[0], idx)
	assert.NotNil(t, res.Value.Schema().AdditionalProperties.Value.(*SchemaProxy).Schema())
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

	res, err := ExtractSchema(idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "a frosty cold beverage can be coke or sprite",
		res.Value.Schema().OneOf.Value[0].Value.Schema().Description.Value)
}

func TestSchema_Hash_Equal(t *testing.T) {
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
  additionalProperties: anything
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
  additionalProperties: anything
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

	lDoc, _ := ExtractSchema(lNode.Content[0], nil)
	rDoc, _ := ExtractSchema(rNode.Content[0], nil)

	assert.NotNil(t, lDoc)
	assert.NotNil(t, rDoc)

	lHash := lDoc.Value.Schema().Hash()
	rHash := rDoc.Value.Schema().Hash()

	assert.Equal(t, lHash, rHash)
}

func TestSchema_Hash_AdditionalPropsSlice(t *testing.T) {
	left := `schema:
  additionalProperties:
    - type: string`

	right := `schema:
  additionalProperties:
    - type: string`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	lDoc, _ := ExtractSchema(lNode.Content[0], nil)
	rDoc, _ := ExtractSchema(rNode.Content[0], nil)

	assert.NotNil(t, lDoc)
	assert.NotNil(t, rDoc)

	lHash := lDoc.Value.Schema().Hash()
	rHash := rDoc.Value.Schema().Hash()

	assert.Equal(t, lHash, rHash)
}

func TestSchema_Hash_AdditionalPropsSliceNoMap(t *testing.T) {
	left := `schema:
  additionalProperties:
    - hello`

	right := `schema:
  additionalProperties:
    - hello`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	lDoc, _ := ExtractSchema(lNode.Content[0], nil)
	rDoc, _ := ExtractSchema(rNode.Content[0], nil)

	assert.NotNil(t, lDoc)
	assert.NotNil(t, rDoc)

	lHash := lDoc.Value.Schema().Hash()
	rHash := rDoc.Value.Schema().Hash()

	assert.Equal(t, lHash, rHash)
}

func TestSchema_Hash_NotEqual(t *testing.T) {
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

	lDoc, _ := ExtractSchema(lNode.Content[0], nil)
	rDoc, _ := ExtractSchema(rNode.Content[0], nil)

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

	lDoc, _ := ExtractSchema(lNode.Content[0], nil)
	rDoc, _ := ExtractSchema(rNode.Content[0], nil)
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

	res, _ := ExtractSchema(idxNode.Content[0], idx)

	assert.True(t, res.Value.Schema().UnevaluatedProperties.Value.IsB())
	assert.True(t, *res.Value.Schema().UnevaluatedProperties.Value.B)

	assert.Equal(t, "571bd1853c22393131e2dcadce86894da714ec14968895c8b7ed18154b2be8cd",
		low.GenerateHashString(res.Value.Schema().UnevaluatedProperties.Value))
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

	res, _ := ExtractSchema(idxNode.Content[0], idx)

	assert.True(t, res.Value.Schema().UnevaluatedProperties.Value.IsB())
	assert.False(t, *res.Value.Schema().UnevaluatedProperties.Value.B)
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

	res, _ := ExtractSchema(idxNode.Content[0], idx)

	assert.Nil(t, res.Value.Schema().UnevaluatedProperties.Value)
}
