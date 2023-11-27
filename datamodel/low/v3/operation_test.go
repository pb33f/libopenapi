// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
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

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)

	assert.Len(t, n.Tags.Value, 2)
	assert.Equal(t, "building a business", n.Summary.Value)
	assert.Equal(t, "takes hard work", n.Description.Value)
	assert.Equal(t, "some docs", n.ExternalDocs.Value.Description.Value)
	assert.Equal(t, "beefyBeef", n.OperationId.Value)
	assert.Len(t, n.Parameters.Value, 2)
	assert.Equal(t, "a requestBody", n.RequestBody.Value.Description.Value)
	assert.Equal(t, 1, n.Responses.Value.Codes.Len())
	assert.Equal(t, "an OK response", n.Responses.Value.FindResponseByCode("200").Value.Description.Value)
	assert.Equal(t, 1, n.Callbacks.Value.Len())
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

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
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

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
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

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
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

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
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

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
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

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
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

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestOperation_Hash_n_Grab(t *testing.T) {
	yml := `tags:
  - nice
  - rice
summary: a thing
description: another thing
externalDocs:
  url: https://pb33f.io/docs
operationId: sleepyMornings
parameters: 
  - name: parammy
    in: my head
requestBody:
  description: a thing
responses:
  "200":
    description: ok
callbacks: 
  callMe:
    something: blue
deprecated: true
security:
  - lego:
    dont: stand
    or: eat
servers:
  - url: https://pb33f.io
x-mint: sweet`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Operation
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	yml2 := `tags:
  - nice
  - rice
summary: a thing
description: another thing
externalDocs:
  url: https://pb33f.io/docs
operationId: sleepyMornings
parameters: 
  - name: parammy
    in: my head
requestBody:
  description: a thing
responses:
  "200":
    description: ok
callbacks: 
  callMe:
    something: blue
deprecated: true
security:
  - lego:
    dont: stand
    or: eat
servers:
  - url: https://pb33f.io
x-mint: sweet`

	var idxNode2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml2), &idxNode2)
	idx2 := index.NewSpecIndex(&idxNode2)

	var n2 Operation
	_ = low.BuildModel(idxNode2.Content[0], &n2)
	_ = n2.Build(context.Background(), nil, idxNode2.Content[0], idx2)

	// hash
	assert.Equal(t, n.Hash(), n2.Hash())

	// n grab
	assert.Len(t, n.GetTags().Value, 2)
	assert.Equal(t, "a thing", n.GetSummary().Value)
	assert.Equal(t, "another thing", n.GetDescription().Value)
	assert.Equal(t, "https://pb33f.io/docs", n.GetExternalDocs().Value.(*base.ExternalDoc).URL.Value)
	assert.Equal(t, "sleepyMornings", n.GetOperationId().Value)
	assert.Len(t, n.GetParameters().Value, 1)
	assert.Len(t, n.GetSecurity().Value, 1)
	assert.True(t, n.GetDeprecated().Value)
	assert.Len(t, n.GetExtensions(), 1)
	assert.Len(t, n.GetServers().Value.([]low.ValueReference[*Server]), 1)
	assert.Equal(t, 1, n.GetCallbacks().Value.Len())
	assert.Equal(t, 1, n.GetResponses().Value.(*Responses).Codes.Len())
	assert.Nil(t, n.FindSecurityRequirement("I do not exist"))
}

func TestOperation_EmptySecurity(t *testing.T) {
	yml := `
security: []`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Operation
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)

	assert.Len(t, n.Security.Value, 0)
}
