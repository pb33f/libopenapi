// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
    "github.com/stretchr/testify/assert"
    "gopkg.in/yaml.v3"
    "testing"
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
                    items: true
   `
    var rootNode yaml.Node
    _ = yaml.Unmarshal([]byte(yml), &rootNode)
    c := CreateOpenAPIIndexConfig()
    idx := NewSpecIndexWithConfig(&rootNode, c)
    assert.Len(t, idx.allInlineSchemaDefinitions, 2)
    assert.Len(t, idx.allInlineSchemaObjectDefinitions, 1)
}