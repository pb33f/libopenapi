// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"fmt"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestNewSchemaProxy(t *testing.T) {

	// check proxy
	yml := `components:
    schemas:
     rice:
       type: string
     nice:
       properties:
         rice:
           $ref: '#/components/schemas/rice'
     ice:
       properties:
         rice:
           $ref: '#/components/schemas/rice'`

	var idxNode, compNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	yml = `properties:
    rice:
     $ref: '#/components/schemas/I-do-not-exist'`

	_ = yaml.Unmarshal([]byte(yml), &compNode)

	sp := new(lowbase.SchemaProxy)
	err := sp.Build(compNode.Content[0], idx)
	assert.NoError(t, err)

	lowproxy := low.NodeReference[*lowbase.SchemaProxy]{
		Value:     sp,
		ValueNode: idxNode.Content[0],
	}

	sch1 := SchemaProxy{schema: &lowproxy}
	assert.Nil(t, sch1.Schema())
	assert.Error(t, sch1.GetBuildError())

	g, o := sch1.BuildSchema()
	assert.Nil(t, g)
	assert.Error(t, o)

}

func TestNewSchemaProxy_WithObject(t *testing.T) {

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
  somethingBee:
    type: number
  somethingThree:
    type: number
  somethingTwo:
    type: number
  somethingOne:
    type: number
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
additionalProperties: true
xml:
  name: XML Thing
externalDocs:
  url: https://pb33f.io/docs
enum: [fish, cake]
required: [cake, fish]
maxLength: 10
minLength: 1
maxItems: 10
minItems: 1
maxProperties: 10
minProperties: 1
nullable: true
readOnly: true
writeOnly: false
deprecated: true`

	var compNode yaml.Node
	_ = yaml.Unmarshal([]byte(testSpec), &compNode)

	sp := new(lowbase.SchemaProxy)
	err := sp.Build(compNode.Content[0], nil)
	assert.NoError(t, err)

	lowproxy := low.NodeReference[*lowbase.SchemaProxy]{
		Value:     sp,
		ValueNode: compNode.Content[0],
	}

	schemaProxy := NewSchemaProxy(&lowproxy)
	compiled := schemaProxy.Schema()

	assert.Equal(t, schemaProxy, compiled.ParentProxy)

	assert.NotNil(t, compiled)
	assert.Nil(t, schemaProxy.GetBuildError())

	wentLow := compiled.GoLow()
	assert.Equal(t, 114, wentLow.AdditionalProperties.ValueNode.Line)

}

func TestNewSchemaProxy_WithObject_FinishPoly(t *testing.T) {

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
  somethingB:
    exclusiveMinimum: 123
    exclusiveMaximum: 334
    type: object
    description: an object
    externalDocs:
      description: the best docs
      url: https://pb33f.io
    properties:
      somethingBProp:
        exclusiveMinimum: 3
        exclusiveMaximum: 120
        type: 
         - string
         - null
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
exclusiveMaximum: true
exclusiveMinimum: false
xml:
  name: XML Thing
externalDocs:
  url: https://pb33f.io/docs
enum: [fish, cake]
required: [cake, fish]`

	var compNode yaml.Node
	_ = yaml.Unmarshal([]byte(testSpec), &compNode)

	sp := new(lowbase.SchemaProxy)
	err := sp.Build(compNode.Content[0], nil)
	assert.NoError(t, err)

	lowproxy := low.NodeReference[*lowbase.SchemaProxy]{
		Value:     sp,
		ValueNode: compNode.Content[0],
	}

	schemaProxy := NewSchemaProxy(&lowproxy)
	compiled := schemaProxy.Schema()

	assert.NotNil(t, compiled)
	assert.Nil(t, schemaProxy.GetBuildError())

	assert.True(t, *compiled.ExclusiveMaximumBool)
	assert.False(t, *compiled.ExclusiveMinimumBool)
	assert.Equal(t, int64(123), *compiled.Properties["somethingB"].Schema().ExclusiveMinimum)
	assert.Equal(t, int64(334), *compiled.Properties["somethingB"].Schema().ExclusiveMaximum)
	assert.Len(t, compiled.Properties["somethingB"].Schema().Properties["somethingBProp"].Schema().Type, 2)

	wentLow := compiled.GoLow()
	assert.Equal(t, 96, wentLow.AdditionalProperties.ValueNode.Line)
	assert.Equal(t, 100, wentLow.XML.ValueNode.Line)

	wentLower := compiled.XML.GoLow()
	assert.Equal(t, 100, wentLower.Name.ValueNode.Line)

}

func TestSchemaProxy_GoLow(t *testing.T) {

	const ymlComponents = `components:
    schemas:
     rice:
       type: string
     nice:
       properties:
         rice:
           $ref: '#/components/schemas/rice'
     ice:
       properties:
         rice:
           $ref: '#/components/schemas/rice'`

	idx := func() *index.SpecIndex {
		var idxNode yaml.Node
		err := yaml.Unmarshal([]byte(ymlComponents), &idxNode)
		assert.NoError(t, err)
		return index.NewSpecIndex(&idxNode)
	}()

	const ref = "#/components/schemas/nice"
	const ymlSchema = `$ref: '` + ref + `'`
	var node yaml.Node
	_ = yaml.Unmarshal([]byte(ymlSchema), &node)

	lowProxy := new(lowbase.SchemaProxy)
	err := lowProxy.Build(node.Content[0], idx)
	assert.NoError(t, err)

	lowRef := low.NodeReference[*lowbase.SchemaProxy]{
		Value: lowProxy,
	}

	sp := NewSchemaProxy(&lowRef)
	assert.Equal(t, lowProxy, sp.GoLow())
	assert.Equal(t, ref, sp.GoLow().GetSchemaReference())

	spNil := NewSchemaProxy(nil)
	assert.Nil(t, spNil.GoLow())
}

