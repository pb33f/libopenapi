// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestDynamicValue_IsA(t *testing.T) {
	dv := &DynamicValue[int, bool]{N: 0, A: 23}
	assert.True(t, dv.IsA())
	assert.False(t, dv.IsB())
}

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

	rend, rendErr := sch1.Render()
	assert.Nil(t, rend)
	assert.Error(t, rendErr)

	g, o := sch1.BuildSchema()
	assert.Nil(t, g)
	assert.Error(t, o)
}

func TestNewSchemaProxyRender(t *testing.T) {
	// check proxy
	yml := `components:
    schemas:
        rice:
            type: string
            description: a rice`

	var idxNode, compNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())

	yml = `properties:
    rice:
     $ref: '#/components/schemas/rice'`

	_ = yaml.Unmarshal([]byte(yml), &compNode)

	sp := new(lowbase.SchemaProxy)
	err := sp.Build(compNode.Content[0], idx)
	assert.NoError(t, err)

	lowproxy := low.NodeReference[*lowbase.SchemaProxy]{
		Value:     sp,
		ValueNode: idxNode.Content[0],
	}

	sch1 := SchemaProxy{schema: &lowproxy}
	assert.NotNil(t, sch1.Schema())
	assert.NoError(t, sch1.GetBuildError())

	g, o := sch1.BuildSchema()
	assert.NotNil(t, g)
	assert.NoError(t, o)

	rend, _ := sch1.Render()
	desired := `properties:
    rice:
        $ref: '#/components/schemas/rice'`
	assert.Equal(t, desired, strings.TrimSpace(string(rend)))

}

func TestNewSchemaProxy_WithObject(t *testing.T) {
	testSpec := `type: object
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
            example: allOfAExp
        allOfB:
            type: string
            description: allOfB description
            example: allOfBExp
oneOf:
    - type: object
      description: a oneof thing
      properties:
        oneOfA:
            type: string
            description: oneOfA description
            example: oneOfAExp
        oneOfB:
            type: string
            description: oneOfB description
            example: oneOfBExp
anyOf:
    - type: object
      description: an anyOf thing
      properties:
        anyOfA:
            type: string
            description: anyOfA description
            example: anyOfAExp
        anyOfB:
            type: string
            description: anyOfB description
            example: anyOfBExp
not:
    type: object
    description: a not thing
    properties:
        notA:
            type: string
            description: notA description
            example: notAExp
        notB:
            type: string
            description: notB description
            example: notBExp
items:
    type: object
    description: an items thing
    properties:
        itemsA:
            type: string
            description: itemsA description
            example: itemsAExp
        itemsB:
            type: string
            description: itemsB description
            example: itemsBExp
prefixItems:
    - type: object
      description: an items thing
      properties:
        itemsA:
            type: string
            description: itemsA description
            example: itemsAExp
        itemsB:
            type: string
            description: itemsB description
            example: itemsBExp
properties:
    somethingA:
        type: number
        description: a number
        example: "2"
        additionalProperties:
            - chicken
            - nugget
            - soup
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
deprecated: true
writeOnly: true
uniqueItems: true
readOnly: true
nullable: true
maxLength: 10
minLength: 1
maxItems: 20
minItems: 10
maxProperties: 30
minProperties: 1
$anchor: anchor`

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

	// check 3.1 properties
	assert.Equal(t, "int", compiled.Contains.Schema().Type[0])
	assert.Equal(t, int64(10), *compiled.MaxContains)
	assert.Equal(t, int64(1), *compiled.MinContains)
	assert.Equal(t, int64(10), *compiled.MaxLength)
	assert.Equal(t, int64(1), *compiled.MinLength)
	assert.Equal(t, int64(20), *compiled.MaxItems)
	assert.Equal(t, int64(10), *compiled.MinItems)
	assert.Equal(t, int64(30), *compiled.MaxProperties)
	assert.Equal(t, int64(1), *compiled.MinProperties)
	assert.Equal(t, "string", compiled.If.Schema().Type[0])
	assert.Equal(t, "integer", compiled.Else.Schema().Type[0])
	assert.Equal(t, "boolean", compiled.Then.Schema().Type[0])
	assert.Equal(t, "string", compiled.PatternProperties["patternOne"].Schema().Type[0])
	assert.Equal(t, "string", compiled.DependentSchemas["schemaOne"].Schema().Type[0])
	assert.Equal(t, "string", compiled.PropertyNames.Schema().Type[0])
	assert.Equal(t, "boolean", compiled.UnevaluatedItems.Schema().Type[0])
	assert.Equal(t, "integer", compiled.UnevaluatedProperties.A.Schema().Type[0])
	assert.True(t, compiled.ReadOnly)
	assert.True(t, compiled.WriteOnly)
	assert.True(t, *compiled.Deprecated)
	assert.True(t, *compiled.Nullable)
	assert.Equal(t, "anchor", compiled.Anchor)

	wentLow := compiled.GoLow()
	assert.Equal(t, 129, wentLow.AdditionalProperties.ValueNode.Line)
	assert.NotNil(t, compiled.GoLowUntyped())

	// now render it out!
	schemaBytes, _ := compiled.Render()
	assert.Len(t, schemaBytes, 3494)
}

