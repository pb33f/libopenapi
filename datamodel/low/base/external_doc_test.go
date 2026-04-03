// Copyright 2022-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestExternalDoc_FindExtension(t *testing.T) {
	yml := `x-fish: cake`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n ExternalDoc
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)

	var xFish string
	_ = n.FindExtension("x-fish").Value.Decode(&xFish)

	assert.Equal(t, "cake", xFish)
	assert.NotNil(t, n.GetRootNode())
	assert.Nil(t, n.GetKeyNode())
	assert.NotNil(t, n.GetContext())
	assert.NotNil(t, n.GetIndex())
}

func TestExternalDoc_Build(t *testing.T) {
	yml := `url: https://pb33f.io
description: the ranch
x-b33f: princess`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n ExternalDoc
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "https://pb33f.io", n.URL.Value)
	assert.Equal(t, "the ranch", n.Description.Value)

	var xB33f string
	_ = n.FindExtension("x-b33f").Value.Decode(&xB33f)
	assert.Equal(t, "princess", xB33f)
}

func TestExternalDoc_Hash(t *testing.T) {
	left := `url: https://pb33f.io
description: the ranch
x-b33f: princess`

	right := `url: https://pb33f.io
x-b33f: princess
description: the ranch`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc ExternalDoc
	var rDoc ExternalDoc
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	assert.Equal(t, lDoc.Hash(), rDoc.Hash())
	assert.Equal(t, 1, orderedmap.Len(lDoc.GetExtensions()))
}

func TestExternalDoc_Build_NilRoot(t *testing.T) {
	var n ExternalDoc
	err := n.Build(context.Background(), nil, nil, nil)
	assert.NoError(t, err)
	assert.Nil(t, n.GetRootNode())
	assert.Nil(t, n.GetExtensions())
}

func TestExternalDoc_Build_ScalarRoot(t *testing.T) {
	var scalar yaml.Node
	_ = yaml.Unmarshal([]byte("hello"), &scalar)

	var n ExternalDoc
	err := low.BuildModel(scalar.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, scalar.Content[0], nil)
	assert.NoError(t, err)

	nodes := n.GetNodes()
	assert.Len(t, nodes[scalar.Content[0].Line], 1)
	assert.Equal(t, "hello", nodes[scalar.Content[0].Line][0].Value)
}