func getHighSchema(t *testing.T, yml string) *Schema {
	// unmarshal raw bytes
	var node yaml.Node
	assert.NoError(t, yaml.Unmarshal([]byte(yml), &node))

	// build out the low-level model
	var lowSchema lowbase.Schema
	assert.NoError(t, low.BuildModel(node.Content[0], &lowSchema))
	assert.NoError(t, lowSchema.Build(node.Content[0], nil))

	// build the high level model
	return NewSchema(&lowSchema)
}

func TestSchemaNumberNoValidation(t *testing.T) {
	yml := `
type: number
`
	highSchema := getHighSchema(t, yml)

	assert.Nil(t, highSchema.MultipleOf)
	assert.Nil(t, highSchema.Minimum)
	assert.Nil(t, highSchema.ExclusiveMinimum)
	assert.Nil(t, highSchema.ExclusiveMinimumBool)
	assert.Nil(t, highSchema.Maximum)
	assert.Nil(t, highSchema.ExclusiveMaximum)
	assert.Nil(t, highSchema.ExclusiveMaximumBool)
}

func TestSchemaNumberMultipleOf(t *testing.T) {
	yml := `
type: number
multipleOf: 5
`
	highSchema := getHighSchema(t, yml)

	value := int64(5)
	assert.EqualValues(t, &value, highSchema.MultipleOf)
}

func TestSchemaNumberMinimum(t *testing.T) {
	yml := `
type: number
minimum: 5
`
	highSchema := getHighSchema(t, yml)

	value := int64(5)
	assert.EqualValues(t, &value, highSchema.Minimum)
}

func TestSchemaNumberMinimumZero(t *testing.T) {
	yml := `
type: number
minimum: 0
`
	highSchema := getHighSchema(t, yml)

	value := int64(0)
	assert.EqualValues(t, &value, highSchema.Minimum)
}

func TestSchemaNumberExclusiveMinimum(t *testing.T) {
	yml := `
type: number
exclusiveMinimum: 5
`
	highSchema := getHighSchema(t, yml)

	value := int64(5)
	assert.EqualValues(t, &value, highSchema.ExclusiveMinimum)
}

func TestSchemaNumberMaximum(t *testing.T) {
	yml := `
type: number
maximum: 5
`
	highSchema := getHighSchema(t, yml)

	value := int64(5)
	assert.EqualValues(t, &value, highSchema.Maximum)
}

func TestSchemaNumberMaximumZero(t *testing.T) {
	yml := `
type: number
maximum: 0
`
	highSchema := getHighSchema(t, yml)

	value := int64(0)
	assert.EqualValues(t, &value, highSchema.Maximum)
}

func TestSchemaNumberExclusiveMaximum(t *testing.T) {
	yml := `
type: number
exclusiveMaximum: 5
`
	highSchema := getHighSchema(t, yml)

	value := int64(5)
	assert.EqualValues(t, &value, highSchema.ExclusiveMaximum)
}

func TestSchemaExamples(t *testing.T) {
	yml := `
type: number
examples:
- 5
- 10
`
	highSchema := getHighSchema(t, yml)

	assert.Equal(t, []any{int64(5), int64(10)}, highSchema.Examples)
}

func ExampleNewSchema() {

	// create an example schema object
	// this can be either JSON or YAML.
	yml := `
title: this is a schema
type: object
properties:
  aProperty:
    description: this is an integer property
    type: integer
    format: int64`

	// unmarshal raw bytes
	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	// build out the low-level model
	var lowSchema lowbase.Schema
	_ = low.BuildModel(node.Content[0], &lowSchema)
	_ = lowSchema.Build(node.Content[0], nil)

	// build the high level model
	highSchema := NewSchema(&lowSchema)

	// print out the description of 'aProperty'
	fmt.Print(highSchema.Properties["aProperty"].Schema().Description)
	// Output: this is an integer property

}

func ExampleNewSchemaProxy() {

	// create an example schema object
	// this can be either JSON or YAML.
	yml := `
title: this is a schema
type: object
properties:
  aProperty:
    description: this is an integer property
    type: integer
    format: int64`

	// unmarshal raw bytes
	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	// build out the low-level model
	var lowSchema lowbase.SchemaProxy
	_ = low.BuildModel(node.Content[0], &lowSchema)
	_ = lowSchema.Build(node.Content[0], nil)

	// build the high level schema proxy
	highSchema := NewSchemaProxy(&low.NodeReference[*lowbase.SchemaProxy]{
		Value: &lowSchema,
	})

	// print out the description of 'aProperty'
	fmt.Print(highSchema.Schema().Properties["aProperty"].Schema().Description)
	// Output: this is an integer property

}
