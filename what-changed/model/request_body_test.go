// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"context"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestCompareRequestBodies(t *testing.T) {

	left := `description: something
required: true
content:
  application/json:
    schema:
      type: int`

	right := `description: something
required: true
content:
  application/json:
    schema:
      type: int`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.RequestBody
	var rDoc v3.RequestBody
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareRequestBodies(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareRequestBodies_Modified(t *testing.T) {

	left := `description: something
required: true
x-pizza: thin
content:
  application/json:
    schema:
      type: int`

	right := `x-pizza: oven
description: nothing
required: false
content:
  application/json:
    schema:
      type: string`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.RequestBody
	var rDoc v3.RequestBody
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareRequestBodies(&lDoc, &rDoc)

	assert.Equal(t, 4, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 4)
	assert.Equal(t, 2, extChanges.TotalBreakingChanges())
}
