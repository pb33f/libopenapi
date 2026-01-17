// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"fmt"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

var testComponentsYaml = `
  x-pizza: crispy
  schemas:
    one:
      description: one of many
    two:
      description: two of many
  responses:
    three:
      description: three of many
    four:
      description: four of many
  parameters:
    five:
      description: five of many
    six:
      description: six of many
  examples:
    seven:
      description: seven of many
    eight:
      description: eight of many
  requestBodies:
    nine:
      description: nine of many
    ten:
      description: ten of many
  headers:
    eleven:
      description: eleven of many
    twelve:
      description: twelve of many
  securitySchemes:
    thirteen:
      description: thirteen of many
    fourteen:
      description: fourteen of many
  links:
    fifteen:
      description: fifteen of many
    sixteen:
      description: sixteen of many
  callbacks:
    seventeen:
      '{reference}':
        post:
          description: seventeen of many
    eighteen:
      '{raference}':
        post:
          description: eighteen of many
  pathItems:
    /nineteen:
      get:
        description: nineteen of many
  mediaTypes:
    jsonMediaType:
      schema:
        description: twenty of many`

func TestComponents_Build_Success(t *testing.T) {
	low.ClearHashCache()
	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(testComponentsYaml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Components
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.NotNil(t, n.GetRootNode())
	assert.NotNil(t, n.GetKeyNode())
	assert.Equal(t, "one of many", n.FindSchema("one").Value.Schema().Description.Value)
	assert.Equal(t, "two of many", n.FindSchema("two").Value.Schema().Description.Value)
	assert.Equal(t, "three of many", n.FindResponse("three").Value.Description.Value)
	assert.Equal(t, "four of many", n.FindResponse("four").Value.Description.Value)
	assert.Equal(t, "five of many", n.FindParameter("five").Value.Description.Value)
	assert.Equal(t, "six of many", n.FindParameter("six").Value.Description.Value)
	assert.Equal(t, "seven of many", n.FindExample("seven").Value.Description.Value)
	assert.Equal(t, "eight of many", n.FindExample("eight").Value.Description.Value)
	assert.Equal(t, "nine of many", n.FindRequestBody("nine").Value.Description.Value)
	assert.Equal(t, "ten of many", n.FindRequestBody("ten").Value.Description.Value)
	assert.Equal(t, "eleven of many", n.FindHeader("eleven").Value.Description.Value)
	assert.Equal(t, "twelve of many", n.FindHeader("twelve").Value.Description.Value)
	assert.Equal(t, "thirteen of many", n.FindSecurityScheme("thirteen").Value.Description.Value)
	assert.Equal(t, "fourteen of many", n.FindSecurityScheme("fourteen").Value.Description.Value)
	assert.Equal(t, "fifteen of many", n.FindLink("fifteen").Value.Description.Value)
	assert.Equal(t, "seventeen of many",
		n.FindCallback("seventeen").Value.FindExpression("{reference}").Value.Post.Value.Description.Value)
	assert.Equal(t, "eighteen of many",
		n.FindCallback("eighteen").Value.FindExpression("{raference}").Value.Post.Value.Description.Value)
	assert.Equal(t, "nineteen of many", n.FindPathItem("/nineteen").Value.Get.Value.Description.Value)
	assert.Equal(t, "twenty of many", n.FindMediaType("jsonMediaType").Value.Schema.Value.Schema().Description.Value)

	// maphash uses random seed per process, so just test non-empty
	assert.NotEmpty(t, low.GenerateHashString(&n))

	assert.NotNil(t, n.GetContext())
	assert.NotNil(t, n.GetIndex())
}

func TestComponents_Build_Success_Skip(t *testing.T) {
	low.ClearHashCache()
	yml := `components:`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Components
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)
}

func TestComponents_Build_Fail(t *testing.T) {
	low.ClearHashCache()
	yml := `
  parameters:
    schema:
      $ref: '#/this is a problem.'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Components
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestComponents_Build_ParameterFail(t *testing.T) {
	low.ClearHashCache()
	yml := `
  parameters:
    pizza:
      schema:
        $ref: '#/this is a problem.'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Components
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), idxNode.Content[0], idx)
	assert.Error(t, err)
}

