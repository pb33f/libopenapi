// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package bundler

import (
	"testing"

	v3 "github.com/pkg-base/libopenapi/datamodel/low/v3"
	"github.com/pkg-base/yaml"
	"github.com/stretchr/testify/assert"
)

func TestDetectOpenAPIComponentType_NilNode(t *testing.T) {
	componentType, detected := DetectOpenAPIComponentType(nil)
	assert.Equal(t, "", componentType)
	assert.False(t, detected)
}

func TestDetectOpenAPIComponentType_Schema(t *testing.T) {
	// Test schema with type property
	schemaYaml := `
type: object
properties:
  name:
    type: string
`
	node := parseYaml(t, schemaYaml)
	componentType, detected := DetectOpenAPIComponentType(node)
	assert.Equal(t, v3.SchemasLabel, componentType)
	assert.True(t, detected)

	// Test schema with allOf property
	schemaYaml = `
allOf:
  - $ref: '#/components/schemas/Pet'
  - type: object
    properties:
      name:
        type: string
`
	node = parseYaml(t, schemaYaml)
	componentType, detected = DetectOpenAPIComponentType(node)
	assert.Equal(t, v3.SchemasLabel, componentType)
	assert.True(t, detected)

	// Test schema with anyOf property
	schemaYaml = `
anyOf:
  - type: string
  - type: integer
`
	node = parseYaml(t, schemaYaml)
	componentType, detected = DetectOpenAPIComponentType(node)
	assert.Equal(t, v3.SchemasLabel, componentType)
	assert.True(t, detected)

	// Test schema with oneOf property
	schemaYaml = `
oneOf:
  - type: string
  - type: integer
`
	node = parseYaml(t, schemaYaml)
	componentType, detected = DetectOpenAPIComponentType(node)
	assert.Equal(t, v3.SchemasLabel, componentType)
	assert.True(t, detected)

	// Test schema with enum property
	schemaYaml = `
type: string
enum:
  - available
  - pending
  - sold
`
	node = parseYaml(t, schemaYaml)
	componentType, detected = DetectOpenAPIComponentType(node)
	assert.Equal(t, v3.SchemasLabel, componentType)
	assert.True(t, detected)

	// Test schema with items property
	schemaYaml = `
type: array
items:
  type: string
`
	node = parseYaml(t, schemaYaml)
	componentType, detected = DetectOpenAPIComponentType(node)
	assert.Equal(t, v3.SchemasLabel, componentType)
	assert.True(t, detected)
}

func TestDetectOpenAPIComponentType_Response(t *testing.T) {
	// Test response with description and content
	responseYaml := `
description: A successful response
content:
  application/json:
    schema:
      $ref: '#/components/schemas/Pet'
`
	node := parseYaml(t, responseYaml)
	componentType, detected := DetectOpenAPIComponentType(node)
	assert.Equal(t, v3.ResponsesLabel, componentType)
	assert.True(t, detected)

	// Test response with description and headers
	responseYaml = `
description: A successful response
headers:
  X-Rate-Limit:
    description: Rate limit information
    schema:
      type: integer
`
	node = parseYaml(t, responseYaml)
	componentType, detected = DetectOpenAPIComponentType(node)
	assert.Equal(t, v3.ResponsesLabel, componentType)
	assert.True(t, detected)

	// Test object with description but no content or headers (not a response)
	responseYaml = `
description: Just a description
`
	node = parseYaml(t, responseYaml)
	componentType, detected = DetectOpenAPIComponentType(node)
	assert.NotEqual(t, v3.ResponsesLabel, componentType)
	assert.False(t, detected)
}

func TestDetectOpenAPIComponentType_Parameter(t *testing.T) {
	// Test parameter with name and in
	parameterYaml := `
name: petId
in: path
description: ID of pet to return
required: true
schema:
  type: integer
  format: int64
`
	node := parseYaml(t, parameterYaml)
	componentType, detected := DetectOpenAPIComponentType(node)
	assert.Equal(t, v3.ParametersLabel, componentType)
	assert.True(t, detected)
}

