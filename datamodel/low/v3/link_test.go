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

func TestLink_Build(t *testing.T) {
	cleanHashCacheForTest(t)

	yml := `operationRef: '#/someref'
operationId: someId
parameters:
  param1: something
  param2: somethingElse
requestBody: somebody
description: this is a link object.
server:
  url: https://pb33f.io
x-linky: slinky  
`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Link
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.NotNil(t, n.GetRootNode())
	assert.Nil(t, n.GetKeyNode())
	assert.Equal(t, "#/someref", n.OperationRef.Value)
	assert.Equal(t, "someId", n.OperationId.Value)
	assert.Equal(t, "this is a link object.", n.Description.Value)

	var xLinky string
	_ = n.FindExtension("x-linky").Value.Decode(&xLinky)
	assert.Equal(t, "slinky", xLinky)

	param1 := n.FindParameter("param1")
	assert.Equal(t, "something", param1.Value)
	param2 := n.FindParameter("param2")
	assert.Equal(t, "somethingElse", param2.Value)

	assert.NotNil(t, n.Server.Value)
	assert.Equal(t, "https://pb33f.io", n.Server.Value.URL.Value)
	assert.Equal(t, 1, orderedmap.Len(n.GetExtensions()))
	assert.NotNil(t, n.GetContext())
	assert.NotNil(t, n.GetIndex())
}

func TestLink_Build_Fail(t *testing.T) {
	cleanHashCacheForTest(t)
	yml := `operationRef: '#/someref'
operationId: someId
parameters:
  param1: something
  param2: somethingElse
requestBody: somebody
description: this is a link object.
server:
  $ref: #bork`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Link
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestLink_Hash(t *testing.T) {
	cleanHashCacheForTest(t)
	yml := `operationRef: something
operationId: someWhere
parameters:
  fried: sausage
  bacon: eggs
requestBody: burgers please
description: a useless and invalid link
server:
  url: https://pb33f.io
x-mcdonalds: bigmac`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Link
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	yml2 := `parameters:
  bacon: eggs
  fried: sausage
requestBody: burgers please
operationId: someWhere
operationRef: something
description: a useless and invalid link
x-mcdonalds: bigmac
server:
  url: https://pb33f.io`

	var idxNode2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml2), &idxNode2)
	idx2 := index.NewSpecIndex(&idxNode2)

	var n2 Link
	_ = low.BuildModel(idxNode2.Content[0], &n2)
	_ = n2.Build(context.Background(), nil, idxNode2.Content[0], idx2)

	// hash
	assert.Equal(t, n.Hash(), n2.Hash())
}
