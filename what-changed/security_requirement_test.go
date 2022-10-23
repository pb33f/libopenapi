// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package what_changed

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/v2"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestCompareSecurityRequirement(t *testing.T) {

	left := `auth:
 - pizza
 - pie`

	right := `auth:
 - pie
 - pizza`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.SecurityRequirement
	var rDoc v2.SecurityRequirement
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare
	extChanges := CompareSecurityRequirement(&lDoc, &rDoc)
	assert.Nil(t, extChanges)

}
func TestCompareSecurityRequirement_NewReq(t *testing.T) {

	left := `tip:
  - tap
auth:
  - pizza
  - pie`

	right := `auth:
  - pie
  - pizza
tip:
  - tap
biscuit:
  - digestive`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.SecurityRequirement
	var rDoc v2.SecurityRequirement
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare
	extChanges := CompareSecurityRequirement(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, extChanges.Changes[0].ChangeType)
	assert.Equal(t, "biscuit", extChanges.Changes[0].NewObject)
}

func TestCompareSecurityRequirement_RemoveReq(t *testing.T) {

	left := `auth:
  - pizza
  - pie`

	right := `auth:
  - pie
  - pizza
biscuit:
  - digestive`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.SecurityRequirement
	var rDoc v2.SecurityRequirement
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare
	extChanges := CompareSecurityRequirement(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
}

func TestCompareSecurityRequirement_SwapOut(t *testing.T) {

	left := `cheese:
  - pizza
  - pie
biscuit:
  - digestive`

	right := `bread:
  - pie
  - pizza
milk:
  - digestive`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.SecurityRequirement
	var rDoc v2.SecurityRequirement
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare
	extChanges := CompareSecurityRequirement(&lDoc, &rDoc)
	assert.Equal(t, 4, extChanges.TotalChanges())
	assert.Equal(t, 2, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, ObjectRemoved, extChanges.Changes[1].ChangeType)
	assert.Equal(t, ObjectAdded, extChanges.Changes[2].ChangeType)
	assert.Equal(t, ObjectAdded, extChanges.Changes[3].ChangeType)
}

func TestCompareSecurityRequirement_SwapLeft(t *testing.T) {

	left := `cheese:
  - pizza
  - pie
biscuit:
  - digestive`

	right := `cheese:
  - pie
  - pizza
milk:
  - digestive`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.SecurityRequirement
	var rDoc v2.SecurityRequirement
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare
	extChanges := CompareSecurityRequirement(&lDoc, &rDoc)
	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, ObjectAdded, extChanges.Changes[1].ChangeType)
}

func TestCompareSecurityRequirement_AddedRole(t *testing.T) {

	left := `cheese:
  - pizza
  - pie
biscuit:
  - digestive`

	right := `cheese:
  - pizza
  - pie
biscuit:
  - digestive
  - rich tea`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.SecurityRequirement
	var rDoc v2.SecurityRequirement
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare
	extChanges := CompareSecurityRequirement(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, extChanges.Changes[0].ChangeType)
	assert.Equal(t, "rich tea", extChanges.Changes[0].New)
}

func TestCompareSecurityRequirement_AddedMultiple(t *testing.T) {

	left := `cheese:
  - pizza
  - pie
biscuit:
  - digestive`

	right := `cheese:
  - pizza
  - pie
cake:
  - vanilla
  - choccy
biscuit:
  - digestive
  - rich tea`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.SecurityRequirement
	var rDoc v2.SecurityRequirement
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare
	extChanges := CompareSecurityRequirement(&lDoc, &rDoc)
	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, extChanges.Changes[0].ChangeType)
}

func TestCompareSecurityRequirement_ReplaceRole(t *testing.T) {

	left := `cheese:
  - pizza
  - pie
biscuit:
  - digestive`

	right := `cheese:
  - pizza
  - pie
biscuit:
  - biscotti
  - rich tea`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.SecurityRequirement
	var rDoc v2.SecurityRequirement
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare
	extChanges := CompareSecurityRequirement(&lDoc, &rDoc)
	assert.Equal(t, 3, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
}

func TestCompareSecurityRequirement_Identical(t *testing.T) {

	left := `cheese:
  - pizza
  - pie
biscuit:
  - biscotti
  - rich tea`

	right := `cheese:
  - pizza
  - pie
biscuit:
  - biscotti
  - rich tea`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.SecurityRequirement
	var rDoc v2.SecurityRequirement
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare
	extChanges := CompareSecurityRequirement(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareSecurityRequirement_RemovedRole(t *testing.T) {

	left := `cheese:
  - pizza
  - pie
biscuit:
  - digestive`

	right := `cheese:
  - pizza
  - pie
biscuit:
  - digestive
  - rich tea`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.SecurityRequirement
	var rDoc v2.SecurityRequirement
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare
	extChanges := CompareSecurityRequirement(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, "rich tea", extChanges.Changes[0].Original)
}