func TestDetectOpenAPIComponentType_Example(t *testing.T) {
	// Test example with value
	exampleYaml := `
summary: A pet example
value:
  name: Fluffy
  petType: Cat
`
	node := parseYaml(t, exampleYaml)
	componentType, detected := DetectOpenAPIComponentType(node)
	assert.Equal(t, v3.ExamplesLabel, componentType)
	assert.True(t, detected)

	// Test example with externalValue
	exampleYaml = `
summary: A pet example
externalValue: https://example.com/examples/pet.json
`
	node = parseYaml(t, exampleYaml)
	componentType, detected = DetectOpenAPIComponentType(node)
	assert.Equal(t, v3.ExamplesLabel, componentType)
	assert.True(t, detected)
}

func TestDetectOpenAPIComponentType_Link(t *testing.T) {
	// Test link with operationId
	linkYaml := `
operationId: getPetById
parameters:
  petId: $request.path.id
`
	node := parseYaml(t, linkYaml)
	componentType, detected := DetectOpenAPIComponentType(node)
	assert.Equal(t, v3.LinksLabel, componentType)
	assert.True(t, detected)

	// Test link with operationRef
	linkYaml = `
operationRef: '#/paths/~1pets~1{petId}/get'
parameters:
  petId: $request.path.id
`
	node = parseYaml(t, linkYaml)
	componentType, detected = DetectOpenAPIComponentType(node)
	assert.Equal(t, v3.LinksLabel, componentType)
	assert.True(t, detected)
}

func TestDetectOpenAPIComponentType_Callback(t *testing.T) {
	// Test callback with path expression
	callbackYaml := `
'{$request.body#/callbackUrl}':
  post:
    requestBody:
      description: Callback payload
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/CallbackPayload'
    responses:
      '200':
        description: callback successfully processed
`
	node := parseYaml(t, callbackYaml)
	componentType, detected := DetectOpenAPIComponentType(node)
	assert.Equal(t, v3.CallbacksLabel, componentType)
	assert.True(t, detected)

	// Test object without path expression (not a callback)
	callbackYaml = `
regularKey:
  post:
    responses:
      '200':
        description: OK
`
	node = parseYaml(t, callbackYaml)
	componentType, detected = DetectOpenAPIComponentType(node)
	assert.NotEqual(t, v3.CallbacksLabel, componentType)
	assert.False(t, detected)
}

func TestDetectOpenAPIComponentType_PathItem(t *testing.T) {
	// Test pathItem with HTTP method
	pathItemYaml := `
get:
  summary: Get a pet
  operationId: getPet
  responses:
    '200':
      description: Successful operation
post:
  summary: Create a pet
  operationId: createPet
  responses:
    '201':
      description: Pet created
`
	node := parseYaml(t, pathItemYaml)
	componentType, detected := DetectOpenAPIComponentType(node)
	assert.Equal(t, v3.PathItemsLabel, componentType)
	assert.True(t, detected)

	// Test pathItem with parameters
	pathItemYaml = `
parameters:
  - name: petId
    in: path
    required: true
    schema:
      type: integer
get:
  summary: Get a pet
  responses:
    '200':
      description: Successful operation
`
	node = parseYaml(t, pathItemYaml)
	componentType, detected = DetectOpenAPIComponentType(node)
	assert.Equal(t, v3.PathItemsLabel, componentType)
	assert.True(t, detected)
}

func TestDetectOpenAPIComponentType_UnknownComponent(t *testing.T) {
	// Test object that doesn't match any component type
	unknownYaml := `
someProp: someValue
anotherProp: anotherValue
`
	node := parseYaml(t, unknownYaml)
	componentType, detected := DetectOpenAPIComponentType(node)
	assert.Equal(t, "", componentType)
	assert.False(t, detected)
}

