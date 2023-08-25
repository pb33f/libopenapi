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

func TestMediaType_Build(t *testing.T) {
	yml := `schema:
  type: string
example: hello
examples:
  what:
    value: why?
  where:
    value: there?
encoding:
  chicken:
    explode: true
x-rock: and roll`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n MediaType
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(nil, idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "and roll", n.FindExtension("x-rock").Value)
	assert.Equal(t, "string", n.Schema.Value.Schema().Type.Value.A)
	assert.Equal(t, "hello", n.Example.Value)
	assert.Equal(t, "why?", n.FindExample("what").Value.Value.Value)
	assert.Equal(t, "there?", n.FindExample("where").Value.Value.Value)
	assert.True(t, n.FindPropertyEncoding("chicken").Value.Explode.Value)
	assert.Len(t, n.GetAllExamples(), 2)
}

func TestMediaType_Build_Fail_Schema(t *testing.T) {
	yml := `schema:
  $ref: #bork`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n MediaType
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(nil, idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestMediaType_Build_Fail_Examples(t *testing.T) {
	yml := `examples:
  waff:
    $ref: #bork`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n MediaType
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(nil, idxNode.Content[0], idx)
	assert.Error(t, err)

}

func TestMediaType_Build_Fail_Encoding(t *testing.T) {
	yml := `encoding:
  wiff:
    $ref: #bork`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n MediaType
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(nil, idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestMediaType_Hash(t *testing.T) {

	yml := `schema:
  type: string
example: a thing
examples:
  thing1: 
    description: thing1
  shinyNew:
    description: booyakka!
encoding:
  meaty/chewy:
    style: suave
x-done: for the day!`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n MediaType
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(nil, idxNode.Content[0], idx)

	yml2 := `encoding:
  meaty/chewy:
    style: suave
examples:
  thing1: 
    description: thing1
  shinyNew:
    description: booyakka!
schema:
  type: string
x-done: for the day!
example: a thing`

	var idxNode2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml2), &idxNode2)
	idx2 := index.NewSpecIndex(&idxNode2)

	var n2 MediaType
	_ = low.BuildModel(idxNode2.Content[0], &n2)
	_ = n2.Build(nil, idxNode2.Content[0], idx2)

	// hash
	assert.Equal(t, n.Hash(), n2.Hash())
	assert.Len(t, n.GetExtensions(), 1)
}