// Test parse failure among many parameters.
// This stresses `TranslatePipeline`'s error handling.
func TestComponents_Build_ParameterFail_Many(t *testing.T) {
	low.ClearHashCache()
	yml := `
  parameters:
`

	for i := 0; i < 1000; i++ {
		format := `
    pizza%d:
      schema:
        $ref: '#/this is a problem.'
`
		yml += fmt.Sprintf(format, i)
	}

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Components
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestComponents_Build_Fail_TypeFail(t *testing.T) {
	low.ClearHashCache()
	yml := `
  parameters:
    - schema:
        $ref: #/this is a problem.`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Components
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestComponents_Build_ExtensionTest(t *testing.T) {
	low.ClearHashCache()
	yml := `x-curry: seagull
headers:
  x-curry-gull: vinadloo`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Components
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)

	var xCurry string
	_ = n.FindExtension("x-curry").Value.Decode(&xCurry)

	assert.Equal(t, "seagull", xCurry)
}

func TestComponents_Build_HashEmpty(t *testing.T) {
	low.ClearHashCache()
	yml := `x-curry: seagull`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Components
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)

	var xCurry string
	_ = n.FindExtension("x-curry").Value.Decode(&xCurry)

	assert.Equal(t, "seagull", xCurry)
	assert.Equal(t, 1, orderedmap.Len(n.GetExtensions()))
	// maphash uses random seed per process, so just test non-empty
	assert.NotEmpty(t, low.GenerateHashString(&n))
}

func TestComponents_IsReference(t *testing.T) {
	low.ClearHashCache()
	yml := `
schemas:
    one:
        description: one of many
    two:
        $ref: "#/schemas/one"
responses:
    three:
        description: three of many
    four:
        $ref: "#/responses/three"
parameters:
    five:
        description: five of many
    six:
        $ref: "#/parameters/five"
examples:
    seven:
        description: seven of many
    eight:
        $ref: "#/examples/seven"
requestBodies:
    nine:
        description: nine of many
    ten:
        $ref: "#/requestBodies/nine"
headers:
    eleven:
        description: eleven of many
    twelve:
        $ref: "#/headers/eleven"
securitySchemes:
    thirteen:
        description: thirteen of many
    fourteen:
        $ref: "#/securitySchemes/thirteen"
links:
    fifteen:
        description: fifteen of many
    sixteen:
        $ref: "#/links/fifteen"
callbacks:
    seventeen:
        '{reference}':
            post:
            description: seventeen of many
    eighteen:
        $ref: "#/callbacks/seventeen"
`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Components
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)

	assert.Equal(t, "#/schemas/one", n.FindSchema("two").Value.GetReference())
	assert.Equal(t, "#/responses/three", n.FindResponse("four").Value.GetReference())
	assert.Equal(t, "#/parameters/five", n.FindParameter("six").Value.GetReference())
	assert.Equal(t, "#/examples/seven", n.FindExample("eight").Value.GetReference())
	assert.Equal(t, "#/requestBodies/nine", n.FindRequestBody("ten").Value.GetReference())
	assert.Equal(t, "#/headers/eleven", n.FindHeader("twelve").Value.GetReference())
	assert.Equal(t, "#/securitySchemes/thirteen", n.FindSecurityScheme("fourteen").Value.GetReference())
	assert.Equal(t, "#/links/fifteen", n.FindLink("sixteen").Value.GetReference())
	assert.Equal(t, "#/callbacks/seventeen", n.FindCallback("eighteen").Value.GetReference())
}

func TestComponents_IsReference_OutOfSpecification_PathItem(t *testing.T) {
	low.ClearHashCache()
	yml := `
pathItems:
    one:
        description: one of many
    two:
        $ref: "#/pathItems/one"
`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Components
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)

	assert.Equal(t, "#/pathItems/one", n.FindPathItem("two").Value.GetReference())
}

func TestComponents_MediaTypes(t *testing.T) {
	low.ClearHashCache()
	yml := `mediaTypes:
  JsonMediaType:
    schema:
      type: object
      properties:
        id:
          type: integer
    examples:
      user:
        value:
          id: 123
          name: John
  XmlMediaType:
    schema:
      type: string
      xml:
        name: xmlData`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Components
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)

	assert.Equal(t, 2, n.MediaTypes.Value.Len())

	jsonMediaType := n.FindMediaType("JsonMediaType")
	assert.NotNil(t, jsonMediaType)
	assert.NotNil(t, jsonMediaType.Value.Schema.Value)
	assert.Equal(t, "object", jsonMediaType.Value.Schema.Value.Schema().Type.Value.A)
	assert.Equal(t, 1, jsonMediaType.Value.Examples.Value.Len())

	xmlMediaType := n.FindMediaType("XmlMediaType")
	assert.NotNil(t, xmlMediaType)
	assert.NotNil(t, xmlMediaType.Value.Schema.Value)
	assert.Equal(t, "string", xmlMediaType.Value.Schema.Value.Schema().Type.Value.A)

	// test hash includes mediaTypes
	hash1 := n.Hash()
	n.MediaTypes = low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[*MediaType]]]{}
	hash2 := n.Hash()
	assert.NotEqual(t, hash1, hash2)
}

