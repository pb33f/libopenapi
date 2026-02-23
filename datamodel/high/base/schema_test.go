// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel"

	"github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
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
	err := sp.Build(context.Background(), nil, compNode.Content[0], idx)
	assert.NoError(t, err)

	lowproxy := low.NodeReference[*lowbase.SchemaProxy]{
		Value:     sp,
		ValueNode: idxNode.Content[0],
	}

	sch1 := NewSchemaProxy(&lowproxy)
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
	err := sp.Build(context.Background(), nil, compNode.Content[0], idx)
	assert.NoError(t, err)

	lowproxy := low.NodeReference[*lowbase.SchemaProxy]{
		Value:     sp,
		ValueNode: idxNode.Content[0],
	}

	sch1 := NewSchemaProxy(&lowproxy)
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
        additionalProperties: false
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
            type: string
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
$anchor: anchor
$dynamicAnchor: dynamicAnchorValue
$dynamicRef: "#dynamicRefTarget"
$schema: https://example.com/custom-json-schema-dialect`

	var compNode yaml.Node
	_ = yaml.Unmarshal([]byte(testSpec), &compNode)

	sp := new(lowbase.SchemaProxy)
	err := sp.Build(context.Background(), nil, compNode.Content[0], nil)
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
	assert.Equal(t, "string", compiled.PatternProperties.GetOrZero("patternOne").Schema().Type[0])
	assert.Equal(t, "string", compiled.DependentSchemas.GetOrZero("schemaOne").Schema().Type[0])
	assert.Equal(t, "string", compiled.PropertyNames.Schema().Type[0])
	assert.Equal(t, "boolean", compiled.UnevaluatedItems.Schema().Type[0])
	assert.Equal(t, "integer", compiled.UnevaluatedProperties.A.Schema().Type[0])
	assert.True(t, *compiled.ReadOnly)
	assert.True(t, *compiled.WriteOnly)
	assert.True(t, *compiled.Deprecated)
	assert.True(t, *compiled.Nullable)
	assert.Equal(t, "anchor", compiled.Anchor)
	assert.Equal(t, "dynamicAnchorValue", compiled.DynamicAnchor)
	assert.Equal(t, "#dynamicRefTarget", compiled.DynamicRef)
	assert.Equal(t, "https://example.com/custom-json-schema-dialect", compiled.SchemaTypeRef)

	wentLow := compiled.GoLow()
	assert.Equal(t, 125, wentLow.AdditionalProperties.ValueNode.Line)
	assert.NotNil(t, compiled.GoLowUntyped())

	// now render it out!
	schemaBytes, _ := compiled.Render()
	assert.Len(t, schemaBytes, 3541)
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
	err := sp.Build(context.Background(), nil, compNode.Content[0], nil)
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
	err := sp.Build(context.Background(), nil, compNode.Content[0], nil)
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
	assert.Equal(t, float64(123), compiled.Properties.GetOrZero("somethingB").Schema().ExclusiveMinimum.B)
	assert.Equal(t, float64(334), compiled.Properties.GetOrZero("somethingB").Schema().ExclusiveMaximum.B)
	assert.Len(t, compiled.Properties.GetOrZero("somethingB").Schema().Properties.GetOrZero("somethingBProp").Schema().Type, 2)

	assert.Equal(t, "nice", compiled.AdditionalProperties.A.Schema().Description)

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
	err := lowProxy.Build(context.Background(), nil, node.Content[0], idx)
	assert.NoError(t, err)

	lowRef := low.NodeReference[*lowbase.SchemaProxy]{
		Value: lowProxy,
	}

	sp := NewSchemaProxy(&lowRef)
	assert.Equal(t, lowProxy, sp.GoLow())
	assert.Equal(t, ref, sp.GoLow().GetReference())
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
	assert.NoError(t, lowSchema.Build(context.Background(), node.Content[0], nil))

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

	examples := []any{}
	for _, ex := range highSchema.Examples {
		var v int64
		assert.NoError(t, ex.Decode(&v))
		examples = append(examples, v)
	}

	assert.Equal(t, []any{int64(5), int64(10)}, examples)
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
	_ = lowSchema.Build(context.Background(), node.Content[0], nil)

	// build the high level model
	highSchema := NewSchema(&lowSchema)

	// print out the description of 'aProperty'
	fmt.Print(highSchema.Properties.GetOrZero("aProperty").Schema().Description)
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
	_ = lowSchema.Build(context.Background(), nil, node.Content[0], nil)

	// build the high level schema proxy
	highSchema := NewSchemaProxy(&low.NodeReference[*lowbase.SchemaProxy]{
		Value: &lowSchema,
	})

	// print out the description of 'aProperty'
	fmt.Print(highSchema.Schema().Properties.GetOrZero("aProperty").Schema().Description)
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
	err := sp.Build(context.Background(), nil, compNode.Content[0], nil)
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

func TestNewSchemaProxy_RenderSchema_JSON(t *testing.T) {
	testSpec := `type: object
description: something object
`

	var compNode yaml.Node
	_ = yaml.Unmarshal([]byte(testSpec), &compNode)

	sp := new(lowbase.SchemaProxy)

	// add a config
	idxConfig := index.CreateOpenAPIIndexConfig()
	idxConfig.SpecInfo = &datamodel.SpecInfo{
		VersionNumeric: 3,
	}
	idx := index.NewSpecIndexWithConfig(nil, idxConfig)

	err := sp.Build(context.Background(), nil, compNode.Content[0], idx)
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

	// now render it out, it should be identical, but in JSON
	schemaBytes, _ := compiled.MarshalJSON()
	assert.Equal(t, `{"description":"something object","type":"object"}`, string(schemaBytes))
}

