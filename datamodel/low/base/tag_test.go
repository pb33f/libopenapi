// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
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
	assert.Equal(t, "tasty", n.FindExtension("x-coffee").Value)
	assert.Len(t, n.GetExtensions(), 1)

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
