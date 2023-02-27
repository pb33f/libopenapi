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
