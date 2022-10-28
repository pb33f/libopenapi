// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	v2 "github.com/pb33f/libopenapi/datamodel/low/v2"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestCompareResponses_V2(t *testing.T) {

	left := `default:
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
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

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
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	extChanges := CompareResponses(&lDoc, &rDoc)
	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Equal(t, 2, extChanges.TotalBreakingChanges())
	assert.Equal(t, Modified, extChanges.ResponseChanges["404"].SchemaChanges.Changes[0].ChangeType)
	assert.Equal(t, Modified, extChanges.ResponseChanges["200"].SchemaChanges.Changes[0].ChangeType)
}

func TestCompareResponses_V2_AddSchema(t *testing.T) {
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
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	extChanges := CompareResponses(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
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
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	extChanges := CompareResponses(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
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
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	extChanges := CompareResponses(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
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
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	extChanges := CompareResponses(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, extChanges.Changes[0].ChangeType)
}