func TestHasSchemaProperties(t *testing.T) {
	// Test with valid schema
	schemaYaml := `
type: object
properties:
  name:
    type: string
`
	node := parseYaml(t, schemaYaml)
	assert.True(t, hasSchemaProperties(node))

	// Test with non-schema
	nonSchemaYaml := `
description: Not a schema
`
	node = parseYaml(t, nonSchemaYaml)
	assert.False(t, hasSchemaProperties(node))
}

func TestHasResponseProperties(t *testing.T) {
	// Test with valid response
	responseYaml := `
description: A response
content:
  application/json:
    schema:
      type: object
`
	node := parseYaml(t, responseYaml)
	assert.True(t, hasResponseProperties(node))

	// Test with description only (not enough)
	nonResponseYaml := `
description: Just a description
`
	node = parseYaml(t, nonResponseYaml)
	assert.False(t, hasResponseProperties(node))
}

func TestHasParameterProperties(t *testing.T) {
	// Test with valid parameter
	parameterYaml := `
name: petId
in: path
`
	node := parseYaml(t, parameterYaml)
	assert.True(t, hasParameterProperties(node))

	// Test with name but no in
	nonParameterYaml := `
name: petId
`
	node = parseYaml(t, nonParameterYaml)
	assert.True(t, hasParameterProperties(node))

	// Test with in but no name
	nonParameterYaml = `
in: path
`
	node = parseYaml(t, nonParameterYaml)
	assert.True(t, hasParameterProperties(node))
}

func TestHasRequestBodyProperties(t *testing.T) {
	// Test with valid requestBody
	requestBodyYaml := `
content:
  application/json:
    schema:
      type: object
`
	node := parseYaml(t, requestBodyYaml)
	assert.True(t, hasRequestBodyProperties(node))

	// Test without content
	nonRequestBodyYaml := `
description: Not a request body
`
	node = parseYaml(t, nonRequestBodyYaml)
	assert.False(t, hasRequestBodyProperties(node))
}

func TestHasHeaderProperties(t *testing.T) {
	// Test with valid header (schema)
	headerYaml := `
schema:
  type: string
`
	node := parseYaml(t, headerYaml)
	assert.True(t, hasHeaderProperties(node))

	// Test with valid header (content)
	headerYaml = `
content:
  application/json:
    schema:
      type: string
`
	node = parseYaml(t, headerYaml)
	assert.True(t, hasHeaderProperties(node))

	// Test with schema but also with parameter properties
	nonHeaderYaml := `
schema:
  type: string
name: X-Header
in: header
`
	node = parseYaml(t, nonHeaderYaml)
	assert.False(t, hasHeaderProperties(node))

	// Test without schema or content
	nonHeaderYaml = `
description: Not a header
`
	node = parseYaml(t, nonHeaderYaml)
	assert.False(t, hasHeaderProperties(node))
}

func TestHasExampleProperties(t *testing.T) {
	// Test with valid example (value)
	exampleYaml := `
value:
  name: Fluffy
`
	node := parseYaml(t, exampleYaml)
	assert.True(t, hasExampleProperties(node))

	// Test with valid example (externalValue)
	exampleYaml = `
externalValue: https://example.com/example.json
`
	node = parseYaml(t, exampleYaml)
	assert.True(t, hasExampleProperties(node))

	// Test without value or externalValue
	nonExampleYaml := `
description: Not an example
`
	node = parseYaml(t, nonExampleYaml)
	assert.False(t, hasExampleProperties(node))
}

func TestHasLinkProperties(t *testing.T) {
	// Test with valid link (operationId)
	linkYaml := `
operationId: getPetById
`
	node := parseYaml(t, linkYaml)
	assert.True(t, hasLinkProperties(node))

	// Test with valid link (operationRef)
	linkYaml = `
operationRef: '#/paths/~1pets/get'
`
	node = parseYaml(t, linkYaml)
	assert.True(t, hasLinkProperties(node))

	// Test without operationId or operationRef
	nonLinkYaml := `
description: Not a link
`
	node = parseYaml(t, nonLinkYaml)
	assert.False(t, hasLinkProperties(node))
}

