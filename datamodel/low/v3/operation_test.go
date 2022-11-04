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

func TestOperation_Build(t *testing.T) {

	yml := `tags:
 - meddy
 - maddy
summary: building a business
description: takes hard work
externalDocs:
  description: some docs
operationId: beefyBeef
parameters:
  - name: pizza
  - name: cake
requestBody:
  description: a requestBody
responses:
  "200":
    description: an OK response
callbacks:
  niceCallback:
    ohISee:
      description: a nice callback
deprecated: true
security:
  - books:
    - read:books
    - write:books
servers:
 - url: https://pb33f.io`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Operation
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.NoError(t, err)

	assert.Len(t, n.Tags.Value, 2)
	assert.Equal(t, "building a business", n.Summary.Value)
	assert.Equal(t, "takes hard work", n.Description.Value)
	assert.Equal(t, "some docs", n.ExternalDocs.Value.Description.Value)
	assert.Equal(t, "beefyBeef", n.OperationId.Value)
	assert.Len(t, n.Parameters.Value, 2)
	assert.Equal(t, "a requestBody", n.RequestBody.Value.Description.Value)
	assert.Len(t, n.Responses.Value.Codes, 1)
	assert.Equal(t, "an OK response", n.Responses.Value.FindResponseByCode("200").Value.Description.Value)
	assert.Len(t, n.Callbacks.Value, 1)
	assert.Equal(t, "a nice callback",
		n.FindCallback("niceCallback").Value.FindExpression("ohISee").Value.Description.Value)
	assert.True(t, n.Deprecated.Value)
	assert.Len(t, n.Security.Value, 1)
	assert.Len(t, n.FindSecurityRequirement("books"), 2)
	assert.Equal(t, "read:books", n.FindSecurityRequirement("books")[0].Value)
	assert.Equal(t, "write:books", n.FindSecurityRequirement("books")[1].Value)
	assert.Len(t, n.Servers.Value, 1)
	assert.Equal(t, "https://pb33f.io", n.Servers.Value[0].Value.URL.Value)
}

func TestOperation_Build_FailDocs(t *testing.T) {

	yml := `externalDocs:
  $ref: #borked`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Operation
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestOperation_Build_FailParams(t *testing.T) {

	yml := `parameters:
  $ref: #borked`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Operation
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestOperation_Build_FailRequestBody(t *testing.T) {

	yml := `requestBody:
  $ref: #borked`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Operation
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestOperation_Build_FailResponses(t *testing.T) {

	yml := `responses:
  $ref: #borked`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Operation
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestOperation_Build_FailCallbacks(t *testing.T) {

	yml := `callbacks:
  $ref: #borked`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Operation
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestOperation_Build_FailSecurity(t *testing.T) {

	yml := `security:
  $ref: #borked`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Operation
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestOperation_Build_FailServers(t *testing.T) {

	yml := `servers:
  $ref: #borked`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Operation
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.Error(t, err)
}
