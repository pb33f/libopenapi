// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestSpecIndex_ExtractRefs_CheckDescriptionNotMap(t *testing.T) {
	yml := `openapi: 3.1.0
info:
  description: This is a description
paths:
  /herbs/and/spice:
    get:
      description: This is a also a description
      responses:
        200:
          content:
            application/json:
              schema:
                type: array
                properties:
                  description:
                   type: string
   `
	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)
	c := CreateOpenAPIIndexConfig()
	idx := NewSpecIndexWithConfig(&rootNode, c)
	assert.Len(t, idx.allDescriptions, 2)
	assert.Equal(t, 2, idx.descriptionCount)
}

func TestSpecIndex_ExtractRefs_CheckPropertiesForInlineSchema(t *testing.T) {
	yml := `openapi: 3.1.0
servers:
  - url: http://localhost:8080
paths:
  /test:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                type: object
                properties:
                  test:
                    type: array
                    items:
                      type: object
                    prefixItems:
                      - $ref: '#/components/schemas/Test'
                    additionalProperties: false
                    unevaluatedProperties: false
components:
  schemas:
    Test:
      type: object
      additionalProperties:
        type: string
      contains:
        type: string
      not:
        type: number
      unevaluatedProperties:
        type: boolean
      patternProperties:
        ^S_:
          type: string
        ^I_:
          type: integer
      prefixItems:
        - type: string
    AllOf:
      allOf:
        - type: object
          properties:
            test:
              type: string
        - type: object
          properties:
            test2:
              type: string
    AnyOf:
      anyOf:
        - type: object
          properties:
            test:
              type: string
        - type: object
          properties:
            test2:
              type: string
    OneOf:
      oneOf:
        - type: string
        - type: number
        - type: boolean
   `
	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)
	c := CreateOpenAPIIndexConfig()
	idx := NewSpecIndexWithConfig(&rootNode, c)
	assert.Len(t, idx.allInlineSchemaDefinitions, 21)
	assert.Len(t, idx.allInlineSchemaObjectDefinitions, 7)
}
