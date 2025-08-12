// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"context"
	"testing"

	"github.com/pkg-base/libopenapi/datamodel/low"
	"github.com/pkg-base/libopenapi/datamodel/low/base"
	"github.com/pkg-base/yaml"
	"github.com/stretchr/testify/assert"
)

func TestCompareSecurityRequirement_V2(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
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
	var lDoc base.SecurityRequirement
	var rDoc base.SecurityRequirement
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare
	extChanges := CompareSecurityRequirement(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareSecurityRequirement_NewReq_V2(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
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
	var lDoc base.SecurityRequirement
	var rDoc base.SecurityRequirement
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare
	extChanges := CompareSecurityRequirement(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, extChanges.Changes[0].ChangeType)
	assert.Equal(t, "biscuit", extChanges.Changes[0].NewObject)
}

func TestCompareSecurityRequirement_RemoveReq_V2(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
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
	var lDoc base.SecurityRequirement
	var rDoc base.SecurityRequirement
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare
	extChanges := CompareSecurityRequirement(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
}

// codecov seems to get upset with this not being covered.
// so lets run the damn thing a few hundred thousand times.
func BenchmarkCompareSecurityRequirement_Remove(b *testing.B) {
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

	for i := 0; i < b.N; i++ {
		var lDoc base.SecurityRequirement
		var rDoc base.SecurityRequirement
		_ = low.BuildModel(lNode.Content[0], &lDoc)
		_ = low.BuildModel(rNode.Content[0], &rDoc)
		_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
		_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)
		extChanges := CompareSecurityRequirement(&rDoc, &lDoc)
		assert.Equal(b, 1, extChanges.TotalChanges())
		assert.Len(b, extChanges.GetAllChanges(), 1)
		assert.Equal(b, 1, extChanges.TotalBreakingChanges())
	}
}

func TestCompareSecurityRequirement_SwapOut_V2(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
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
	var lDoc base.SecurityRequirement
	var rDoc base.SecurityRequirement
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare
	extChanges := CompareSecurityRequirement(&lDoc, &rDoc)
	assert.Equal(t, 4, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 4)
	assert.Equal(t, 2, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, ObjectRemoved, extChanges.Changes[1].ChangeType)
	assert.Equal(t, ObjectAdded, extChanges.Changes[2].ChangeType)
	assert.Equal(t, ObjectAdded, extChanges.Changes[3].ChangeType)
}

func TestCompareSecurityRequirement_SwapLeft_V2(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
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
	var lDoc base.SecurityRequirement
	var rDoc base.SecurityRequirement
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare
	extChanges := CompareSecurityRequirement(&lDoc, &rDoc)
	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, ObjectAdded, extChanges.Changes[1].ChangeType)
}

func TestCompareSecurityRequirement_AddedRole_V2(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
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
	var lDoc base.SecurityRequirement
	var rDoc base.SecurityRequirement
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare
	extChanges := CompareSecurityRequirement(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, extChanges.Changes[0].ChangeType)
	assert.Equal(t, "rich tea", extChanges.Changes[0].New)
}

func TestCompareSecurityRequirement_AddedMultiple_V2(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
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
	var lDoc base.SecurityRequirement
	var rDoc base.SecurityRequirement
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare
	extChanges := CompareSecurityRequirement(&lDoc, &rDoc)
	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, extChanges.Changes[0].ChangeType)
}

func TestCompareSecurityRequirement_ReplaceRole_V2(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
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
	var lDoc base.SecurityRequirement
	var rDoc base.SecurityRequirement
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare
	extChanges := CompareSecurityRequirement(&lDoc, &rDoc)
	assert.Equal(t, 3, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
}

func TestCompareSecurityRequirement_Identical_V2(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
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
	var lDoc base.SecurityRequirement
	var rDoc base.SecurityRequirement
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare
	extChanges := CompareSecurityRequirement(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareSecurityRequirement_RemovedRole_V2(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
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
	var lDoc base.SecurityRequirement
	var rDoc base.SecurityRequirement
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare
	extChanges := CompareSecurityRequirement(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, "rich tea", extChanges.Changes[0].Original)
}

func TestCompareSecurityRequirement_Add_V2(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `
biscuit:
  - biscotti
  - rich tea`

	right := `punch:
  - nice
cheese:
  - pizza
  - pie
biscuit:
  - biscotti
  - rich tea`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc base.SecurityRequirement
	var rDoc base.SecurityRequirement
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare
	extChanges := CompareSecurityRequirement(&lDoc, &rDoc)
	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, extChanges.Changes[0].ChangeType)
}
