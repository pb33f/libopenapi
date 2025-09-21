// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestCompareMediaTypes(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()

	left := `schema:
  type: string
example: tasty herbs in the morning
examples:
  exampleOne:
    value: yummy coffee
encoding:
  contentType: application/json`

	right := `schema:
  type: string
example: tasty herbs in the morning
examples:
  exampleOne:
    value: yummy coffee
encoding:
  contentType: application/json`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.MediaType
	var rDoc v3.MediaType
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareMediaTypes(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareMediaTypes_Modify(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `schema:
  type: string
example: tasty herbs in the morning
examples:
  exampleOne:
    value: yummy coffee
encoding:
  contentType: application/json
itemSchema:
  type: string
itemEncoding:
 - contentType: application/json`

	right := `schema:
  type: string
example: smoke and a pancake?
examples:
  exampleOne:
    value: yummy coffee
encoding:
  contentType: application/json
itemSchema:
  type: int
itemEncoding:
  contentType: fish/paste`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.MediaType
	var rDoc v3.MediaType
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareMediaTypes(&lDoc, &rDoc)
	assert.NotNil(t, extChanges)
	assert.Equal(t, 3, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 3)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, Modified, extChanges.Changes[0].ChangeType)
	assert.Equal(t, v3.ExampleLabel, extChanges.Changes[0].Property)
}

func TestCompareMediaTypes_Modify_Examples(t *testing.T) {
	left := `schema:
  type: string
example:
  smoke: and a pancake?`

	right := `schema:
  type: string
example:
  smoke: and a pancake?
  pipe: pipe and a crepe?`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.MediaType
	var rDoc v3.MediaType
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareMediaTypes(&lDoc, &rDoc)
	assert.NotNil(t, extChanges)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, Modified, extChanges.Changes[0].ChangeType)
	assert.Equal(t, v3.ExampleLabel, extChanges.Changes[0].Property)
}

func TestCompareMediaTypes_ExampleChangedToMap(t *testing.T) {
	left := `schema:
  type: string`

	right := `schema:
  type: string
example:
  smoke: and a pancake?
  pipe: pipe and a crepe?`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.MediaType
	var rDoc v3.MediaType
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareMediaTypes(&lDoc, &rDoc)
	assert.NotNil(t, extChanges)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, PropertyAdded, extChanges.Changes[0].ChangeType)
	assert.Equal(t, v3.ExampleLabel, extChanges.Changes[0].Property)
}

func TestCompareMediaTypes_ExampleMapRemoved(t *testing.T) {
	left := `schema:
  type: string`

	right := `schema:
  type: string
example:
  smoke: and a pancake?
  pipe: pipe and a crepe?`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.MediaType
	var rDoc v3.MediaType
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareMediaTypes(&rDoc, &lDoc)
	assert.NotNil(t, extChanges)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, PropertyRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, v3.ExampleLabel, extChanges.Changes[0].Property)
}

func TestCompareMediaTypes_AddSchema(t *testing.T) {
	left := `example: tasty herbs in the morning
examples:
  exampleOne:
    value: yummy coffee
encoding:
  contentType: application/json`

	right := `schema:
  type: string
example: tasty herbs in the morning
examples:
  exampleOne:
    value: yummy coffee
encoding:
  contentType: application/json`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.MediaType
	var rDoc v3.MediaType
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareMediaTypes(&lDoc, &rDoc)
	assert.NotNil(t, extChanges)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, extChanges.Changes[0].ChangeType)
	assert.Equal(t, v3.SchemaLabel, extChanges.Changes[0].Property)
}

func TestCompareMediaTypes_RemoveSchema(t *testing.T) {
	left := `example: tasty herbs in the morning
examples:
  exampleOne:
    value: yummy coffee
encoding:
  contentType: application/json`

	right := `schema:
  type: string
example: tasty herbs in the morning
examples:
  exampleOne:
    value: yummy coffee
encoding:
  contentType: application/json`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.MediaType
	var rDoc v3.MediaType
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareMediaTypes(&rDoc, &lDoc)
	assert.NotNil(t, extChanges)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, v3.SchemaLabel, extChanges.Changes[0].Property)
}

func TestCompareMediaTypes_ModifyObjects(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()

	left := `schema:
  type: string
example: tasty herbs in the morning
examples:
  exampleOne:
    value: yummy coffee
itemEncoding:
  cupOfTea:
    contentType: milk/hotwater
encoding:
  something: 
    contentType: application/json
x-tea: cake`

	right := `schema:
  type: int
example: smoke and a pancake?
examples:
  exampleOne:
    value: yummy coffee is great!
itemEncoding:
  cupOfTea:
    contentType: milk/hotwater/sugar
encoding:
  something:
    contentType: application/xml
x-tea: cup`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.MediaType
	var rDoc v3.MediaType
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareMediaTypes(&lDoc, &rDoc)
	assert.NotNil(t, extChanges)
	assert.Equal(t, 6, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 6)
	assert.Equal(t, 3, extChanges.TotalBreakingChanges())
}
