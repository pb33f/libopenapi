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

func TestTag_Build(t *testing.T) {
	yml := `name: a tag
description: a description
externalDocs: 
  url: https://pb33f.io
x-coffee: tasty`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Tag
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "a tag", n.Name.Value)
	assert.Equal(t, "a description", n.Description.Value)
	assert.Equal(t, "https://pb33f.io", n.ExternalDocs.Value.URL.Value)

	var xCoffee string
	_ = n.FindExtension("x-coffee").GetValue().Decode(&xCoffee)

	assert.Equal(t, "tasty", xCoffee)
	assert.Equal(t, 1, orderedmap.Len(n.GetExtensions()))
	assert.NotNil(t, n.GetRootNode())
	assert.Nil(t, n.GetKeyNode())
	assert.NotNil(t, n.GetContext())
	assert.NotNil(t, n.GetIndex())
}

func TestTag_Build_Error(t *testing.T) {
	yml := `name: a tag
description: a description
externalDocs: 
  $ref: #borko`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Tag
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestTag_Hash(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()

	left := `name: melody
description: my princess
externalDocs:
  url: https://pb33f.io
x-b33f: princess`

	right := `name: melody
description: my princess
externalDocs:
  url: https://pb33f.io
x-b33f: princess`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc Tag
	var rDoc Tag
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	assert.Equal(t, lDoc.Hash(), rDoc.Hash())
}

func TestTag_Build_OpenAPI32(t *testing.T) {
	yml := `name: partner
summary: Partner
description: Operations available to the partners network
parent: external
kind: audience
externalDocs: 
  url: https://pb33f.io
  description: Find more info here
x-custom: value`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Tag
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)

	assert.Equal(t, "partner", n.Name.Value)
	assert.Equal(t, "Partner", n.Summary.Value)
	assert.Equal(t, "Operations available to the partners network", n.Description.Value)
	assert.Equal(t, "external", n.Parent.Value)
	assert.Equal(t, "audience", n.Kind.Value)
	assert.Equal(t, "https://pb33f.io", n.ExternalDocs.Value.URL.Value)
	assert.Equal(t, "Find more info here", n.ExternalDocs.Value.Description.Value)

	var xCustom string
	_ = n.FindExtension("x-custom").GetValue().Decode(&xCustom)
	assert.Equal(t, "value", xCustom)

	assert.Equal(t, 1, orderedmap.Len(n.GetExtensions()))
	assert.NotNil(t, n.GetRootNode())
	assert.Nil(t, n.GetKeyNode())
	assert.NotNil(t, n.GetContext())
	assert.NotNil(t, n.GetIndex())
}

func TestTag_Hash_OpenAPI32(t *testing.T) {
	left := `name: partner
summary: Partner
description: Operations available to the partners network
parent: external
kind: audience
externalDocs:
  url: https://pb33f.io
  description: Find more info here
x-custom: value`

	right := `name: partner
summary: Partner
description: Operations available to the partners network
parent: external
kind: audience
externalDocs:
  url: https://pb33f.io
  description: Find more info here
x-custom: value`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc Tag
	var rDoc Tag
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	assert.Equal(t, lDoc.Hash(), rDoc.Hash())

	// test hash difference when fields change
	right2 := `name: partner
summary: Partner API
description: Operations available to the partners network
parent: external
kind: nav
externalDocs:
  url: https://pb33f.io
  description: Find more info here
x-custom: value`

	var rNode2 yaml.Node
	_ = yaml.Unmarshal([]byte(right2), &rNode2)
	var rDoc2 Tag
	_ = low.BuildModel(rNode2.Content[0], &rDoc2)
	_ = rDoc2.Build(context.Background(), nil, rNode2.Content[0], nil)

	assert.NotEqual(t, lDoc.Hash(), rDoc2.Hash())
}