func TestSchemaObjectWithAllOfSequenceOrder(t *testing.T) {
	testSpec := test_get_allOf_schema_blob()

	var compNode yaml.Node
	_ = yaml.Unmarshal([]byte(testSpec), &compNode)

	// test data is a map with one node
	mapContent := compNode.Content[0].Content

	_, vn := utils.FindKeyNodeTop(lowbase.AllOfLabel, mapContent)
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

	got := []string{}
	for i := range compiled.AllOf {
		v := compiled.AllOf[i]
		got = append(got, v.Schema().Description)
	}

	assert.Equal(t, want, got)
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
additionalProperties:
  type: string
  description: nice
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

	assert.True(t, compiled.ExclusiveMaximum.A)
	assert.Equal(t, float64(123), compiled.Properties["somethingB"].Schema().ExclusiveMinimum.B)
	assert.Equal(t, float64(334), compiled.Properties["somethingB"].Schema().ExclusiveMaximum.B)
	assert.Len(t, compiled.Properties["somethingB"].Schema().Properties["somethingBProp"].Schema().Type, 2)

	assert.Equal(t, "nice", compiled.AdditionalProperties.(*SchemaProxy).Schema().Description)

	wentLow := compiled.GoLow()
	assert.Equal(t, 97, wentLow.AdditionalProperties.ValueNode.Line)
	assert.Equal(t, 102, wentLow.XML.ValueNode.Line)

	wentLower := compiled.XML.GoLow()
	assert.Equal(t, 102, wentLower.Name.ValueNode.Line)
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
	assert.Equal(t, ref, sp.GoLow().GetReference())

	spNil := NewSchemaProxy(nil)
	assert.Nil(t, spNil.GoLow())
	assert.Nil(t, spNil.GoLowUntyped())

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
	assert.Nil(t, highSchema.Maximum)
	assert.Nil(t, highSchema.ExclusiveMaximum)
}

func TestSchemaNumberMultipleOfInt(t *testing.T) {
	yml := `
type: number
multipleOf: 5
`
	highSchema := getHighSchema(t, yml)

	value := float64(5)
	assert.EqualValues(t, &value, highSchema.MultipleOf)
}

func TestSchemaNumberMultipleOfFloat(t *testing.T) {
	yml := `
type: number
multipleOf: 0.5
`
	highSchema := getHighSchema(t, yml)

	value := 0.5
	assert.EqualValues(t, &value, highSchema.MultipleOf)
}

func TestSchemaNumberMinimumInt(t *testing.T) {
	yml := `
type: number
minimum: 5
`
	highSchema := getHighSchema(t, yml)

	value := float64(5)
	assert.EqualValues(t, &value, highSchema.Minimum)
}

func TestSchemaNumberMinimumFloat(t *testing.T) {
	yml := `
type: number
minimum: 0.5
`
	highSchema := getHighSchema(t, yml)

	value := 0.5
	assert.EqualValues(t, &value, highSchema.Minimum)
}

func TestSchemaNumberMinimumZero(t *testing.T) {
	yml := `
type: number
minimum: 0
`
	highSchema := getHighSchema(t, yml)

	value := float64(0)
	assert.EqualValues(t, &value, highSchema.Minimum)
}

