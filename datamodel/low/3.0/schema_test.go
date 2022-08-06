package v3

import (
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
	mbErr := BuildModel(&rootNode, &sch)
	assert.NoError(t, mbErr)

	schErr := sch.Build(rootNode.Content[0], 0)
	assert.NoError(t, schErr)
	assert.Equal(t, "something object", sch.Description.Value)
	assert.True(t, sch.AdditionalProperties.Value.(bool))

	assert.Len(t, sch.Properties.Value, 2)
	v := sch.FindProperty("somethingB")

	assert.Equal(t, "https://pb33f.io", v.Value.ExternalDocs.Value.URL.Value)
	assert.Equal(t, "the best docs", v.Value.ExternalDocs.Value.Description.Value)

	j := v.Value.FindProperty("somethingBProp")
	assert.NotNil(t, j.Value)
	assert.NotNil(t, j.Value.XML.Value)
	assert.Equal(t, "an xml thing", j.Value.XML.Value.Name.Value)
	assert.Equal(t, "an xml namespace", j.Value.XML.Value.Namespace.Value)
	assert.Equal(t, "a prefix", j.Value.XML.Value.Prefix.Value)
	assert.Equal(t, true, j.Value.XML.Value.Attribute.Value)
	assert.Len(t, j.Value.XML.Value.Extensions, 1)

	assert.NotNil(t, v.Value.AdditionalProperties.Value)

	var addProps map[string]interface{}
	v.Value.AdditionalProperties.ValueNode.Decode(&addProps)
	assert.Equal(t, "yes", addProps["why"])
	assert.Equal(t, true, addProps["thatIs"])

	// check polymorphic values allOf
	assert.Equal(t, "an allof thing", sch.AllOf.Value[0].Value.Description.Value)
	assert.Len(t, sch.AllOf.Value[0].Value.Properties.Value, 2)

	v = sch.AllOf.Value[0].Value.FindProperty("allOfA")
	assert.NotNil(t, v)
	assert.Equal(t, "allOfA description", v.Value.Description.Value)
	assert.Equal(t, "allOfAExp", v.Value.Example.Value)

	v = sch.AllOf.Value[0].Value.FindProperty("allOfB")
	assert.NotNil(t, v)
	assert.Equal(t, "allOfB description", v.Value.Description.Value)
	assert.Equal(t, "allOfBExp", v.Value.Example.Value)

	// check polymorphic values anyOf
	assert.Equal(t, "an anyOf thing", sch.AnyOf.Value[0].Value.Description.Value)
	assert.Len(t, sch.AnyOf.Value[0].Value.Properties.Value, 2)

	v = sch.AnyOf.Value[0].Value.FindProperty("anyOfA")
	assert.NotNil(t, v)
	assert.Equal(t, "anyOfA description", v.Value.Description.Value)
	assert.Equal(t, "anyOfAExp", v.Value.Example.Value)

	v = sch.AnyOf.Value[0].Value.FindProperty("anyOfB")
	assert.NotNil(t, v)
	assert.Equal(t, "anyOfB description", v.Value.Description.Value)
	assert.Equal(t, "anyOfBExp", v.Value.Example.Value)

	// check polymorphic values oneOf
	assert.Equal(t, "a oneof thing", sch.OneOf.Value[0].Value.Description.Value)
	assert.Len(t, sch.OneOf.Value[0].Value.Properties.Value, 2)

	v = sch.OneOf.Value[0].Value.FindProperty("oneOfA")
	assert.NotNil(t, v)
	assert.Equal(t, "oneOfA description", v.Value.Description.Value)
	assert.Equal(t, "oneOfAExp", v.Value.Example.Value)

	v = sch.OneOf.Value[0].Value.FindProperty("oneOfB")
	assert.NotNil(t, v)
	assert.Equal(t, "oneOfB description", v.Value.Description.Value)
	assert.Equal(t, "oneOfBExp", v.Value.Example.Value)

	// check values NOT
	assert.Equal(t, "a not thing", sch.Not.Value[0].Value.Description.Value)
	assert.Len(t, sch.Not.Value[0].Value.Properties.Value, 2)

	v = sch.Not.Value[0].Value.FindProperty("notA")
	assert.NotNil(t, v)
	assert.Equal(t, "notA description", v.Value.Description.Value)
	assert.Equal(t, "notAExp", v.Value.Example.Value)

	v = sch.Not.Value[0].Value.FindProperty("notB")
	assert.NotNil(t, v)
	assert.Equal(t, "notB description", v.Value.Description.Value)
	assert.Equal(t, "notBExp", v.Value.Example.Value)

	// check values Items
	assert.Equal(t, "an items thing", sch.Items.Value[0].Value.Description.Value)
	assert.Len(t, sch.Items.Value[0].Value.Properties.Value, 2)

	v = sch.Items.Value[0].Value.FindProperty("itemsA")
	assert.NotNil(t, v)
	assert.Equal(t, "itemsA description", v.Value.Description.Value)
	assert.Equal(t, "itemsAExp", v.Value.Example.Value)

	v = sch.Items.Value[0].Value.FindProperty("itemsB")
	assert.NotNil(t, v)
	assert.Equal(t, "itemsB description", v.Value.Description.Value)
	assert.Equal(t, "itemsBExp", v.Value.Example.Value)

	// check discriminator
	assert.NotNil(t, sch.Discriminator.Value)
	assert.Equal(t, "athing", sch.Discriminator.Value.PropertyName.Value)
	assert.Len(t, sch.Discriminator.Value.Mapping, 2)
	mv := sch.Discriminator.Value.FindMappingValue("log")
	assert.Equal(t, "cat", mv.Value)
	mv = sch.Discriminator.Value.FindMappingValue("pizza")
	assert.Equal(t, "party", mv.Value)
}
