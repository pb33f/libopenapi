// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/what-changed/core"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestCompareMediaTypes(t *testing.T) {

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
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareMediaTypes(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareMediaTypes_Modify(t *testing.T) {

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
example: smoke and a pancake?
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
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareMediaTypes(&lDoc, &rDoc)
	assert.NotNil(t, extChanges)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, core.Modified, extChanges.Changes[0].ChangeType)
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
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareMediaTypes(&lDoc, &rDoc)
	assert.NotNil(t, extChanges)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, core.ObjectAdded, extChanges.Changes[0].ChangeType)
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
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareMediaTypes(&rDoc, &lDoc)
	assert.NotNil(t, extChanges)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, core.ObjectRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, v3.SchemaLabel, extChanges.Changes[0].Property)
}

func TestCompareMediaTypes_ModifyObjects(t *testing.T) {

	left := `schema:
  type: string
example: tasty herbs in the morning
examples:
  exampleOne:
    value: yummy coffee
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
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareMediaTypes(&lDoc, &rDoc)
	assert.NotNil(t, extChanges)
	assert.Equal(t, 5, extChanges.TotalChanges())
	assert.Equal(t, 2, extChanges.TotalBreakingChanges())
}