func TestNewSchemaProxy_RenderSchema_JSONInline(t *testing.T) {
	testSpec := `type: object
description: something object
`

	var compNode yaml.Node
	_ = yaml.Unmarshal([]byte(testSpec), &compNode)

	sp := new(lowbase.SchemaProxy)

	// add a config
	idxConfig := index.CreateOpenAPIIndexConfig()
	idxConfig.SpecInfo = &datamodel.SpecInfo{
		VersionNumeric: 3,
	}
	idx := index.NewSpecIndexWithConfig(nil, idxConfig)

	err := sp.Build(context.Background(), nil, compNode.Content[0], idx)
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

	// now render it out, it should be identical, but in JSON
	schemaBytes, _ := compiled.MarshalJSONInline()
	assert.Equal(t, `{"description":"something object","type":"object"}`, string(schemaBytes))
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
	err := sp.Build(context.Background(), nil, compNode.Content[0], nil)
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
            type: string
additionalProperties: true
xml:
    name: XML Thing`

	var compNode yaml.Node
	_ = yaml.Unmarshal([]byte(testSpec), &compNode)

	sp := new(lowbase.SchemaProxy)
	err := sp.Build(context.Background(), nil, compNode.Content[0], nil)
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
	err := sp.Build(context.Background(), nil, compNode.Content[0], nil)
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

func TestNewSchemaProxy_CheckDefaultBooleanFalse(t *testing.T) {
	testSpec := `default: false`

	var compNode yaml.Node
	_ = yaml.Unmarshal([]byte(testSpec), &compNode)

	sp := new(lowbase.SchemaProxy)
	err := sp.Build(context.Background(), nil, compNode.Content[0], nil)
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
	err := sp.Build(context.Background(), nil, compNode.Content[0], nil)
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

	err := sp.Build(context.Background(), nil, compNode.Content[0], idx)
	assert.NoError(t, err)

	lowproxy := low.NodeReference[*lowbase.SchemaProxy]{
		Value:     sp,
		ValueNode: idxNode.Content[0],
	}

	sch1 := NewSchemaProxy(&lowproxy)
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

	err := sp.Build(context.Background(), nil, compNode.Content[0], idx)
	assert.NoError(t, err)

	lowproxy := low.NodeReference[*lowbase.SchemaProxy]{
		Value:     sp,
		ValueNode: idxNode.Content[0],
	}

	sch1 := NewSchemaProxy(&lowproxy)
	compiled := sch1.Schema()

	// now render it out, it should be identical.
	schemaBytes, _ := compiled.RenderInline()
	assert.Equal(t, "properties:\n    bigBank:\n        type: object\n        properties:\n            failure_balance_transaction:\n                allOf:\n                    - type: object\n                      properties:\n                        name:\n                            type: string\n                        price:\n                            type: number\n                      anyOf:\n                        - description: A balance transaction\n                anyOf:\n                    - maxLength: 5000\n                      type: string\n                    - description: A balance transaction\n", string(schemaBytes))
}

func TestSchema_RenderInline_MapEncodedNestedProperties_NoCircularDetection(t *testing.T) {
	// Ensure inline rendering doesn't falsely detect circular refs when schema
	// nodes are built from yaml.Node.Encode (no line/column metadata).
	schemaMap := map[string]any{
		"type": "array",
		"contains": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"const": "X-Required-Version",
				},
				"in": map[string]any{
					"const": "header",
				},
			},
			"required": []any{"name", "in"},
		},
	}

	var node yaml.Node
	require.NoError(t, node.Encode(schemaMap))

	lowSchema := new(lowbase.Schema)
	require.NoError(t, lowSchema.Build(context.Background(), &node, nil))

	highSchema := NewSchema(lowSchema)
	_, err := highSchema.RenderInline()
	assert.NoError(t, err)
}

func TestUnevaluatedPropertiesBoolean_True(t *testing.T) {
	yml := `
type: number
unevaluatedProperties: true
`
	highSchema := getHighSchema(t, yml)

	assert.True(t, highSchema.UnevaluatedProperties.B)
}

func TestUnevaluatedPropertiesBoolean_False(t *testing.T) {
	yml := `
type: number
unevaluatedProperties: false
`
	highSchema := getHighSchema(t, yml)

	assert.False(t, highSchema.UnevaluatedProperties.B)
}

func TestUnevaluatedPropertiesBoolean_Unset(t *testing.T) {
	yml := `
type: number
`
	highSchema := getHighSchema(t, yml)

	assert.Nil(t, highSchema.UnevaluatedProperties)
}

func TestAdditionalProperties(t *testing.T) {
	testSpec := `type: object
properties:
  additionalPropertiesSimpleSchema:
    type: object
    additionalProperties:
      type: string
  additionalPropertiesBool:
    type: object
    additionalProperties: true
  additionalPropertiesAnyOf:
    type: object
    additionalProperties:
      anyOf:
        - type: string
        - type: array
          items:
            type: string
`

	var compNode yaml.Node
	_ = yaml.Unmarshal([]byte(testSpec), &compNode)

	sp := new(lowbase.SchemaProxy)
	err := sp.Build(context.Background(), nil, compNode.Content[0], nil)
	assert.NoError(t, err)

	lowproxy := low.NodeReference[*lowbase.SchemaProxy]{
		Value:     sp,
		ValueNode: compNode.Content[0],
	}

	schemaProxy := NewSchemaProxy(&lowproxy)
	compiled := schemaProxy.Schema()

	assert.Equal(t, []string{"string"}, compiled.Properties.GetOrZero("additionalPropertiesSimpleSchema").Schema().AdditionalProperties.A.Schema().Type)
	assert.Equal(t, true, compiled.Properties.GetOrZero("additionalPropertiesBool").Schema().AdditionalProperties.B)
	assert.Equal(t, []string{"string"}, compiled.Properties.GetOrZero("additionalPropertiesAnyOf").Schema().AdditionalProperties.A.Schema().AnyOf[0].Schema().Type)
}

func TestSchema_RenderProxyWithConfig_3(t *testing.T) {
	testSpec := `exclusiveMinimum: true`

	var compNode yaml.Node
	_ = yaml.Unmarshal([]byte(testSpec), &compNode)

	sp := new(lowbase.SchemaProxy)
	err := sp.Build(context.Background(), nil, compNode.Content[0], nil)
	assert.NoError(t, err)

	config := index.CreateOpenAPIIndexConfig()
	config.SpecInfo = &datamodel.SpecInfo{
		VersionNumeric: 3.0,
	}
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

func TestSchema_RenderProxyWithConfig_Corrected_31(t *testing.T) {
	testSpec := `exclusiveMinimum: true`
	testSpecCorrect := `exclusiveMinimum: 0`

	var compNode yaml.Node
	_ = yaml.Unmarshal([]byte(testSpec), &compNode)

	sp := new(lowbase.SchemaProxy)
	config := index.CreateOpenAPIIndexConfig()
	config.SpecInfo = &datamodel.SpecInfo{
		VersionNumeric: 3.1,
	}
	idx := index.NewSpecIndexWithConfig(compNode.Content[0], config)

	err := sp.Build(context.Background(), nil, compNode.Content[0], idx)
	assert.NoError(t, err)

	lowproxy := low.NodeReference[*lowbase.SchemaProxy]{
		Value:     sp,
		ValueNode: compNode.Content[0],
	}

	schemaProxy := NewSchemaProxy(&lowproxy)
	compiled := schemaProxy.Schema()

	// now render it out, it should be identical.
	schemaBytes, _ := compiled.Render()
	assert.Equal(t, testSpecCorrect, strings.TrimSpace(string(schemaBytes)))

	schemaBytes, _ = compiled.RenderInline()
	assert.Equal(t, testSpecCorrect, strings.TrimSpace(string(schemaBytes)))
}

func TestSchema_RenderProxyWithConfig_Corrected_3(t *testing.T) {
	testSpec := `exclusiveMinimum: 0`
	testSpecCorrect := `exclusiveMinimum: false`

	var compNode yaml.Node
	_ = yaml.Unmarshal([]byte(testSpec), &compNode)

	sp := new(lowbase.SchemaProxy)
	config := index.CreateOpenAPIIndexConfig()
	config.SpecInfo = &datamodel.SpecInfo{
		VersionNumeric: 3.0,
	}
	idx := index.NewSpecIndexWithConfig(compNode.Content[0], config)

	err := sp.Build(context.Background(), nil, compNode.Content[0], idx)
	assert.NoError(t, err)

	lowproxy := low.NodeReference[*lowbase.SchemaProxy]{
		Value:     sp,
		ValueNode: compNode.Content[0],
	}

	schemaProxy := NewSchemaProxy(&lowproxy)
	compiled := schemaProxy.Schema()

	// now render it out, it should be identical.
	schemaBytes, _ := compiled.Render()
	assert.Equal(t, testSpecCorrect, strings.TrimSpace(string(schemaBytes)))

	schemaBytes, _ = compiled.RenderInline()
	assert.Equal(t, testSpecCorrect, strings.TrimSpace(string(schemaBytes)))
}

func TestNewSchema_DependentRequired_Success(t *testing.T) {
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

	var lowSchema lowbase.Schema
	_ = lowSchema.Build(context.Background(), idxNode.Content[0], idx)

	// Create high-level schema
	schema := NewSchema(&lowSchema)

	// Check that DependentRequired was mapped correctly
	assert.NotNil(t, schema.DependentRequired)
	assert.Equal(t, 2, schema.DependentRequired.Len())

	// Check billingAddress dependency
	billingReq := schema.DependentRequired.GetOrZero("billingAddress")
	assert.Equal(t, []string{"street_address", "locality", "region"}, billingReq)

	// Check creditCard dependency
	creditReq := schema.DependentRequired.GetOrZero("creditCard")
	assert.Equal(t, []string{"billing_address"}, creditReq)
}

func TestNewSchema_DependentRequired_Empty(t *testing.T) {
	yml := `type: object
description: something object
properties:
  name:
    type: string`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var lowSchema lowbase.Schema
	_ = lowSchema.Build(context.Background(), idxNode.Content[0], idx)

	// Create high-level schema
	schema := NewSchema(&lowSchema)

	// Check that DependentRequired is nil when not present
	assert.Nil(t, schema.DependentRequired)
}

func TestNewSchema_DependentRequired_EmptyArray(t *testing.T) {
	yml := `type: object
dependentRequired:
  billingAddress: []`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var lowSchema lowbase.Schema
	_ = lowSchema.Build(context.Background(), idxNode.Content[0], idx)

	// Create high-level schema
	schema := NewSchema(&lowSchema)

	// Check that DependentRequired has empty array (nil is equivalent to empty slice in Go)
	assert.NotNil(t, schema.DependentRequired)
	billingReq := schema.DependentRequired.GetOrZero("billingAddress")
	assert.Empty(t, billingReq) // Use Empty() which handles both nil and empty slices
}

func TestNewSchema_DependentRequired_SingleProperty(t *testing.T) {
	yml := `type: object
dependentRequired:
  creditCard:
    - billing_address`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var lowSchema lowbase.Schema
	_ = lowSchema.Build(context.Background(), idxNode.Content[0], idx)

	// Create high-level schema
	schema := NewSchema(&lowSchema)

	// Check that DependentRequired is mapped correctly for single property
	assert.NotNil(t, schema.DependentRequired)
	assert.Equal(t, 1, schema.DependentRequired.Len())

	creditReq := schema.DependentRequired.GetOrZero("creditCard")
	assert.Equal(t, []string{"billing_address"}, creditReq)
}

func TestNewSchema_DependentRequired_MultipleProperties_SingleDependency(t *testing.T) {
	yml := `type: object
dependentRequired:
  firstName:
    - lastName
  lastName:
    - firstName
  email:
    - username`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var lowSchema lowbase.Schema
	_ = lowSchema.Build(context.Background(), idxNode.Content[0], idx)

	// Create high-level schema
	schema := NewSchema(&lowSchema)

	// Check that all DependentRequired mappings are correct
	assert.NotNil(t, schema.DependentRequired)
	assert.Equal(t, 3, schema.DependentRequired.Len())

	assert.Equal(t, []string{"lastName"}, schema.DependentRequired.GetOrZero("firstName"))
	assert.Equal(t, []string{"firstName"}, schema.DependentRequired.GetOrZero("lastName"))
	assert.Equal(t, []string{"username"}, schema.DependentRequired.GetOrZero("email"))
}

// Tests for Schema.MarshalYAMLInline discriminator reference preservation (lines 539-549 in schema.go)

func TestSchema_MarshalYAMLInline_DiscriminatorPreservesOneOfRefs(t *testing.T) {
	// Test that when a schema has a discriminator, oneOf refs are preserved (not inlined)
	// This covers lines 539-544 in schema.go

	idxYaml := `openapi: 3.1.0
components:
  schemas:
    Cat:
      type: object
      properties:
        type:
          type: string
        meow:
          type: boolean
    Dog:
      type: object
      properties:
        type:
          type: string
        bark:
          type: boolean`

	testSpec := `discriminator:
  propertyName: type
  mapping:
    cat: '#/components/schemas/Cat'
    dog: '#/components/schemas/Dog'
oneOf:
  - $ref: '#/components/schemas/Cat'
  - $ref: '#/components/schemas/Dog'`

	var compNode, idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(testSpec), &compNode)
	_ = yaml.Unmarshal([]byte(idxYaml), &idxNode)

	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())

	sp := new(lowbase.SchemaProxy)
	err := sp.Build(context.Background(), nil, compNode.Content[0], idx)
	assert.NoError(t, err)

	lowproxy := low.NodeReference[*lowbase.SchemaProxy]{
		Value:     sp,
		ValueNode: compNode.Content[0],
	}

	schemaProxy := NewSchemaProxy(&lowproxy)
	compiled := schemaProxy.Schema()

	// Call MarshalYAMLInline - this should set preserveReference on oneOf refs
	result, err := compiled.MarshalYAMLInline()
	assert.NoError(t, err)

	// Marshal to YAML to check output
	yamlBytes, _ := yaml.Marshal(result)
	output := string(yamlBytes)

	// The oneOf refs should be preserved as $ref, not inlined
	assert.Contains(t, output, "$ref:")
	assert.Contains(t, output, "#/components/schemas/Cat")
	assert.Contains(t, output, "#/components/schemas/Dog")
	// Should NOT contain the inlined properties
	assert.NotContains(t, output, "meow:")
	assert.NotContains(t, output, "bark:")
}

func TestSchema_MarshalYAMLInline_DiscriminatorPreservesAnyOfRefs(t *testing.T) {
	// Test that when a schema has a discriminator, anyOf refs are preserved (not inlined)
	// This covers lines 545-549 in schema.go

	idxYaml := `openapi: 3.1.0
components:
  schemas:
    Cat:
      type: object
      properties:
        type:
          type: string
        meow:
          type: boolean
    Dog:
      type: object
      properties:
        type:
          type: string
        bark:
          type: boolean`

	testSpec := `discriminator:
  propertyName: type
  mapping:
    cat: '#/components/schemas/Cat'
    dog: '#/components/schemas/Dog'
anyOf:
  - $ref: '#/components/schemas/Cat'
  - $ref: '#/components/schemas/Dog'`

	var compNode, idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(testSpec), &compNode)
	_ = yaml.Unmarshal([]byte(idxYaml), &idxNode)

	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())

	sp := new(lowbase.SchemaProxy)
	err := sp.Build(context.Background(), nil, compNode.Content[0], idx)
	assert.NoError(t, err)

	lowproxy := low.NodeReference[*lowbase.SchemaProxy]{
		Value:     sp,
		ValueNode: compNode.Content[0],
	}

	schemaProxy := NewSchemaProxy(&lowproxy)
	compiled := schemaProxy.Schema()

	// Call MarshalYAMLInline - this should set preserveReference on anyOf refs
	result, err := compiled.MarshalYAMLInline()
	assert.NoError(t, err)

	// Marshal to YAML to check output
	yamlBytes, _ := yaml.Marshal(result)
	output := string(yamlBytes)

	// The anyOf refs should be preserved as $ref, not inlined
	assert.Contains(t, output, "$ref:")
	assert.Contains(t, output, "#/components/schemas/Cat")
	assert.Contains(t, output, "#/components/schemas/Dog")
	// Should NOT contain the inlined properties
	assert.NotContains(t, output, "meow:")
	assert.NotContains(t, output, "bark:")
}

func TestSchema_MarshalYAMLInline_DiscriminatorMixedOneOf(t *testing.T) {
	// Test that when a schema has a discriminator with mixed oneOf (refs and inline),
	// only the refs are preserved, inline schemas remain inline

	idxYaml := `openapi: 3.1.0
components:
  schemas:
    Cat:
      type: object
      properties:
        type:
          type: string
        meow:
          type: boolean`

	testSpec := `discriminator:
  propertyName: type
  mapping:
    cat: '#/components/schemas/Cat'
oneOf:
  - $ref: '#/components/schemas/Cat'
  - type: object
    properties:
      type:
        type: string
      inline_prop:
        type: string`

	var compNode, idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(testSpec), &compNode)
	_ = yaml.Unmarshal([]byte(idxYaml), &idxNode)

	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())

	sp := new(lowbase.SchemaProxy)
	err := sp.Build(context.Background(), nil, compNode.Content[0], idx)
	assert.NoError(t, err)

	lowproxy := low.NodeReference[*lowbase.SchemaProxy]{
		Value:     sp,
		ValueNode: compNode.Content[0],
	}

	schemaProxy := NewSchemaProxy(&lowproxy)
	compiled := schemaProxy.Schema()

	// Call MarshalYAMLInline
	result, err := compiled.MarshalYAMLInline()
	assert.NoError(t, err)

	// Marshal to YAML to check output
	yamlBytes, _ := yaml.Marshal(result)
	output := string(yamlBytes)

	// The ref should be preserved
	assert.Contains(t, output, "$ref:")
	assert.Contains(t, output, "#/components/schemas/Cat")
	// The inline schema properties should still be present
	assert.Contains(t, output, "inline_prop:")
}

func TestSchema_MarshalYAMLInline_NoDiscriminatorInlinesRefs(t *testing.T) {
	// Test that without a discriminator, oneOf refs ARE inlined
	// This is the control case to verify the discriminator logic makes a difference

	idxYaml := `openapi: 3.1.0
components:
  schemas:
    Cat:
      type: object
      properties:
        meow:
          type: boolean`

	testSpec := `oneOf:
  - $ref: '#/components/schemas/Cat'`

	var compNode, idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(testSpec), &compNode)
	_ = yaml.Unmarshal([]byte(idxYaml), &idxNode)

	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())

	sp := new(lowbase.SchemaProxy)
	err := sp.Build(context.Background(), nil, compNode.Content[0], idx)
	assert.NoError(t, err)

	lowproxy := low.NodeReference[*lowbase.SchemaProxy]{
		Value:     sp,
		ValueNode: compNode.Content[0],
	}

	schemaProxy := NewSchemaProxy(&lowproxy)
	compiled := schemaProxy.Schema()

	// Call MarshalYAMLInline - without discriminator, refs should be inlined
	result, err := compiled.MarshalYAMLInline()
	assert.NoError(t, err)

	// Marshal to YAML to check output
	yamlBytes, _ := yaml.Marshal(result)
	output := string(yamlBytes)

	// Without discriminator, the ref should be inlined - we should see the property
	assert.Contains(t, output, "meow:")
}

func TestSchema_MarshalYAMLInline_DiscriminatorWithNonRefSchemaProxy(t *testing.T) {
	// Test that non-reference SchemaProxy entries are handled correctly
	// (IsReference() returns false, so SetPreserveReference is not called)
	//
	// This tests that the `sp.IsReference()` check in lines 541 and 546 works correctly

	idxYaml := `openapi: 3.1.0
components:
  schemas:
    Dog:
      type: object
      properties:
        bark:
          type: boolean`

	// Schema with discriminator, but oneOf contains an inline schema (not a ref)
	testSpec := `discriminator:
  propertyName: type
oneOf:
  - type: object
    properties:
      type:
        type: string
      meow:
        type: boolean`

	var compNode, idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(testSpec), &compNode)
	_ = yaml.Unmarshal([]byte(idxYaml), &idxNode)

	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())

	sp := new(lowbase.SchemaProxy)
	err := sp.Build(context.Background(), nil, compNode.Content[0], idx)
	assert.NoError(t, err)

	lowproxy := low.NodeReference[*lowbase.SchemaProxy]{
		Value:     sp,
		ValueNode: compNode.Content[0],
	}

	schemaProxy := NewSchemaProxy(&lowproxy)
	compiled := schemaProxy.Schema()

	// The oneOf entry is an inline schema, not a reference
	assert.Len(t, compiled.OneOf, 1)
	assert.False(t, compiled.OneOf[0].IsReference())

	// Call MarshalYAMLInline - inline schemas should remain inline
	result, err := compiled.MarshalYAMLInline()
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Marshal to YAML to check output
	yamlBytes, _ := yaml.Marshal(result)
	output := string(yamlBytes)

	// Should contain the inline schema properties, not a $ref
	assert.Contains(t, output, "meow:")
	assert.Contains(t, output, "type:")
}

func TestSchema_RenderInlineWithContext_Error(t *testing.T) {
	// Test the error path in RenderInlineWithContext (line 506)
	// Create a schema with a circular reference that will trigger an error

	idxYaml := `components:
  schemas:
    Circular:
      type: object
      properties:
        self:
          $ref: '#/components/schemas/Circular'`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(idxYaml), &idxNode)

	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())

	// Build the circular schema
	schemas := idxNode.Content[0].Content[1].Content[1] // components -> schemas -> Circular
	sp := new(lowbase.SchemaProxy)
	err := sp.Build(context.Background(), nil, schemas.Content[1], idx)
	assert.NoError(t, err)

	lowproxy := low.NodeReference[*lowbase.SchemaProxy]{
		Value:     sp,
		ValueNode: schemas.Content[1],
	}

	schemaProxy := NewSchemaProxy(&lowproxy)
	compiled := schemaProxy.Schema()

	// Create a context and pre-mark the schema's render key to simulate a cycle
	ctx := NewInlineRenderContext()

	// Get the render key for the self-referencing property's schema proxy
	if compiled.Properties != nil {
		selfProp := compiled.Properties.GetOrZero("self")
		if selfProp != nil {
			// Pre-mark this key as rendering to force a cycle error
			renderKey := selfProp.getInlineRenderKey()
			if renderKey != "" {
				ctx.StartRendering(renderKey)
			}
		}
	}

	// RenderInlineWithContext should return an error due to the pre-marked cycle
	result, err := compiled.RenderInlineWithContext(ctx)

	// The error path should be triggered
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "circular reference")
}

// Tests for RenderingModeValidation - discriminator refs should be inlined in validation mode

func TestSchema_MarshalYAMLInlineWithContext_ValidationMode_InlinesDiscriminatorOneOfRefs(t *testing.T) {
	// Test that in validation mode, discriminator oneOf refs are inlined (not preserved)
	// This is the opposite of bundle mode behavior

	idxYaml := `openapi: 3.1.0
components:
  schemas:
    Cat:
      type: object
      properties:
        type:
          type: string
        meow:
          type: boolean
    Dog:
      type: object
      properties:
        type:
          type: string
        bark:
          type: boolean`

	testSpec := `discriminator:
  propertyName: type
  mapping:
    cat: '#/components/schemas/Cat'
    dog: '#/components/schemas/Dog'
oneOf:
  - $ref: '#/components/schemas/Cat'
  - $ref: '#/components/schemas/Dog'`

	var compNode, idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(testSpec), &compNode)
	_ = yaml.Unmarshal([]byte(idxYaml), &idxNode)

	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())

	sp := new(lowbase.SchemaProxy)
	err := sp.Build(context.Background(), nil, compNode.Content[0], idx)
	assert.NoError(t, err)

	lowproxy := low.NodeReference[*lowbase.SchemaProxy]{
		Value:     sp,
		ValueNode: compNode.Content[0],
	}

	schemaProxy := NewSchemaProxy(&lowproxy)
	compiled := schemaProxy.Schema()

	// Use validation mode context - refs should be inlined
	ctx := NewInlineRenderContextForValidation()
	result, err := compiled.MarshalYAMLInlineWithContext(ctx)
	assert.NoError(t, err)

	// Marshal to YAML to check output
	yamlBytes, _ := yaml.Marshal(result)
	output := string(yamlBytes)

	// In validation mode, the oneOf refs should be INLINED, not preserved as $ref
	// Should contain the inlined properties
	assert.Contains(t, output, "meow:")
	assert.Contains(t, output, "bark:")
}

func TestSchema_MarshalYAMLInlineWithContext_ValidationMode_InlinesDiscriminatorAnyOfRefs(t *testing.T) {
	// Test that in validation mode, discriminator anyOf refs are inlined (not preserved)

	idxYaml := `openapi: 3.1.0
components:
  schemas:
    Cat:
      type: object
      properties:
        type:
          type: string
        meow:
          type: boolean
    Dog:
      type: object
      properties:
        type:
          type: string
        bark:
          type: boolean`

	testSpec := `discriminator:
  propertyName: type
anyOf:
  - $ref: '#/components/schemas/Cat'
  - $ref: '#/components/schemas/Dog'`

	var compNode, idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(testSpec), &compNode)
	_ = yaml.Unmarshal([]byte(idxYaml), &idxNode)

	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())

	sp := new(lowbase.SchemaProxy)
	err := sp.Build(context.Background(), nil, compNode.Content[0], idx)
	assert.NoError(t, err)

	lowproxy := low.NodeReference[*lowbase.SchemaProxy]{
		Value:     sp,
		ValueNode: compNode.Content[0],
	}

	schemaProxy := NewSchemaProxy(&lowproxy)
	compiled := schemaProxy.Schema()

	// Use validation mode context - refs should be inlined
	ctx := NewInlineRenderContextForValidation()
	result, err := compiled.MarshalYAMLInlineWithContext(ctx)
	assert.NoError(t, err)

	// Marshal to YAML to check output
	yamlBytes, _ := yaml.Marshal(result)
	output := string(yamlBytes)

	// In validation mode, the anyOf refs should be INLINED, not preserved as $ref
	// Should contain the inlined properties
	assert.Contains(t, output, "meow:")
	assert.Contains(t, output, "bark:")
}

func TestSchema_MarshalYAMLInlineWithContext_BundleMode_PreservesDiscriminatorRefs(t *testing.T) {
	// Test that in bundle mode (default), discriminator refs are preserved

	idxYaml := `openapi: 3.1.0
components:
  schemas:
    Cat:
      type: object
      properties:
        type:
          type: string
        meow:
          type: boolean`

	testSpec := `discriminator:
  propertyName: type
oneOf:
  - $ref: '#/components/schemas/Cat'`

	var compNode, idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(testSpec), &compNode)
	_ = yaml.Unmarshal([]byte(idxYaml), &idxNode)

	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())

	sp := new(lowbase.SchemaProxy)
	err := sp.Build(context.Background(), nil, compNode.Content[0], idx)
	assert.NoError(t, err)

	lowproxy := low.NodeReference[*lowbase.SchemaProxy]{
		Value:     sp,
		ValueNode: compNode.Content[0],
	}

	schemaProxy := NewSchemaProxy(&lowproxy)
	compiled := schemaProxy.Schema()

	// Use bundle mode context (default) - refs should be preserved
	ctx := NewInlineRenderContext()
	assert.Equal(t, RenderingModeBundle, ctx.Mode)

	result, err := compiled.MarshalYAMLInlineWithContext(ctx)
	assert.NoError(t, err)

	// Marshal to YAML to check output
	yamlBytes, _ := yaml.Marshal(result)
	output := string(yamlBytes)

	// In bundle mode, the oneOf refs should be PRESERVED as $ref
	assert.Contains(t, output, "$ref:")
	assert.Contains(t, output, "#/components/schemas/Cat")
	// Should NOT contain the inlined properties
	assert.NotContains(t, output, "meow:")
}

func TestSchema_MarshalYAMLInlineWithContext_NilContext_PreservesDiscriminatorRefs(t *testing.T) {
	// Test that with nil context (backward compatibility), discriminator refs are preserved

	idxYaml := `openapi: 3.1.0
components:
  schemas:
    Cat:
      type: object
      properties:
        meow:
          type: boolean`

	testSpec := `discriminator:
  propertyName: type
oneOf:
  - $ref: '#/components/schemas/Cat'`

	var compNode, idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(testSpec), &compNode)
	_ = yaml.Unmarshal([]byte(idxYaml), &idxNode)

	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())

	sp := new(lowbase.SchemaProxy)
	err := sp.Build(context.Background(), nil, compNode.Content[0], idx)
	assert.NoError(t, err)

	lowproxy := low.NodeReference[*lowbase.SchemaProxy]{
		Value:     sp,
		ValueNode: compNode.Content[0],
	}

	schemaProxy := NewSchemaProxy(&lowproxy)
	compiled := schemaProxy.Schema()

	// Pass nil context - should behave like bundle mode (backward compatible)
	result, err := compiled.MarshalYAMLInlineWithContext(nil)
	assert.NoError(t, err)

	// Marshal to YAML to check output
	yamlBytes, _ := yaml.Marshal(result)
	output := string(yamlBytes)

	// Nil context should preserve refs (like bundle mode)
	assert.Contains(t, output, "$ref:")
	assert.Contains(t, output, "#/components/schemas/Cat")
	// Should NOT contain the inlined properties
	assert.NotContains(t, output, "meow:")
}

func TestSchema_MarshalYAMLInlineWithContext_NoDiscriminator_ModeDoesNotMatter(t *testing.T) {
	// Test that without discriminator, both modes behave the same (refs inlined)

	idxYaml := `openapi: 3.1.0
components:
  schemas:
    Cat:
      type: object
      properties:
        meow:
          type: boolean`

	testSpec := `oneOf:
  - $ref: '#/components/schemas/Cat'`

	var compNode, idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(testSpec), &compNode)
	_ = yaml.Unmarshal([]byte(idxYaml), &idxNode)

	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())

	sp := new(lowbase.SchemaProxy)
	err := sp.Build(context.Background(), nil, compNode.Content[0], idx)
	assert.NoError(t, err)

	lowproxy := low.NodeReference[*lowbase.SchemaProxy]{
		Value:     sp,
		ValueNode: compNode.Content[0],
	}

	schemaProxy := NewSchemaProxy(&lowproxy)
	compiled := schemaProxy.Schema()

	// Test with bundle mode
	ctxBundle := NewInlineRenderContext()
	resultBundle, err := compiled.MarshalYAMLInlineWithContext(ctxBundle)
	assert.NoError(t, err)
	yamlBundle, _ := yaml.Marshal(resultBundle)

	// Without discriminator, refs should be inlined in both modes
	// Need to reset the proxy for second render
	schemaProxy2 := NewSchemaProxy(&lowproxy)
	compiled2 := schemaProxy2.Schema()

	ctxValidation := NewInlineRenderContextForValidation()
	resultValidation, err := compiled2.MarshalYAMLInlineWithContext(ctxValidation)
	assert.NoError(t, err)
	yamlValidation, _ := yaml.Marshal(resultValidation)

	// Both should contain the inlined properties (refs not preserved without discriminator)
	assert.Contains(t, string(yamlBundle), "meow:")
	assert.Contains(t, string(yamlValidation), "meow:")
}

// TestNewSchema_Id tests that the $id field is correctly mapped from low to high level
func TestNewSchema_Id(t *testing.T) {
	yml := `type: object
$id: "https://example.com/schemas/pet.json"
description: A pet schema`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	var lowSch lowbase.Schema
	_ = low.BuildModel(idxNode.Content[0], &lowSch)
	_ = lowSch.Build(context.Background(), idxNode.Content[0], nil)

	highSch := NewSchema(&lowSch)

	assert.Equal(t, "https://example.com/schemas/pet.json", highSch.Id)
	assert.Equal(t, "object", highSch.Type[0])
	assert.Equal(t, "A pet schema", highSch.Description)
}

// TestNewSchema_Id_Empty tests that empty $id results in empty string
func TestNewSchema_Id_Empty(t *testing.T) {
	yml := `type: object
description: A schema without $id`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	var lowSch lowbase.Schema
	_ = low.BuildModel(idxNode.Content[0], &lowSch)
	_ = lowSch.Build(context.Background(), idxNode.Content[0], nil)

	highSch := NewSchema(&lowSch)

	assert.Equal(t, "", highSch.Id)
}

// TestNewSchema_Comment tests that $comment is populated in high-level schema
func TestNewSchema_Comment(t *testing.T) {
	yml := `type: object
$comment: This is a test comment explaining the schema purpose
description: A schema with $comment`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	var lowSch lowbase.Schema
	_ = low.BuildModel(idxNode.Content[0], &lowSch)
	_ = lowSch.Build(context.Background(), idxNode.Content[0], nil)

	highSch := NewSchema(&lowSch)

	assert.Equal(t, "This is a test comment explaining the schema purpose", highSch.Comment)
	assert.Equal(t, "object", highSch.Type[0])
}

// TestNewSchema_ContentSchema tests that contentSchema is populated in high-level schema
func TestNewSchema_ContentSchema(t *testing.T) {
	yml := `type: string
contentMediaType: application/json
contentSchema:
  type: object
  properties:
    name:
      type: string`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	var lowSch lowbase.Schema
	_ = low.BuildModel(idxNode.Content[0], &lowSch)
	_ = lowSch.Build(context.Background(), idxNode.Content[0], nil)

	highSch := NewSchema(&lowSch)

	assert.NotNil(t, highSch.ContentSchema)
	contentSch := highSch.ContentSchema.Schema()
	assert.NotNil(t, contentSch)
	assert.Equal(t, "object", contentSch.Type[0])
	assert.NotNil(t, contentSch.Properties)
	assert.Equal(t, 1, contentSch.Properties.Len())
}

// TestNewSchema_Vocabulary tests that $vocabulary is populated in high-level schema
func TestNewSchema_Vocabulary(t *testing.T) {
	yml := `$vocabulary:
  "https://json-schema.org/draft/2020-12/vocab/core": true
  "https://json-schema.org/draft/2020-12/vocab/validation": false
  "https://json-schema.org/draft/2020-12/vocab/applicator": true`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	var lowSch lowbase.Schema
	_ = low.BuildModel(idxNode.Content[0], &lowSch)
	_ = lowSch.Build(context.Background(), idxNode.Content[0], nil)

	highSch := NewSchema(&lowSch)

	assert.NotNil(t, highSch.Vocabulary)
	assert.Equal(t, 3, highSch.Vocabulary.Len())

	// Check specific vocabulary entries
	for k, v := range highSch.Vocabulary.FromOldest() {
		switch k {
		case "https://json-schema.org/draft/2020-12/vocab/core":
			assert.True(t, v)
		case "https://json-schema.org/draft/2020-12/vocab/validation":
			assert.False(t, v)
		case "https://json-schema.org/draft/2020-12/vocab/applicator":
			assert.True(t, v)
		}
	}
}

// TestNewSchema_ContentEncoding tests that contentEncoding is populated in high-level schema
func TestNewSchema_ContentEncoding(t *testing.T) {
	yml := `type: string
contentEncoding: base64
description: A base64 encoded string`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	var lowSch lowbase.Schema
	_ = low.BuildModel(idxNode.Content[0], &lowSch)
	_ = lowSch.Build(context.Background(), idxNode.Content[0], nil)

	highSch := NewSchema(&lowSch)

	assert.Equal(t, "base64", highSch.ContentEncoding)
	assert.Equal(t, "string", highSch.Type[0])
}

// TestNewSchema_ContentMediaType tests that contentMediaType is populated in high-level schema
func TestNewSchema_ContentMediaType(t *testing.T) {
	yml := `type: string
contentMediaType: image/png
description: A binary image encoded as string`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	var lowSch lowbase.Schema
	_ = low.BuildModel(idxNode.Content[0], &lowSch)
	_ = lowSch.Build(context.Background(), idxNode.Content[0], nil)

	highSch := NewSchema(&lowSch)

	assert.Equal(t, "image/png", highSch.ContentMediaType)
	assert.Equal(t, "string", highSch.Type[0])
}
