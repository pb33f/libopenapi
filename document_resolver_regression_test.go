// Copyright 2026 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package libopenapi

import (
	"fmt"
	"github.com/pb33f/testify/require"
	"testing"
)

func TestBuildV3Model_RecursiveRequiredAlias(t *testing.T) {
	config := externalRegressionConfig(t, map[string]string{
		"schemas/id.yaml":   "$ref: ./uuid.yaml\n",
		"schemas/uuid.yaml": "type: string\nformat: uuid\n",
	})
	doc, err := NewDocumentWithConfiguration([]byte(`openapi: 3.1.0
info: {title: Recursive required alias, version: "1"}
paths: {}
components:
  schemas:
    Item:
      type: object
      required: [id]
      properties:
        id: {$ref: ./schemas/id.yaml}
        children:
          type: array
          items: {$ref: '#/components/schemas/Item'}
`), config)
	require.NoError(t, err)
	model, err := doc.BuildV3Model()
	require.NoError(t, err)
	item := model.Model.Components.Schemas.GetOrZero("Item").Schema()
	require.NotNil(t, item)
	require.Equal(t, "uuid", item.Properties.GetOrZero("id").Schema().Format)
}

func TestBuildV3Model_ReferencedExamplePayload(t *testing.T) {
	doc, err := NewDocument([]byte(`openapi: 3.1.0
info: {title: Example payload, version: "1"}
paths:
  /items:
    get:
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Item'}
components:
  schemas:
    Item:
      type: object
      properties:
        schema: {type: object}
      example:
        schema:
          properties:
            id: {$ref: '#/$defs/id'}
          $defs:
            id: {type: string}
`))
	require.NoError(t, err)
	model, err := doc.BuildV3Model()
	require.NoError(t, err)
	require.NotNil(t, model)
}

func TestBuildV3Model_ReferencedExampleObjects(t *testing.T) {
	for _, version := range []string{"3.0.3", "3.1.0"} {
		for _, field := range []string{"value", "dataValue"} {
			t.Run(version+"/"+field, func(t *testing.T) {
				response := fmt.Sprintf(`description: ok
content:
  application/json:
    schema:
      type: object
      properties:
        example: {$ref: ./string.yaml}
        examples: {$ref: ./string.yaml}
    examples:
      payload:
        %s:
          $ref: '#/payload-only'
          nested: {$ref: '#/also-payload-only'}
`, field)
				config := externalRegressionConfig(t, map[string]string{
					"responses/ok.yaml":     response,
					"responses/string.yaml": "type: string\n",
				})
				root := fmt.Sprintf(`openapi: %s
info: {title: External example payload, version: "1"}
paths:
  /items:
    get:
      responses:
        '200': {$ref: ./responses/ok.yaml}
`, version)
				doc, err := NewDocumentWithConfiguration([]byte(root), config)
				require.NoError(t, err)
				model, err := doc.BuildV3Model()
				require.NoError(t, err)
				media := model.Model.Paths.PathItems.GetOrZero("/items").Get.Responses.Codes.GetOrZero("200").Content.GetOrZero("application/json")
				for _, name := range []string{"example", "examples"} {
					proxy := media.Schema.Schema().Properties.GetOrZero(name)
					require.Equal(t, []string{"string"}, proxy.Schema().Type)
					require.NoError(t, proxy.GetBuildError())
				}
			})
		}
	}
}
