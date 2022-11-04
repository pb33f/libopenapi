// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestLink_Build(t *testing.T) {

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

	err = n.Build(idxNode.Content[0], idx)
	assert.NoError(t, err)

	assert.Equal(t, "#/someref", n.OperationRef.Value)
	assert.Equal(t, "someId", n.OperationId.Value)
	assert.Equal(t, "this is a link object.", n.Description.Value)

	ext := n.FindExtension("x-linky")
	assert.NotNil(t, ext)
	assert.Equal(t, "slinky", ext.Value)

	param1 := n.FindParameter("param1")
	assert.Equal(t, "something", param1.Value)
	param2 := n.FindParameter("param2")
	assert.Equal(t, "somethingElse", param2.Value)

	assert.NotNil(t, n.Server.Value)
	assert.Equal(t, "https://pb33f.io", n.Server.Value.URL.Value)

}

func TestLink_Build_Fail(t *testing.T) {

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

	err = n.Build(idxNode.Content[0], idx)
	assert.Error(t, err)

}
