// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"testing"

	"github.com/pkg-base/libopenapi/datamodel/low"
	"github.com/pkg-base/libopenapi/index"
	"github.com/pkg-base/libopenapi/orderedmap"
	"github.com/pkg-base/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExample_Build_Success_NoValue(t *testing.T) {
	yml := `summary: hot
description: cakes
x-cake: hot`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	var n Example
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "hot", n.Summary.Value)
	assert.Equal(t, "cakes", n.Description.Value)
	assert.Nil(t, n.Value.Value)

	var xCake string
	_ = n.FindExtension("x-cake").Value.Decode(&xCake)
	assert.Equal(t, "hot", xCake)
	assert.NotNil(t, n.GetRootNode())
	assert.Nil(t, n.GetKeyNode())
	assert.NotNil(t, n.GetContext())
	assert.NotNil(t, n.GetIndex())
}

func TestExample_Build_Success_Simple(t *testing.T) {
	yml := `summary: hot
description: cakes
value: a string example
x-cake: hot`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	var n Example
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "hot", n.Summary.Value)
	assert.Equal(t, "cakes", n.Description.Value)

	var example string
	err = n.Value.Value.Decode(&example)
	require.NoError(t, err)
	assert.Equal(t, "a string example", example)

	var xCake string
	_ = n.FindExtension("x-cake").Value.Decode(&xCake)
	assert.Equal(t, "hot", xCake)
}

func TestExample_Build_Success_Object(t *testing.T) {
	yml := `summary: hot
description: cakes
value:
  pizza: oven
  yummy: pizza`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	var n Example
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "hot", n.Summary.Value)
	assert.Equal(t, "cakes", n.Description.Value)

	var m map[string]interface{}
	err = n.Value.Value.Decode(&m)
	require.NoError(t, err)

	assert.Equal(t, "oven", m["pizza"])
	assert.Equal(t, "pizza", m["yummy"])
}

func TestExample_Build_Success_Array(t *testing.T) {
	yml := `summary: hot
description: cakes
value:
  - wow
  - such array`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	var n Example
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "hot", n.Summary.Value)
	assert.Equal(t, "cakes", n.Description.Value)

	var a []any
	err = n.Value.Value.Decode(&a)
	require.NoError(t, err)

	assert.Equal(t, "wow", a[0])
	assert.Equal(t, "such array", a[1])
}

func TestExample_Build_Success_MergeNode(t *testing.T) {
	yml := `x-things: &things
  summary: hot
  description: cakes
  value:
    - wow
    - such array
<<: *things`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	var n Example
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "hot", n.Summary.Value)
	assert.Equal(t, "cakes", n.Description.Value)

	var a []any
	err = n.Value.GetValue().Decode(&a)
	require.NoError(t, err)

	assert.Equal(t, "wow", a[0])
	assert.Equal(t, "such array", a[1])
}

func TestExample_Hash(t *testing.T) {
	left := `summary: hot
description: cakes
x-burger: nice
externalValue: cake
value:
  pizza: oven
  yummy: pizza`

	right := `externalValue: cake
summary: hot
value:
  pizza: oven
  yummy: pizza
description: cakes
x-burger: nice`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc Example
	var rDoc Example
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	assert.Equal(t, lDoc.Hash(), rDoc.Hash())
	assert.Equal(t, 1, orderedmap.Len(lDoc.GetExtensions()))
}

func TestExample_Build_Success_Ref(t *testing.T) {
	yml := `$ref: "#/responses/abc"`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	var n Example
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)

	assert.True(t, n.IsReference())
	assert.Equal(t, "#/responses/abc", n.GetReference())
}
