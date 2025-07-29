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
	"gopkg.in/yaml.v3"
)

func TestServer_Build(t *testing.T) {
	yml := `x-coffee: hot
url: https://pb33f.io
description: high quality software for developers.
variables:
  var1: 
    default: hello
    description: a var
    enum: [one, two]`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Server
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)
	assert.Nil(t, n.GetRootNode())

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.NotNil(t, n.GetRootNode())
	assert.Equal(t, "0c2c833ff3934ac3a0351f56e0ed42e5ffee1d5a7856fb0278857701ef52d6ae",
		low.GenerateHashString(&n))

	assert.Equal(t, "https://pb33f.io", n.URL.Value)
	assert.Equal(t, "high quality software for developers.", n.Description.Value)
	assert.Equal(t, "hello", n.FindVariable("var1").Value.Default.Value)
	assert.Equal(t, "a var", n.FindVariable("var1").Value.Description.Value)

	// test var hash
	s := n.FindVariable("var1")
	assert.Equal(t, "c58f2e9eb6548e9ea9c3bd6ca3ff1f6c5ba850cdb20eb6f362eba5520fc3a011",
		low.GenerateHashString(s.Value))

	assert.Equal(t, 1, orderedmap.Len(n.GetExtensions()))
	assert.NotNil(t, n.GetContext())
	assert.NotNil(t, n.GetIndex())

	// check nodes on variables
	for v := range n.Variables.Value.ValuesFromOldest() {
		assert.NotNil(t, v.Value.GetKeyNode())
		assert.NotNil(t, v.Value.GetRootNode())
		assert.Equal(t, 0, v.Value.GetExtensions().Len())
	}
}

func TestServerWithVariableExtension_Build(t *testing.T) {
	yml := `url: https://pb33f.io
description: high quality software for developers.
variables:
  var1: 
    default: hello
    description: a var
    enum: [one, two]
    x-transforms:
        allowMissing: true`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Server
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)
	assert.Nil(t, n.GetRootNode())

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.NotNil(t, n.GetRootNode())
	assert.Equal(t, "841e49335ea4ae63b677544ae815f9605988f625d24fbaa0a1992ec97f71b00b",
		low.GenerateHashString(&n))

	assert.Equal(t, "https://pb33f.io", n.URL.Value)
	assert.Equal(t, "high quality software for developers.", n.Description.Value)

	variable := n.FindVariable("var1").Value
	assert.Equal(t, "hello", variable.Default.Value)
	assert.Equal(t, "a var", variable.Description.Value)

	var_extensions := variable.Extensions.First()
	assert.Equal(t, "x-transforms", var_extensions.Key().Value)
	variable_pair := variable.Extensions.GetPair(var_extensions.Key())
	assert.Equal(t, "allowMissing", variable_pair.Value.Value.Content[0].Value)
	assert.Equal(t, "true", variable_pair.Value.Value.Content[1].Value)

	// test var hash
	s := n.FindVariable("var1")
	assert.Equal(t, "c58f2e9eb6548e9ea9c3bd6ca3ff1f6c5ba850cdb20eb6f362eba5520fc3a011",
		low.GenerateHashString(s.Value))

	assert.Equal(t, 0, orderedmap.Len(n.GetExtensions()))

	// check nodes on variables
	for v := range n.Variables.Value.ValuesFromOldest() {
		assert.NotNil(t, v.Value.GetKeyNode())
		assert.NotNil(t, v.Value.GetRootNode())
		assert.NotNil(t, v.Value.GetExtensions())
	}
}

func TestServer_Build_NoVars(t *testing.T) {
	yml := `url: https://pb33f.io
description: high quality software for developers.`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Server
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "https://pb33f.io", n.URL.Value)
	assert.Equal(t, "high quality software for developers.", n.Description.Value)
	assert.Equal(t, 0, orderedmap.Len(n.Variables.Value))
}
