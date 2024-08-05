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
	assert.Equal(t, "25535d0a6dd30c609aeae6e08f9eaa82fef49df540fc048fe4adffbce7841c0b",
		low.GenerateHashString(&n))

	assert.Equal(t, "https://pb33f.io", n.URL.Value)
	assert.Equal(t, "high quality software for developers.", n.Description.Value)
	assert.Equal(t, "hello", n.FindVariable("var1").Value.Default.Value)
	assert.Equal(t, "a var", n.FindVariable("var1").Value.Description.Value)

	// test var hash
	s := n.FindVariable("var1")
	assert.Equal(t, "00eef99ee4a7b746be7b4ccdece59c5a96222c6206f846fafed782c9f3f9b46b",
		low.GenerateHashString(s.Value))

	assert.Equal(t, 1, orderedmap.Len(n.GetExtensions()))

	// check nodes on variables
	for k := n.Variables.Value.First(); k != nil; k = k.Next() {
		assert.NotNil(t, k.Value().Value.GetKeyNode())
		assert.NotNil(t, k.Value().Value.GetRootNode())
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
