// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestOperation_Build_ExternalDocs(t *testing.T) {
	yml := `externalDocs:
  $ref: break`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Operation
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestOperation_Build_Params(t *testing.T) {
	yml := `parameters:
  $ref: break`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Operation
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestOperation_Build_Responses(t *testing.T) {
	yml := `responses:
  $ref: break`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Operation
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestOperation_Build_Security(t *testing.T) {
	yml := `security:
  $ref: break`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
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
  - hat
summary: a nice day
description: a nice day for a walk in the park
externalDocs:
  url: https://pb33f.io
operationId: theMagicCastle
consumes:
  - burgers
  - beer
produces:
  - burps
  - farts
parameters:
  - in: head
    name: drinks
deprecated: true
security:
  - winter:
    - cold
    - snow
schemes:
  - ws
  - https
responses:
  200:
    description: fruity
x-smoke: not for a while`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Operation
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	yml2 := `summary: a nice day
tags:
  - hat
  - nice
description: a nice day for a walk in the park
externalDocs:
  url: https://pb33f.io
consumes:
  - beer
  - burgers
schemes:
  - https
  - ws
x-smoke: not for a while
produces:
  - farts
  - burps
operationId: theMagicCastle
parameters:
  - in: head
    name: drinks
deprecated: true
responses:
  200:
    description: fruity
security:
  - winter:
    - snow
    - cold`

	var idxNode2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml2), &idxNode2)
	idx2 := index.NewSpecIndex(&idxNode2)

	var n2 Operation
	_ = low.BuildModel(idxNode2.Content[0], &n2)
	_ = n2.Build(context.Background(), nil, idxNode2.Content[0], idx2)

	// hash
	assert.Equal(t, n.Hash(), n2.Hash())

	// and grab
	assert.Equal(t, "a nice day", n.GetSummary().Value)
	assert.Equal(t, "a nice day for a walk in the park", n.GetDescription().Value)
	assert.Len(t, n.GetTags().Value, 2)
	assert.Equal(t, "https://pb33f.io", n.GetExternalDocs().Value.(*base.ExternalDoc).URL.Value)
	assert.Len(t, n.GetConsumes().Value, 2)
	assert.Len(t, n.GetSchemes().Value, 2)
	assert.Len(t, n.GetProduces().Value, 2)
	assert.Equal(t, "theMagicCastle", n.GetOperationId().Value)
	assert.Len(t, n.GetParameters().Value, 1)
	assert.True(t, n.GetDeprecated().Value)
	assert.Equal(t, 1, orderedmap.Len(n.GetResponses().Value.(*Responses).Codes))
	assert.Len(t, n.GetSecurity().Value, 1)
	assert.Equal(t, 1, orderedmap.Len(n.GetExtensions()))
}
