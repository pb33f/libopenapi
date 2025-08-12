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