// TestComponents_XPrefixedComponentNames tests that component names starting with x- are correctly
// parsed as components and not incorrectly filtered out as extensions.
// This is a regression test for https://github.com/pb33f/libopenapi/issues/503
func TestComponents_XPrefixedComponentNames(t *testing.T) {
	low.ClearHashCache()

	yml := `schemas:
  x-custom-schema:
    type: object
    description: A schema with x- prefix
  RegularSchema:
    type: string
parameters:
  x-custom-param:
    name: x-custom-param
    in: header
    schema:
      type: string
  regular-param:
    name: regular-param
    in: query
    schema:
      type: string
responses:
  x-custom-response:
    description: A response with x- prefix
headers:
  x-rate-limit:
    schema:
      type: integer
    description: Rate limit header
examples:
  x-custom-example:
    value: example-value
    description: An example with x- prefix
requestBodies:
  x-custom-body:
    description: A request body with x- prefix
    content:
      application/json:
        schema:
          type: object
securitySchemes:
  x-custom-auth:
    type: apiKey
    name: X-API-Key
    in: header
    description: Custom auth scheme
links:
  x-custom-link:
    description: A link with x- prefix
callbacks:
  x-custom-callback:
    '{$request.body#/callbackUrl}':
      post:
        description: Callback operation
pathItems:
  /x-custom-path:
    get:
      description: A path item with x- prefix
mediaTypes:
  x-custom-media:
    schema:
      type: object
      description: Custom media type`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Components
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)

	// Test x-prefixed schemas are included
	xSchema := n.FindSchema("x-custom-schema")
	assert.NotNil(t, xSchema, "x-custom-schema should be found")
	assert.Equal(t, "A schema with x- prefix", xSchema.Value.Schema().Description.Value)

	regularSchema := n.FindSchema("RegularSchema")
	assert.NotNil(t, regularSchema, "RegularSchema should also be found")

	// Test x-prefixed parameters are included
	xParam := n.FindParameter("x-custom-param")
	assert.NotNil(t, xParam, "x-custom-param should be found")
	assert.Equal(t, "x-custom-param", xParam.Value.Name.Value)
	assert.Equal(t, "header", xParam.Value.In.Value)

	regularParam := n.FindParameter("regular-param")
	assert.NotNil(t, regularParam, "regular-param should also be found")

	// Test x-prefixed responses are included
	xResponse := n.FindResponse("x-custom-response")
	assert.NotNil(t, xResponse, "x-custom-response should be found")
	assert.Equal(t, "A response with x- prefix", xResponse.Value.Description.Value)

	// Test x-prefixed headers are included
	xHeader := n.FindHeader("x-rate-limit")
	assert.NotNil(t, xHeader, "x-rate-limit should be found")
	assert.Equal(t, "Rate limit header", xHeader.Value.Description.Value)

	// Test x-prefixed examples are included
	xExample := n.FindExample("x-custom-example")
	assert.NotNil(t, xExample, "x-custom-example should be found")
	assert.Equal(t, "An example with x- prefix", xExample.Value.Description.Value)

	// Test x-prefixed request bodies are included
	xRequestBody := n.FindRequestBody("x-custom-body")
	assert.NotNil(t, xRequestBody, "x-custom-body should be found")
	assert.Equal(t, "A request body with x- prefix", xRequestBody.Value.Description.Value)

	// Test x-prefixed security schemes are included
	xSecurityScheme := n.FindSecurityScheme("x-custom-auth")
	assert.NotNil(t, xSecurityScheme, "x-custom-auth should be found")
	assert.Equal(t, "Custom auth scheme", xSecurityScheme.Value.Description.Value)
	assert.Equal(t, "apiKey", xSecurityScheme.Value.Type.Value)

	// Test x-prefixed links are included
	xLink := n.FindLink("x-custom-link")
	assert.NotNil(t, xLink, "x-custom-link should be found")
	assert.Equal(t, "A link with x- prefix", xLink.Value.Description.Value)

	// Test x-prefixed callbacks are included
	xCallback := n.FindCallback("x-custom-callback")
	assert.NotNil(t, xCallback, "x-custom-callback should be found")
	expr := xCallback.Value.FindExpression("{$request.body#/callbackUrl}")
	assert.NotNil(t, expr, "Callback expression should be found")
	assert.Equal(t, "Callback operation", expr.Value.Post.Value.Description.Value)

	// Test x-prefixed path items are included
	xPathItem := n.FindPathItem("/x-custom-path")
	assert.NotNil(t, xPathItem, "/x-custom-path should be found")
	assert.Equal(t, "A path item with x- prefix", xPathItem.Value.Get.Value.Description.Value)

	// Test x-prefixed media types are included
	xMediaType := n.FindMediaType("x-custom-media")
	assert.NotNil(t, xMediaType, "x-custom-media should be found")
	assert.Equal(t, "Custom media type", xMediaType.Value.Schema.Value.Schema().Description.Value)
}

