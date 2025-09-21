// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
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

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)

	assert.NotNil(t, n.GetRootNode())
	assert.Nil(t, n.GetKeyNode())

	var xRock string
	_ = n.FindExtension("x-rock").Value.Decode(&xRock)
	assert.Equal(t, "and roll", xRock)
	assert.Equal(t, "string", n.Schema.Value.Schema().Type.Value.A)
	var example string
	_ = n.Example.Value.Decode(&example)
	assert.Equal(t, "hello", example)

	var whatExample string
	_ = n.FindExample("what").Value.Value.Value.Decode(&whatExample)
	assert.Equal(t, "why?", whatExample)

	var whereExample string
	_ = n.FindExample("where").Value.Value.Value.Decode(&whereExample)
	assert.Equal(t, "there?", whereExample)
	assert.True(t, n.FindPropertyEncoding("chicken").Value.Explode.Value)
	assert.Equal(t, n.GetAllExamples().Len(), 2)
	assert.NotNil(t, n.GetContext())
	assert.NotNil(t, n.GetIndex())
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

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
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

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
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

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestMediaType_Build_Fail_ItemEncoding(t *testing.T) {
	yml := `itemEncoding:
  wiff:
    $ref: #bork`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n MediaType
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestMediaType_Hash(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()

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
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

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
	_ = n2.Build(context.Background(), nil, idxNode2.Content[0], idx2)

	// hash
	assert.Equal(t, n.Hash(), n2.Hash())
	assert.Equal(t, 1, orderedmap.Len(n.GetExtensions()))
}

func TestMediaType_Examples(t *testing.T) {
	yml := `examples:
    pbjBurger:
        summary: A horrible, nutty, sticky mess.
        value:
            name: Peanut And Jelly
            numPatties: 3
    cakeBurger:
        summary: A sickly, sweet, atrocity
        value:
            name: Chocolate Cake Burger
            numPatties: 5`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n MediaType
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	assert.Equal(t, 2, orderedmap.Len(n.Examples.Value))
}

func TestMediaType_Examples_NotFromSchema(t *testing.T) {
	yml := `schema:
  type: string
  examples:
    - example 1
    - example 2
    - example 3`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n MediaType
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	assert.Equal(t, 0, orderedmap.Len(n.Examples.Value))
}