func TestHasCallbackProperties(t *testing.T) {
	// Test with valid callback
	callbackYaml := `
'{$request.body#/callbackUrl}':
  post:
    responses:
      '200':
        description: OK
`
	node := parseYaml(t, callbackYaml)
	assert.True(t, hasCallbackProperties(node))

	// Test with regular keys (not a callback)
	nonCallbackYaml := `
regularKey:
  post:
    responses:
      '200':
        description: OK
`
	node = parseYaml(t, nonCallbackYaml)
	assert.False(t, hasCallbackProperties(node))

	// Test non-mapping node
	nonCallbackYaml = `
- item1
- item2
`
	node = parseYaml(t, nonCallbackYaml)
	assert.False(t, hasCallbackProperties(node))

	// Test empty mapping node
	nonCallbackYaml = `{}`
	node = parseYaml(t, nonCallbackYaml)
	assert.False(t, hasCallbackProperties(node))
}

func TestHasPathItemProperties(t *testing.T) {
	// Test with valid pathItem (HTTP method)
	pathItemYaml := `
get:
  responses:
    '200':
      description: OK
`
	node := parseYaml(t, pathItemYaml)
	assert.True(t, hasPathItemProperties(node))

	// Test with valid pathItem (parameters)
	pathItemYaml = `
parameters:
  - name: petId
    in: path
    required: true
`
	node = parseYaml(t, pathItemYaml)
	assert.True(t, hasPathItemProperties(node))

	// Test without HTTP methods or parameters
	nonPathItemYaml := `
description: Not a path item
`
	node = parseYaml(t, nonPathItemYaml)
	assert.False(t, hasPathItemProperties(node))
}

func TestGetNodeKeys(t *testing.T) {
	// Test with mapping node
	yamlStr := `
key1: value1
key2: value2
key3: value3
`
	node := parseYaml(t, yamlStr)
	keys := getNodeKeys(node)
	assert.ElementsMatch(t, []string{"key1", "key2", "key3"}, keys)

	// Test with sequence node
	yamlStr = `
- item1
- item2
- item3
`
	node = parseYaml(t, yamlStr)
	keys = getNodeKeys(node)
	assert.Nil(t, keys)

	// Test with scalar node
	yamlStr = `scalar value`
	node = parseYaml(t, yamlStr)
	keys = getNodeKeys(node)
	assert.Nil(t, keys)
}

func TestContainsKey(t *testing.T) {
	keys := []string{"key1", "key2", "key3"}

	assert.True(t, containsKey(keys, "key1"))
	assert.True(t, containsKey(keys, "key2"))
	assert.True(t, containsKey(keys, "key3"))
	assert.False(t, containsKey(keys, "key4"))
	assert.False(t, containsKey(keys, ""))
	assert.False(t, containsKey(nil, "key1"))
}

func TestGetNodeValueForKey(t *testing.T) {
	// Test with mapping node
	yamlStr := `
key1: value1
key2: value2
key3: value3
`
	node := parseYaml(t, yamlStr)

	assert.Equal(t, "value1", getNodeValueForKey(node, "key1"))
	assert.Equal(t, "value2", getNodeValueForKey(node, "key2"))
	assert.Equal(t, "value3", getNodeValueForKey(node, "key3"))
	assert.Equal(t, "", getNodeValueForKey(node, "key4"))

	// Test with sequence node
	yamlStr = `
- item1
- item2
`
	node = parseYaml(t, yamlStr)
	assert.Equal(t, "", getNodeValueForKey(node, "key"))

	// Test with scalar node
	yamlStr = `scalar value`
	node = parseYaml(t, yamlStr)
	assert.Equal(t, "", getNodeValueForKey(node, "key"))
}

// Helper function to parse YAML into a yaml.Node
func parseYaml(t *testing.T, yamlStr string) *yaml.Node {
	var node yaml.Node
	err := yaml.Unmarshal([]byte(yamlStr), &node)
	assert.NoError(t, err)

	// The root node is a document node, we want its content
	if len(node.Content) > 0 {
		return node.Content[0]
	}
	return &node
}
