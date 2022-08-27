package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func Test_Schema(t *testing.T) {
	testSpec := `type: object
description: something object
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
properties:
  somethingA:
    type: number
    description: a number
    example: 2
  somethingB:
    type: object
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
additionalProperties: true      `

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(testSpec), &rootNode)
	assert.NoError(t, mErr)

	sch := Schema{}
	mbErr := low.BuildModel(&rootNode, &sch)
	assert.NoError(t, mbErr)

	schErr := sch.Build(rootNode.Content[0], nil)
	assert.NoError(t, schErr)
	assert.Equal(t, "something object", sch.Description.Value)
	assert.True(t, sch.AdditionalProperties.Value.(bool))

	assert.Len(t, sch.Properties.Value, 2)
	v := sch.FindProperty("somethingB")

	assert.Equal(t, "https://pb33f.io", v.Value.Schema().ExternalDocs.Value.URL.Value)
	assert.Equal(t, "the best docs", v.Value.Schema().ExternalDocs.Value.Description.Value)

	j := v.Value.Schema().FindProperty("somethingBProp").Value.Schema()
	assert.NotNil(t, j)
	assert.NotNil(t, j.XML.Value)
	assert.Equal(t, "an xml thing", j.XML.Value.Name.Value)
	assert.Equal(t, "an xml namespace", j.XML.Value.Namespace.Value)
	assert.Equal(t, "a prefix", j.XML.Value.Prefix.Value)
	assert.Equal(t, true, j.XML.Value.Attribute.Value)
	assert.Len(t, j.XML.Value.Extensions, 1)

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
	assert.Equal(t, "a not thing", sch.Not.Value[0].Value.Schema().Description.Value)
	assert.Len(t, sch.Not.Value[0].Value.Schema().Properties.Value, 2)

	v = sch.Not.Value[0].Value.Schema().FindProperty("notA")
	assert.NotNil(t, v)
	assert.Equal(t, "notA description", v.Value.Schema().Description.Value)
	assert.Equal(t, "notAExp", v.Value.Schema().Example.Value)

	v = sch.Not.Value[0].Value.Schema().FindProperty("notB")
	assert.NotNil(t, v)
	assert.Equal(t, "notB description", v.Value.Schema().Description.Value)
	assert.Equal(t, "notBExp", v.Value.Schema().Example.Value)

	// check values Items
	assert.Equal(t, "an items thing", sch.Items.Value[0].Value.Schema().Description.Value)
	assert.Len(t, sch.Items.Value[0].Value.Schema().Properties.Value, 2)

	v = sch.Items.Value[0].Value.Schema().FindProperty("itemsA")
	assert.NotNil(t, v)
	assert.Equal(t, "itemsA description", v.Value.Schema().Description.Value)
	assert.Equal(t, "itemsAExp", v.Value.Schema().Example.Value)

	v = sch.Items.Value[0].Value.Schema().FindProperty("itemsB")
	assert.NotNil(t, v)
	assert.Equal(t, "itemsB description", v.Value.Schema().Description.Value)
	assert.Equal(t, "itemsBExp", v.Value.Schema().Example.Value)

	// check discriminator
	assert.NotNil(t, sch.Discriminator.Value)
	assert.Equal(t, "athing", sch.Discriminator.Value.PropertyName.Value)
	assert.Len(t, sch.Discriminator.Value.Mapping, 2)
	mv := sch.Discriminator.Value.FindMappingValue("log")
	assert.Equal(t, "cat", mv.Value)
	mv = sch.Discriminator.Value.FindMappingValue("pizza")
	assert.Equal(t, "party", mv.Value)
}

//func TestSchema_BuildLevel_TooDeep(t *testing.T) {
//
//	// if you design data models like this, you're doing it fucking wrong. Seriously. why, what is so complex about a model
//	// that it needs to be 30+ levels deep? I have seen this shit in the wild, it's unreadable, un-parsable garbage.
//	yml := `type: object
//properties:
//  aValue:
//    type: object
//    properties:
//      aValue:
//        type: object
//        properties:
//          aValue:
//            type: object
//            properties:
//              aValue:
//                type: object
//                properties:
//                  aValue:
//                    type: object
//                    properties:
//                      aValue:
//                        type: object
//                        properties:
//                          aValue:
//                            type: object
//                            properties:
//                              aValue:
//                                type: object
//                                properties:
//                                  aValue:
//                                    type: object
//                                    properties:
//                                      aValue:
//                                        type: object
//                                        properties:
//                                          aValue:
//                                            type: object
//                                            properties:
//                                              aValue:
//                                                type: object
//                                                properties:
//                                                  aValue:
//                                                    type: object
//                                                    properties:
//                                                      aValue:
//                                                        type: object
//                                                        properties:
//                                                          aValue:
//                                                            type: object
//                                                            properties:
//                                                              aValue:
//                                                                type: object
//                                                                properties:
//                                                                  aValue:
//                                                                    type: object
//                                                                    properties:
//                                                                      aValue:
//                                                                        type: object
//                                                                        properties:
//                                                                          aValue:
//                                                                            type: object
//                                                                            properties:
//                                                                              aValue:
//                                                                                type: object
//                                                                                properties:
//                                                                                  aValue:
//                                                                                    type: object
//                                                                                    properties:
//                                                                                      aValue:
//                                                                                        type: object
//                                                                                        properties:
//                                                                                          aValue:
//                                                                                            type: object
//                                                                                            properties:
//                                                                                              aValue:
//                                                                                                type: object
//                                                                                                properties:
//                                                                                                  aValue:
//                                                                                                    type: object
//                                                                                                    properties:
//                                                                                                      aValue:
//                                                                                                        type: object
//                                                                                                        properties:
//                                                                                                          aValue:
//                                                                                                            type: object
//                                                                                                            properties:
//                                                                                                              aValue:
//                                                                                                                type: object
//                                                                                                                properties:
//                                                                                                                  aValue:
//                                                                                                                    type: object
//                                                                                                                    properties:
//                                                                                                                      aValue:
//                                                                                                                        type: object
//                                                                                                                        properties:
//                                                                                                                          aValue:
//                                                                                                                            type: object
//                                                                                                                            properties:
//                                                                                                                              aValue:
//                                                                                                                                type: object
//                                                                                                                                properties:
//                                                                                                                                  aValue:
//                                                                                                                                    type: object`
//
//	var idxNode yaml.Node
//	_ = yaml.Unmarshal([]byte(yml), &idxNode)
//	idx := index.NewSpecIndex(&idxNode)
//
//	var n Schema
//	err := low.BuildModel(&idxNode, &n)
//	assert.NoError(t, err)
//
//	err = n.Build(idxNode.Content[0], idx)
//	assert.Error(t, err)
//
//}

func TestSchema_Build_ErrorAdditionalProps(t *testing.T) {

	yml := `additionalProperties:
  $ref: #borko`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Schema
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.Error(t, err)

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
	assert.Equal(t, desc, sch.Not.Value[0].Value.Schema().Description.Value)
	assert.Equal(t, desc, sch.Items.Value[0].Value.Schema().Description.Value)
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
	assert.Equal(t, desc, sch.Not.Value[0].Value.Schema().Description.Value)
	assert.Equal(t, desc, sch.Items.Value[0].Value.Schema().Description.Value)
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

	_ = sch.Build(idxNode.Content[0], idx)
	assert.Nil(t, sch.AllOf.Value[0].Value.Schema())

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

	s, _ := ExtractSchema(idxNode.Content[0], idx)

	b := s.Value.Schema()
	assert.Nil(t, b)
	assert.Error(t, s.Value.GetBuildError())

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
