// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
    "github.com/pb33f/libopenapi/datamodel/low"
    v3 "github.com/pb33f/libopenapi/datamodel/low/3.0"
    "github.com/pb33f/libopenapi/index"
    "github.com/stretchr/testify/assert"
    "gopkg.in/yaml.v3"
    "testing"
)

func TestNewSchema(t *testing.T) {

    // tests async schema lookup, by essentially running it twice, without a cache cleanup.
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
    $ref: '#/components/schemas/rice'`

    var n v3.Schema
    _ = yaml.Unmarshal([]byte(yml), &compNode)
    err := low.BuildModel(&idxNode, &n)
    assert.NoError(t, err)

    err = n.Build(idxNode.Content[0], idx)
    assert.NoError(t, err)

    sch1 := NewSchema(&n)
    sch2 := NewSchema(&n)

    assert.Equal(t, sch1, sch2)

}