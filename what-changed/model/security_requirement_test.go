// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
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

// Test that scope additions show the scheme name as the property
func TestCompareSecurityRequirement_AddScope_ShowsSchemeName(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `OAuth2:
  - read`

	right := `OAuth2:
  - read
  - write`

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
	assert.Equal(t, "OAuth2", extChanges.Changes[0].Property) // Verify scheme name in property
	assert.Equal(t, "write", extChanges.Changes[0].New)       // Verify scope name in new value
}

// Test that scope removals show the scheme name as the property
func TestCompareSecurityRequirement_RemoveScope_ShowsSchemeName(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `OAuth2:
  - read
  - write`

	right := `OAuth2:
  - read`

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
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, "OAuth2", extChanges.Changes[0].Property) // Verify scheme name in property
	assert.Equal(t, "write", extChanges.Changes[0].Original)  // Verify scope name in original value
}

// Test that multiple scope additions show the scheme name as the property
func TestCompareSecurityRequirement_AddMultipleScopes_ShowsSchemeName(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `OAuth2:
  - read`

	right := `OAuth2:
  - read
  - write
  - admin`

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

	// Both changes should have OAuth2 as the property
	for _, change := range extChanges.Changes {
		assert.Equal(t, ObjectAdded, change.ChangeType)
		assert.Equal(t, "OAuth2", change.Property)
	}
	// Verify the scope names
	scopes := []string{extChanges.Changes[0].New, extChanges.Changes[1].New}
	assert.Contains(t, scopes, "write")
	assert.Contains(t, scopes, "admin")
}

// Test that adding an entire scheme shows the scheme name as both property and value
func TestCompareSecurityRequirement_AddEntireScheme_ShowsSchemeName(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `OAuth2:
  - read`

	right := `OAuth2:
  - read
ApiKey: []`

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
	assert.Equal(t, "ApiKey", extChanges.Changes[0].Property) // Verify scheme name in property
	assert.Equal(t, "ApiKey", extChanges.Changes[0].New)      // Verify scheme name in new value (entire scheme added)
}

// Test that removing an entire scheme shows the scheme name as both property and value
func TestCompareSecurityRequirement_RemoveEntireScheme_ShowsSchemeName(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `OAuth2:
  - read
ApiKey: []`

	right := `OAuth2:
  - read`

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
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, "ApiKey", extChanges.Changes[0].Property) // Verify scheme name in property
	assert.Equal(t, "ApiKey", extChanges.Changes[0].Original) // Verify scheme name in original value (entire scheme removed)
}

// Test real-world OAuth2 example from the bug report
func TestCompareSecurityRequirement_OAuth2_RealWorld(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `OAuth2:
  - read
  - execute`

	right := `OAuth2:
  - read
  - write
  - execute`

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
	// The key assertion: property should be "OAuth2", not "security"
	assert.Equal(t, "OAuth2", extChanges.Changes[0].Property)
	assert.Equal(t, "write", extChanges.Changes[0].New)
}

// Test that empty scope values from malformed YAML are ignored
// This handles cases where YAML like "secure:\n  lock:" (mapping) instead of
// "secure:\n  - lock" (sequence) causes empty string scope values
func TestCompareSecurityRequirement_IgnoresEmptyScopeValues(t *testing.T) {
	low.ClearHashCache()

	// Simulating malformed YAML where scope values include empty strings
	// Original: secure with no scopes
	left := `secure:`

	// Modified: secure with "lock" scope (plus potential empty string from malformed YAML)
	// In properly formed YAML this would be "secure:\n  - lock"
	// But malformed "secure:\n  lock:" could create ["lock", ""]
	right := `secure:
  - lock`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	var lDoc base.SecurityRequirement
	var rDoc base.SecurityRequirement
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare
	extChanges := CompareSecurityRequirement(&lDoc, &rDoc)
	assert.NotNil(t, extChanges)
	// Should only have 1 change (the "lock" scope addition), NOT 2 (lock + empty string)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, ObjectAdded, extChanges.Changes[0].ChangeType)
	assert.Equal(t, "secure", extChanges.Changes[0].Property)
	assert.Equal(t, "lock", extChanges.Changes[0].New)
}

// Test that empty scope values are properly filtered during comparison
// This covers lines 126-127 and 136-137 in checkSecurityRequirement
func TestCompareSecurityRequirement_EmptyScopeValuesFiltered(t *testing.T) {
	low.ClearHashCache()

	// Left has scopes with an empty string mixed in
	left := `secure:
  - lock
  - ""
  - key`

	// Right has scopes with an empty string mixed in
	right := `secure:
  - lock
  - ""
  - key
  - newscope`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	var lDoc base.SecurityRequirement
	var rDoc base.SecurityRequirement
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare - empty strings should be filtered out
	extChanges := CompareSecurityRequirement(&lDoc, &rDoc)
	assert.NotNil(t, extChanges)
	// Should only detect the "newscope" addition, empty strings filtered
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, ObjectAdded, extChanges.Changes[0].ChangeType)
	assert.Equal(t, "newscope", extChanges.Changes[0].New)
}
