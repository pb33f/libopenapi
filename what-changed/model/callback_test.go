// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestCompareCallback(t *testing.T) {

	left := `'{$request.query.queryUrl}':
    post:
      requestBody:
        description: Callback payload
        content: 
          'application/json':
            schema:
              type: int
      responses:
        '200':
          description: callback successfully processed`

	right := left

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Callback
	var rDoc v3.Callback
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(nil, lNode.Content[0], nil)
	_ = rDoc.Build(nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareCallback(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareCallback_Add(t *testing.T) {

	left := `'{$request.query.queryUrl}':
    post:
      requestBody:
        description: Callback payload
        content: 
          'application/json':
            schema:
              type: int
      responses:
        '200':
          description: callback successfully processed`

	right := `'{$request.query.queryUrl}':
    post:
      requestBody:
        description: Callback payload
        content: 
          'application/json':
            schema:
              type: int
      responses:
        '200':
          description: callback successfully processed
'slippers':
    post:
      description: toasty toes`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Callback
	var rDoc v3.Callback
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(nil, lNode.Content[0], nil)
	_ = rDoc.Build(nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareCallback(&lDoc, &rDoc)
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, extChanges.Changes[0].ChangeType)
	assert.Equal(t, "slippers", extChanges.Changes[0].Property)
}

func TestCompareCallback_Modify(t *testing.T) {

	left := `x-pizza: tasty
'{$request.query.queryUrl}':
    post:
      requestBody:
        description: Callback payload
        content: 
          'application/json':
            schema:
              type: int
      responses:
        '200':
          description: callback successfully processed`

	right := `x-pizza: cold
'{$request.query.queryUrl}':
    get:
      description: a nice new thing, for the things.
    post:
      requestBody:
        description: Callback payload
        content: 
          'application/json':
            schema:
              type: int
      responses:
        '200':
          description: callback successfully processed`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Callback
	var rDoc v3.Callback
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(nil, lNode.Content[0], nil)
	_ = rDoc.Build(nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareCallback(&lDoc, &rDoc)
	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 2)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, PropertyAdded, extChanges.ExpressionChanges["{$request.query.queryUrl}"].Changes[0].ChangeType)
	assert.Equal(t, v3.GetLabel, extChanges.ExpressionChanges["{$request.query.queryUrl}"].Changes[0].Property)
}

func TestCompareCallback_Remove(t *testing.T) {

	left := `'{$request.query.queryUrl}':
    post:
      requestBody:
        description: Callback payload
        content: 
          'application/json':
            schema:
              type: int
      responses:
        '200':
          description: callback successfully processed`

	right := `'{$request.query.queryUrl}':
    post:
      requestBody:
        description: Callback payload
        content: 
          'application/json':
            schema:
              type: int
      responses:
        '200':
          description: callback successfully processed
'slippers':
    post:
      description: toasty toes`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Callback
	var rDoc v3.Callback
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(nil, lNode.Content[0], nil)
	_ = rDoc.Build(nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareCallback(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, "slippers", extChanges.Changes[0].Property)
}