func TestSchemaNumberExclusiveMinimum(t *testing.T) {
	yml := `
type: number
exclusiveMinimum: 5
`
	highSchema := getHighSchema(t, yml)

	value := int64(5)
	assert.EqualValues(t, value, highSchema.ExclusiveMinimum.B)
	assert.True(t, highSchema.ExclusiveMinimum.IsB())
}

func TestSchemaNumberMaximum(t *testing.T) {
	yml := `
type: number
maximum: 5
`
	highSchema := getHighSchema(t, yml)

	value := float64(5)
	assert.EqualValues(t, &value, highSchema.Maximum)
}

func TestSchemaNumberMaximumZero(t *testing.T) {
	yml := `
type: number
maximum: 0
`
	highSchema := getHighSchema(t, yml)

	value := float64(0)
	assert.EqualValues(t, &value, highSchema.Maximum)
}

func TestSchemaNumberExclusiveMaximum(t *testing.T) {
	yml := `
type: number
exclusiveMaximum: 5
`
	highSchema := getHighSchema(t, yml)

	value := int64(5)
	assert.EqualValues(t, value, highSchema.ExclusiveMaximum.B)
	assert.True(t, highSchema.ExclusiveMaximum.IsB())
}

func TestSchema_Items_Boolean(t *testing.T) {
	yml := `
type: number
items: true
`
	highSchema := getHighSchema(t, yml)

	assert.True(t, highSchema.Items.B)
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
properties:
  somethingBee:
    type: number
  somethingThree:
    type: number
  somethingTwo:
    type: number
  somethingOne:
    type: number
`
}

func TestNewSchemaProxy_RenderSchema(t *testing.T) {
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
            example: allOfAExp
        allOfB:
            type: string
            description: allOfB description
            example: allOfBExp
`

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

	// now render it out, it should be identical.
	schemaBytes, _ := compiled.Render()
	assert.Equal(t, testSpec, string(schemaBytes))

}

func TestNewSchemaProxy_RenderSchemaWithMultipleObjectTypes(t *testing.T) {
	testSpec := `type: object
description: something object
oneOf:
    - type: object
      description: a oneof thing
      properties:
        oneOfA:
            type: string
            example: oneOfAExp
anyOf:
    - type: object
      description: an anyOf thing
      properties:
        anyOfA:
            type: string
            example: anyOfAExp
not:
    type: object
    description: a not thing
    properties:
        notA:
            type: string
            example: notAExp
items:
    type: object
    description: an items thing
    properties:
        itemsA:
            type: string
            description: itemsA description
            example: itemsAExp
        itemsB:
            type: string
            description: itemsB description
            example: itemsBExp
`

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

	// now render it out, it should be identical.
	schemaBytes, _ := compiled.Render()
	assert.Equal(t, testSpec, string(schemaBytes))
}

func TestNewSchemaProxy_RenderSchemaEnsurePropertyOrdering(t *testing.T) {
	testSpec := `properties:
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
        example: "2"
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
                    x-pizza: love
        additionalProperties:
            why: yes
            thatIs: true
additionalProperties: true
xml:
    name: XML Thing`

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

	// now render it out, it should be identical.
	schemaBytes, _ := compiled.Render()
	assert.Equal(t, testSpec, strings.TrimSpace(string(schemaBytes)))
}

func TestNewSchemaProxy_RenderSchemaCheckDiscriminatorMappingOrder(t *testing.T) {
	testSpec := `discriminator:
    mapping:
        log: cat
        pizza: party
        chicken: nuggets
        warm: soup
        cold: heart`

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

	// now render it out, it should be identical.
	schemaBytes, _ := compiled.Render()
	assert.Equal(t, testSpec, strings.TrimSpace(string(schemaBytes)))
}

func TestNewSchemaProxy_RenderSchemaCheckAdditionalPropertiesSlice(t *testing.T) {
	testSpec := `additionalProperties:
    - one
    - two
    - miss a few
    - ninety nine
    - hundred`

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

	// now render it out, it should be identical.
	schemaBytes, _ := compiled.Render()
	assert.Len(t, schemaBytes, 91)
}

