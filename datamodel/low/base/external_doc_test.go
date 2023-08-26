// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestExternalDoc_FindExtension(t *testing.T) {
	t.Parallel()
	yml := `x-fish: cake`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n ExternalDoc
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(nil, idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "cake", n.FindExtension("x-fish").Value)

}

func TestExternalDoc_Build(t *testing.T) {
	t.Parallel()
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

	err = n.Build(nil, idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "https://pb33f.io", n.URL.Value)
	assert.Equal(t, "the ranch", n.Description.Value)
	ext := n.FindExtension("x-b33f")
	assert.NotNil(t, ext)
	assert.Equal(t, "princess", ext.Value)

}

func TestExternalDoc_Hash(t *testing.T) {
	t.Parallel()
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
	_ = lDoc.Build(nil, lNode.Content[0], nil)
	_ = rDoc.Build(nil, rNode.Content[0], nil)

	assert.Equal(t, lDoc.Hash(), rDoc.Hash())
	assert.Len(t, lDoc.GetExtensions(), 1)
}
