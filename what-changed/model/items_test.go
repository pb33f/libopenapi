// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"context"
	"testing"

	"github.com/pkg-base/libopenapi/datamodel/low"
	v2 "github.com/pkg-base/libopenapi/datamodel/low/v2"
	v3 "github.com/pkg-base/libopenapi/datamodel/low/v3"
	"github.com/pkg-base/yaml"
	"github.com/stretchr/testify/assert"
)

func TestCompareItems(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `type: string`

	right := `type: int`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Items
	var rDoc v2.Items
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	changes := CompareItems(&lDoc, &rDoc)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.TypeLabel, changes.Changes[0].Property)
}

func TestCompareItems_RecursiveCheck(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
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
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	changes := CompareItems(&lDoc, &rDoc)
	assert.NotNil(t, changes)
	assert.Equal(t, 2, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 2)
	assert.Equal(t, 2, changes.TotalBreakingChanges())
	assert.Equal(t, 1, changes.ItemsChanges.TotalChanges())
	assert.Equal(t, v3.TypeLabel, changes.Changes[0].Property)
}

func TestCompareItems_AddItems(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
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
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	changes := CompareItems(&lDoc, &rDoc)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.ItemsLabel, changes.Changes[0].Property)
	assert.Equal(t, PropertyAdded, changes.Changes[0].ChangeType)
}

func TestCompareItems_RemoveItems(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
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
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	changes := CompareItems(&rDoc, &lDoc)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Len(t, changes.GetAllChanges(), 1)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
	assert.Equal(t, v3.ItemsLabel, changes.Changes[0].Property)
	assert.Equal(t, PropertyRemoved, changes.Changes[0].ChangeType)
}

func TestCompareItems_RefVsInlineIdentical(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
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
