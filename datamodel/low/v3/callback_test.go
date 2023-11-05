// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestCallback_Build_Success(t *testing.T) {

	yml := `'{$request.query.queryUrl}':
    post:
      requestBody:
        description: Callback payload
        content: 
          'application/json':
            schema:
              type: string
      responses:
        '200':
          description: callback successfully processed`

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	var n Callback
	err := low.BuildModel(rootNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, rootNode.Content[0], nil)
	assert.NoError(t, err)

	assert.Len(t, n.Expression.Value, 1)

}

func TestCallback_Build_Error(t *testing.T) {

	// first we need an index.
	doc := `components:
  schemas:
    Something:
      description: this is something
      type: string`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(doc), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	yml := `'{$request.query.queryUrl}':
  $ref: '#/does/not/exist/and/invalid'`

	var rootNode yaml.Node
	mErr = yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	var n Callback
	err := low.BuildModel(rootNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, rootNode.Content[0], idx)
	assert.Error(t, err)

}

func TestCallback_Build_Using_InlineRef(t *testing.T) {

	// first we need an index.
	doc := `components:
  schemas:
    Something:
      description: this is something
      type: string`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(doc), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	yml := `'{$request.query.queryUrl}':
    post:
      requestBody:
        $ref: '#/components/schemas/Something'
      responses:
        '200':
          description: callback successfully processed`

	var rootNode yaml.Node
	mErr = yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	var n Callback
	err := low.BuildModel(rootNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, rootNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Len(t, n.Expression.Value, 1)

	exp := n.FindExpression("{$request.query.queryUrl}")
	assert.NotNil(t, exp.Value)
	assert.NotNil(t, exp.Value.Post.Value)
	assert.Equal(t, "this is something", exp.Value.Post.Value.RequestBody.Value.Description.Value)

}

func TestCallback_Hash(t *testing.T) {

	yml := `x-seed: grow
pizza:
  description: cheesy
burgers:
  description: tasty!
beer:
  description: fantastic
x-weed: loved`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Callback
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	yml2 := `burgers:
  description: tasty!
pizza:
  description: cheesy
x-weed: loved
x-seed: grow
beer:
  description: fantastic
`
	var idxNode2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml2), &idxNode2)
	idx2 := index.NewSpecIndex(&idxNode2)

	var n2 Callback
	_ = low.BuildModel(idxNode2.Content[0], &n2)
	_ = n2.Build(context.Background(), nil, idxNode2.Content[0], idx2)

	// hash
	assert.Equal(t, n.Hash(), n2.Hash())
	assert.Len(t, n.GetExtensions(), 2)

}
