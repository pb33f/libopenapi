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

// https://github.com/pb33f/libopenapi/issues/112
func TestSpecIndex_ExtractRefs_CheckReferencesWithBracketsInName(t *testing.T) {

	yml := `openapi: 3.0.0
components:
  schemas:
    Cake[Burger]:
      type: string
      description: A cakey burger
    Happy:
      type: object
      properties:
        mingo:
          $ref: '#/components/schemas/Cake[Burger]'
   `
	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)
	c := CreateOpenAPIIndexConfig()
	idx := NewSpecIndexWithConfig(&rootNode, c)
	assert.Len(t, idx.allMappedRefs, 1)
	assert.Equal(t, "Cake[Burger]", idx.allMappedRefs["#/components/schemas/Cake[Burger]"].Name)
}

// https://github.com/daveshanley/vacuum/issues/339
func TestSpecIndex_ExtractRefs_CheckEnumNotPropertyCalledEnum(t *testing.T) {

	yml := `openapi: 3.0.0
components:
  schemas:
    SimpleFieldSchema:
      description: Schema of a field as described in  JSON Schema draft 2019-09
      type: object
      required:
        - type
        - description
      properties:
        type:
          type: string
          enum:
            - string
            - number
        description:
          type: string
          description: A description of the property
        enum:
          type: array
          description: A array of describing the possible values
          items:
            type: string
          example:
            - yo
            - hello
   `
	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)
	c := CreateOpenAPIIndexConfig()
	idx := NewSpecIndexWithConfig(&rootNode, c)
	assert.Len(t, idx.allEnums, 1)

}
