// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/v2"
	"github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestCompareItems(t *testing.T) {

	left := `type: string`

	right := `type: int`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Items
	var rDoc v2.Items
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	changes := CompareItems(&lDoc, &rDoc)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.TypeLabel, changes.Changes[0].Property)
}

func TestCompareItems_RecursiveCheck(t *testing.T) {

	left := `type: string
items:
  type: string`

	right := `type: int
items:
  type: int`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Items
	var rDoc v2.Items
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	changes := CompareItems(&lDoc, &rDoc)
	assert.NotNil(t, changes)
	assert.Equal(t, 2, changes.TotalChanges())
	assert.Equal(t, 2, changes.TotalBreakingChanges())
	assert.Equal(t, 1, changes.ItemsChanges.TotalChanges())
	assert.Equal(t, v3.TypeLabel, changes.Changes[0].Property)
}

func TestCompareItems_AddItems(t *testing.T) {

	left := `type: int`

	right := `type: int
items:
  type: int`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Items
	var rDoc v2.Items
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	changes := CompareItems(&lDoc, &rDoc)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.ItemsLabel, changes.Changes[0].Property)
	assert.Equal(t, PropertyAdded, changes.Changes[0].ChangeType)
}

func TestCompareItems_RemoveItems(t *testing.T) {

	left := `type: int`

	right := `type: int
items:
  type: int`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Items
	var rDoc v2.Items
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	changes := CompareItems(&rDoc, &lDoc)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.ItemsLabel, changes.Changes[0].Property)
	assert.Equal(t, PropertyRemoved, changes.Changes[0].ChangeType)
}

func TestCompareItems_RefVsInlineIdentical(t *testing.T) {

	left := `swagger: 2.0
definitions:
  thing:
    type: string
    items:
      $ref: '#/definitions/thang'
  thang:
    type: bool
paths:
  "/a/path":
    get:
      parameters:
        - name: status
          items:
            $ref: '#/definitions/thing'`

	right := `swagger: 2.0
definitions:
  thing:
    type: string
  thang:
    type: int
paths:
  "/a/path":
    get:
      parameters:
        - name: status
          items:
            type: string
            items:
              type: bool`

	leftDoc, rightDoc := test_BuildDocv2(left, right)

	// extract left reference schema and non reference schema.
	lItems := leftDoc.Paths.Value.FindPath("/a/path").Value.Get.Value.Parameters.
		Value[0].Value.Items.Value
	rItems := rightDoc.Paths.Value.FindPath("/a/path").Value.Get.Value.Parameters.
		Value[0].Value.Items.Value

	// compare.
	changes := CompareItems(lItems, rItems)
	assert.Nil(t, changes)
}
