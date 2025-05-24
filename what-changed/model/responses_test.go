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

func TestCompareResponses_V2(t *testing.T) {
	left := `x-coffee: roasty
default:
  schema:
    type: string
200:
  description: OK response
  schema:
    type: string
404:
  description: not found response
  schema:
    type: string`

	right := left

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Responses
	var rDoc v2.Responses
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	extChanges := CompareResponses(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareResponses_V2_ModifyCode(t *testing.T) {
	left := `200:
  description: OK response
  schema:
    type: int
404:
  description: not found response
  schema:
    type: int
x-ting: tang`

	right := `200:
  description: OK response
  schema:
    type: string
404:
  description: not found response
  schema:
    type: string
x-ting: tang`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Responses
	var rDoc v2.Responses
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	extChanges := CompareResponses(&lDoc, &rDoc)
	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 2)
	assert.Equal(t, 2, extChanges.TotalBreakingChanges())
	assert.Equal(t, Modified, extChanges.ResponseChanges["404"].SchemaChanges.Changes[0].ChangeType)
	assert.Equal(t, Modified, extChanges.ResponseChanges["200"].SchemaChanges.Changes[0].ChangeType)
}

func TestCompareResponses_V2_AddSchema(t *testing.T) {
	left := `x-hack: code
200:
  description: OK response
  schema:
    type: int
x-apple: pie
404:
  description: not found response
  schema:
    type: int`

	right := `404:
  description: not found response
  schema:
    type: int
x-hack: all the code
200:
  description: OK response
x-apple: pie`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Responses
	var rDoc v2.Responses
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	extChanges := CompareResponses(&rDoc, &lDoc)
	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 2)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, extChanges.ResponseChanges["200"].Changes[0].ChangeType)
}

func TestCompareResponses_V2_RemoveSchema(t *testing.T) {
	left := `200:
  description: OK response
  schema:
    type: int
404:
  description: not found response
  schema:
    type: int`

	right := `200:
  description: OK response
404:
  description: not found response
  schema:
    type: int`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Responses
	var rDoc v2.Responses
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	extChanges := CompareResponses(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, extChanges.ResponseChanges["200"].Changes[0].ChangeType)
}

func TestCompareResponses_V2_AddDefault(t *testing.T) {
	left := `200:
  description: OK response
  schema:
    type: int`

	right := `200:
  description: OK response
  schema:
    type: int
default:
  description: not found response
  schema:
    type: int`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Responses
	var rDoc v2.Responses
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	extChanges := CompareResponses(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, extChanges.Changes[0].ChangeType)
}

func TestCompareResponses_V2_RemoveDefault(t *testing.T) {
	left := `200:
  description: OK response
  schema:
    type: int`

	right := `200:
  description: OK response
  schema:
    type: int
default:
  description: not found response
  schema:
    type: int`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Responses
	var rDoc v2.Responses
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	extChanges := CompareResponses(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, extChanges.Changes[0].ChangeType)
}

func TestCompareResponses_V2_ModifyDefault(t *testing.T) {
	left := `200:
  description: OK response
  schema:
    type: int
default:
  description: not found response
  schema:
    type: string`

	right := `200:
  description: OK response
  schema:
    type: int
default:
  description: I am a potato.
  schema:
    type: int`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Responses
	var rDoc v2.Responses
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	extChanges := CompareResponses(&lDoc, &rDoc)
	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 2)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
}

func TestCompareResponses_V3(t *testing.T) {
	left := `x-coffee: roasty
default:
  description: a thing
200:
  description: OK response
404:
  description: not found response`

	right := left

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Responses
	var rDoc v3.Responses
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	extChanges := CompareResponses(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareResponses_V3_ModifyCode(t *testing.T) {
	left := `x-coffee: roasty
default:
  description: a thing
200:
  description: OK response
404:
  description: not found response`

	right := `404:
  description: not found response but new!
default:
  description: a thing that changed
x-coffee: yum
200:
  description: blast off`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Responses
	var rDoc v3.Responses
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	extChanges := CompareResponses(&lDoc, &rDoc)
	assert.Equal(t, 4, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 4)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
}

func TestCompareResponses_V3_AddDefault(t *testing.T) {
	left := `x-coffee: roasty
200:
  description: OK response
404:
  description: not found response`

	right := `x-coffee: roasty
default:
  description: a thing
200:
  description: OK response
404:
  description: not found response`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Responses
	var rDoc v3.Responses
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	extChanges := CompareResponses(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, v3.DefaultLabel, extChanges.Changes[0].Property)
}

func TestCompareResponses_V3_RemoveDefault(t *testing.T) {
	left := `x-coffee: roasty
200:
  description: OK response
404:
  description: not found response`

	right := `x-coffee: roasty
default:
  description: a thing
200:
  description: OK response
404:
  description: not found response`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Responses
	var rDoc v3.Responses
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	extChanges := CompareResponses(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, v3.DefaultLabel, extChanges.Changes[0].Property)
}

func TestCompareResponses_V3_ModifyDefault(t *testing.T) {
	left := `x-coffee: roasty
200:
  description: OK response
404:
  description: not found response
default:
  description: a thing`

	right := `x-coffee: roasty
default:
  description: a thing that changed!
200:
  description: OK response
404:
  description: not found response`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Responses
	var rDoc v3.Responses
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	extChanges := CompareResponses(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, v3.DescriptionLabel, extChanges.DefaultChanges.Changes[0].Property)
}

func TestCompareResponses_V3_AddRemoveMediaType(t *testing.T) {
	left := `200:
  content:
    application/json:
      schema:
        type: int`

	right := `200:
  content:
    application/xml:
      schema:
        type: int`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Responses
	var rDoc v3.Responses
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	extChanges := CompareResponses(&lDoc, &rDoc)
	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 2)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
}
