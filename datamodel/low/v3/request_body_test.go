// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestRequestBody_Build(t *testing.T) {

	yml := `description: a nice request
required: true
content:
  fresh/fish:
    example: nice.
x-requesto: presto`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n RequestBody
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(nil, idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "a nice request", n.Description.Value)
	assert.True(t, n.Required.Value)
	assert.Equal(t, "nice.", n.FindContent("fresh/fish").Value.Example.Value)
	assert.Equal(t, "presto", n.FindExtension("x-requesto").Value)
	assert.Len(t, n.GetExtensions(), 1)

}

func TestRequestBody_Fail(t *testing.T) {

	yml := `content:
  $ref: #illegal`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n RequestBody
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(nil, idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestRequestBody_Hash(t *testing.T) {

	yml := `description: nice toast
content:
  jammy/toast:
    schema:
      type: int
  honey/toast:
    schema:
      type: int
required: true
x-toast: nice
`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n RequestBody
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(nil, idxNode.Content[0], idx)

	yml2 := `description: nice toast
content:
  jammy/toast:
    schema:
      type: int
  honey/toast:
    schema:
      type: int
required: true
x-toast: nice`

	var idxNode2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml2), &idxNode2)
	idx2 := index.NewSpecIndex(&idxNode2)

	var n2 RequestBody
	_ = low.BuildModel(idxNode2.Content[0], &n2)
	_ = n2.Build(nil, idxNode2.Content[0], idx2)

	// hash
	assert.Equal(t, n.Hash(), n2.Hash())

}