// TestComponents_XPrefixedWithUpperCase tests that both x- (lowercase) and X- (uppercase)
// prefixed component names are correctly parsed.
func TestComponents_XPrefixedWithUpperCase(t *testing.T) {
	low.ClearHashCache()

	yml := `schemas:
  x-lowercase-schema:
    type: string
    description: lowercase x- prefix
  X-UPPERCASE-SCHEMA:
    type: string
    description: uppercase X- prefix
parameters:
  x-lowercase-param:
    name: x-param
    in: header
    schema:
      type: string
  X-UPPERCASE-PARAM:
    name: X-param
    in: header
    schema:
      type: string`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Components
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)

	// Test lowercase x- prefix
	xLowerSchema := n.FindSchema("x-lowercase-schema")
	assert.NotNil(t, xLowerSchema, "x-lowercase-schema should be found")
	assert.Equal(t, "lowercase x- prefix", xLowerSchema.Value.Schema().Description.Value)

	// Test uppercase X- prefix
	xUpperSchema := n.FindSchema("X-UPPERCASE-SCHEMA")
	assert.NotNil(t, xUpperSchema, "X-UPPERCASE-SCHEMA should be found")
	assert.Equal(t, "uppercase X- prefix", xUpperSchema.Value.Schema().Description.Value)

	// Test lowercase x- param
	xLowerParam := n.FindParameter("x-lowercase-param")
	assert.NotNil(t, xLowerParam, "x-lowercase-param should be found")

	// Test uppercase X- param
	xUpperParam := n.FindParameter("X-UPPERCASE-PARAM")
	assert.NotNil(t, xUpperParam, "X-UPPERCASE-PARAM should be found")
}

// TestComponents_XPrefixedExtensionsStillWork verifies that extensions at the Components level
// (like x-custom-extension) are still captured correctly, while x-prefixed component names
// within schemas/parameters/etc are also captured.
func TestComponents_XPrefixedExtensionsStillWork(t *testing.T) {
	low.ClearHashCache()

	yml := `x-components-extension: this is an extension at components level
x-another-extension:
  nested: value
schemas:
  x-custom-schema:
    type: object
    description: This is a schema, not an extension
  RegularSchema:
    type: string`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Components
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)

	// Extensions at Components level should still work
	ext1 := n.FindExtension("x-components-extension")
	assert.NotNil(t, ext1, "x-components-extension should be found as extension")
	var ext1Val string
	_ = ext1.Value.Decode(&ext1Val)
	assert.Equal(t, "this is an extension at components level", ext1Val)

	ext2 := n.FindExtension("x-another-extension")
	assert.NotNil(t, ext2, "x-another-extension should be found as extension")

	// x-prefixed schemas should be found as schemas
	xSchema := n.FindSchema("x-custom-schema")
	assert.NotNil(t, xSchema, "x-custom-schema should be found as a schema")
	assert.Equal(t, "This is a schema, not an extension", xSchema.Value.Schema().Description.Value)

	// Regular schemas also work
	regularSchema := n.FindSchema("RegularSchema")
	assert.NotNil(t, regularSchema, "RegularSchema should be found")
}

// TestComponents_XPrefixedReferenceResolution tests that references to x-prefixed components
// resolve correctly.
func TestComponents_XPrefixedReferenceResolution(t *testing.T) {
	low.ClearHashCache()

	yml := `schemas:
  x-base-schema:
    type: object
    properties:
      id:
        type: integer
  derived-schema:
    allOf:
      - $ref: '#/schemas/x-base-schema'
      - type: object
        properties:
          name:
            type: string
parameters:
  x-auth-header:
    name: Authorization
    in: header
    schema:
      type: string
  uses-x-param:
    $ref: '#/parameters/x-auth-header'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Components
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), idxNode.Content[0], idx)
	assert.NoError(t, err)

	// The x-prefixed schema should be found
	xBaseSchema := n.FindSchema("x-base-schema")
	assert.NotNil(t, xBaseSchema, "x-base-schema should be found")

	// The derived schema should also be found
	derivedSchema := n.FindSchema("derived-schema")
	assert.NotNil(t, derivedSchema, "derived-schema should be found")

	// The x-prefixed parameter should be found
	xAuthHeader := n.FindParameter("x-auth-header")
	assert.NotNil(t, xAuthHeader, "x-auth-header should be found")
	assert.Equal(t, "Authorization", xAuthHeader.Value.Name.Value)

	// The parameter that references x-auth-header should have reference
	usesXParam := n.FindParameter("uses-x-param")
	assert.NotNil(t, usesXParam, "uses-x-param should be found")
	assert.Equal(t, "#/parameters/x-auth-header", usesXParam.Value.GetReference())
}
