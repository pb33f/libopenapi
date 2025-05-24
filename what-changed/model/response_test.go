// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/v2"
	"github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestCompareResponse_V2(t *testing.T) {
	left := `description: response
schema:
  type: string
headers:
  thing:
    description: a header
examples:
  bam: alam
server:
  url: https://example.com
x-toot: poot`
	right := left

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Response
	var rDoc v2.Response
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	extChanges := CompareResponse(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareResponse_V2_Modify(t *testing.T) {
	left := `description: response
schema:
  type: string
headers:
  thing:
    description: a header
examples:
  bam: alam`

	right := `description: response changed
schema:
  type: int
headers:
  thing:
    description: a header changed
examples:
  bam: alabama
x-toot: poot`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Response
	var rDoc v2.Response
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	extChanges := CompareResponse(&lDoc, &rDoc)
	assert.Equal(t, 5, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 5)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
}

func TestCompareResponse_V2_Add(t *testing.T) {
	left := `description: response
headers:
  thing:
    description: a header`

	right := `description: response
schema:
  type: int
headers:
  thing:
    description: a header
examples:
  bam: alam`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Response
	var rDoc v2.Response
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	extChanges := CompareResponse(&lDoc, &rDoc)
	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 2)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
}

func TestCompareResponse_V2_Remove(t *testing.T) {
	left := `description: response
headers:
  thing:
    description: a header`

	right := `description: response
schema:
  type: int
headers:
  thing:
    description: a header
examples:
  bam: alabama`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Response
	var rDoc v2.Response
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	extChanges := CompareResponse(&rDoc, &lDoc)
	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 2)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
}

func TestCompareResponse_V3(t *testing.T) {
	left := `description: response
content:
  application/json:
    schema:
      type: string
headers:
  thing:
    description: a header
links:
  aLink:
    operationId: oneTwoThree
x-toot: poot`
	right := left

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Response
	var rDoc v3.Response
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	extChanges := CompareResponse(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareResponse_V3_Modify(t *testing.T) {
	left := `description: response
content:
  application/json:
    schema:
      type: string
headers:
  thing:
    description: a header
links:
  aLink:
    operationId: oneTwoThree
server:
  url: https://pb33f.io
x-toot: poot`

	right := `links:
  aLink:
    operationId: oneTwoThreeFour
content:
  application/json:
    schema:
      type: int
description: response change
headers:
  thing:
    description: a header changed
x-toot: pooty`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Response
	var rDoc v3.Response
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	extChanges := CompareResponse(&lDoc, &rDoc)

	assert.Equal(t, 5, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 5)
	assert.Equal(t, 2, extChanges.TotalBreakingChanges())
}