func TestNewSchemaProxy_RenderSchemaCheckAdditionalPropertiesSliceMap(t *testing.T) {
	testSpec := `additionalProperties:
    - nice: cake
    - yummy: beer
    - hot: coffee`

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

	// now render it out, it should be identical.
	schemaBytes, _ := compiled.Render()
	assert.Len(t, schemaBytes, 75)
}

func TestNewSchemaProxy_CheckDefaultBooleanFalse(t *testing.T) {
	testSpec := `default: false`

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

	// now render it out, it should be identical.
	schemaBytes, _ := compiled.Render()
	assert.Equal(t, testSpec, strings.TrimSpace(string(schemaBytes)))
}

func TestNewSchemaProxy_RenderAdditionalPropertiesFalse(t *testing.T) {
	testSpec := `additionalProperties: false`

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

	// now render it out, it should be identical.
	schemaBytes, _ := compiled.Render()
	assert.Equal(t, testSpec, strings.TrimSpace(string(schemaBytes)))
}

func TestNewSchemaProxy_RenderMultiplePoly(t *testing.T) {
	idxYaml := `openapi: 3.1.0
components:
    schemas:
        balance_transaction:
          description: A balance transaction`

	testSpec := `properties:
    bigBank:
        type: object
        properties:
            failure_balance_transaction:
                anyOf:
                    - maxLength: 5000
                      type: string
                    - $ref: '#/components/schemas/balance_transaction'
                x-expansionResources:
                    oneOf:
                        - $ref: '#/components/schemas/balance_transaction'`

	var compNode, idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(testSpec), &compNode)
	_ = yaml.Unmarshal([]byte(idxYaml), &idxNode)

	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())

	sp := new(lowbase.SchemaProxy)

	err := sp.Build(compNode.Content[0], idx)
	assert.NoError(t, err)

	lowproxy := low.NodeReference[*lowbase.SchemaProxy]{
		Value:     sp,
		ValueNode: idxNode.Content[0],
	}

	sch1 := SchemaProxy{schema: &lowproxy}
	compiled := sch1.Schema()

	// now render it out, it should be identical.
	schemaBytes, _ := compiled.Render()
	assert.Equal(t, testSpec, strings.TrimSpace(string(schemaBytes)))
}

func TestNewSchemaProxy_RenderInline(t *testing.T) {
	idxYaml := `openapi: 3.1.0
components:
    schemas:
        balance_transaction:
          description: A balance transaction
        red_burgers:
          type: object
          properties:
            name:
              type: string
            price:
              type: number
          anyOf:
            - $ref: '#/components/schemas/balance_transaction'`

	testSpec := `properties:
    bigBank:
        type: object
        properties:
            failure_balance_transaction:
                allOf:
                    - $ref: '#/components/schemas/red_burgers'
                anyOf:
                    - maxLength: 5000
                      type: string
                    - $ref: '#/components/schemas/balance_transaction'`

	var compNode, idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(testSpec), &compNode)
	_ = yaml.Unmarshal([]byte(idxYaml), &idxNode)

	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())

	sp := new(lowbase.SchemaProxy)

	err := sp.Build(compNode.Content[0], idx)
	assert.NoError(t, err)

	lowproxy := low.NodeReference[*lowbase.SchemaProxy]{
		Value:     sp,
		ValueNode: idxNode.Content[0],
	}

	sch1 := SchemaProxy{schema: &lowproxy}
	compiled := sch1.Schema()

	// now render it out, it should be identical.
	schemaBytes, _ := compiled.RenderInline()
	assert.Len(t, schemaBytes, 585)
}

func TestUnevaluatedPropertiesBoolean_True(t *testing.T) {
	yml := `
type: number
unevaluatedProperties: true
`
	highSchema := getHighSchema(t, yml)

	value := true
	assert.EqualValues(t, &value, highSchema.UnevaluatedProperties.B)
}

func TestUnevaluatedPropertiesBoolean_False(t *testing.T) {
	yml := `
type: number
unevaluatedProperties: false
`
	highSchema := getHighSchema(t, yml)

	value := false
	assert.EqualValues(t, &value, highSchema.UnevaluatedProperties.B)
}
