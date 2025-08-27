// Copyright 2024 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestComparePathItems_QueryAdded(t *testing.T) {
	// Clear hash cache to ensure deterministic test results
	low.ClearHashCache()

	left := `get:
  summary: Get resource
  operationId: getResource`

	right := `get:
  summary: Get resource
  operationId: getResource
query:
  summary: Query resources
  operationId: queryResources
  requestBody:
    required: true
    content:
      application/json:
        schema:
          type: object`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	lIdx := index.NewSpecIndexWithConfig(&lNode, index.CreateOpenAPIIndexConfig())
	rIdx := index.NewSpecIndexWithConfig(&rNode, index.CreateOpenAPIIndexConfig())
	ctx := context.Background()

	var lPath, rPath v3.PathItem
	_ = low.BuildModel(&lNode, &lPath)
	_ = low.BuildModel(&rNode, &rPath)

	_ = lPath.Build(ctx, nil, lNode.Content[0], lIdx)
	_ = rPath.Build(ctx, nil, rNode.Content[0], rIdx)

	// Compare paths
	changes := ComparePathItems(&lPath, &rPath)

	assert.NotNil(t, changes)
	assert.Nil(t, changes.QueryChanges) // No changes to existing query operation
	assert.Equal(t, 1, changes.TotalChanges())

	// Check that query was added
	foundQueryAdded := false
	for _, change := range changes.GetAllChanges() {
		if change.Property == "query" && change.ChangeType == PropertyAdded {
			foundQueryAdded = true
			break
		}
	}
	assert.True(t, foundQueryAdded, "Query operation should be detected as added")
}

func TestComparePathItems_QueryRemoved(t *testing.T) {
	// Clear hash cache to ensure deterministic test results
	low.ClearHashCache()

	left := `get:
  summary: Get resource
  operationId: getResource
query:
  summary: Query resources
  operationId: queryResources
  requestBody:
    required: true
    content:
      application/json:
        schema:
          type: object`

	right := `get:
  summary: Get resource
  operationId: getResource`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	lIdx := index.NewSpecIndexWithConfig(&lNode, index.CreateOpenAPIIndexConfig())
	rIdx := index.NewSpecIndexWithConfig(&rNode, index.CreateOpenAPIIndexConfig())
	ctx := context.Background()

	var lPath, rPath v3.PathItem
	_ = low.BuildModel(&lNode, &lPath)
	_ = low.BuildModel(&rNode, &rPath)

	_ = lPath.Build(ctx, nil, lNode.Content[0], lIdx)
	_ = rPath.Build(ctx, nil, rNode.Content[0], rIdx)

	// Compare paths
	changes := ComparePathItems(&lPath, &rPath)

	assert.NotNil(t, changes)
	assert.Nil(t, changes.QueryChanges) // No changes to existing query operation
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Equal(t, 1, changes.TotalBreakingChanges()) // Removing an operation is breaking

	// Check that query was removed
	foundQueryRemoved := false
	for _, change := range changes.GetAllChanges() {
		if change.Property == "query" && change.ChangeType == PropertyRemoved {
			foundQueryRemoved = true
			assert.True(t, change.Breaking)
			break
		}
	}
	assert.True(t, foundQueryRemoved, "Query operation should be detected as removed")
}

func TestComparePathItems_QueryModified(t *testing.T) {
	// Clear hash cache to ensure deterministic test results
	low.ClearHashCache()

	left := `query:
  summary: Query resources
  operationId: queryResources
  responses:
    '200':
      description: OK`

	right := `query:
  summary: Query resources with filters
  operationId: queryResourcesV2
  requestBody:
    required: true
    content:
      application/json:
        schema:
          type: object
  responses:
    '200':
      description: Query results`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	lIdx := index.NewSpecIndexWithConfig(&lNode, index.CreateOpenAPIIndexConfig())
	rIdx := index.NewSpecIndexWithConfig(&rNode, index.CreateOpenAPIIndexConfig())
	ctx := context.Background()

	var lPath, rPath v3.PathItem
	_ = low.BuildModel(&lNode, &lPath)
	_ = low.BuildModel(&rNode, &rPath)

	_ = lPath.Build(ctx, nil, lNode.Content[0], lIdx)
	_ = rPath.Build(ctx, nil, rNode.Content[0], rIdx)

	// Compare paths
	changes := ComparePathItems(&lPath, &rPath)

	assert.NotNil(t, changes)
	assert.NotNil(t, changes.QueryChanges)

	// Check for specific changes
	assert.True(t, changes.QueryChanges.TotalChanges() > 0)

	// Summary should have changed
	foundSummaryChange := false
	foundOperationIdChange := false
	foundRequestBodyAdded := false

	for _, change := range changes.QueryChanges.GetAllChanges() {
		if change.Property == "summary" {
			foundSummaryChange = true
			assert.Equal(t, "Query resources", change.Original)
			assert.Equal(t, "Query resources with filters", change.New)
		}
		if change.Property == "operationId" {
			foundOperationIdChange = true
			assert.Equal(t, "queryResources", change.Original)
			assert.Equal(t, "queryResourcesV2", change.New)
		}
		if change.Property == "requestBody" && change.ChangeType == PropertyAdded {
			foundRequestBodyAdded = true
		}
	}

	assert.True(t, foundSummaryChange, "Summary change should be detected")
	assert.True(t, foundOperationIdChange, "OperationId change should be detected")
	assert.True(t, foundRequestBodyAdded, "RequestBody addition should be detected")
}

func TestComparePathItems_AllOperationsIncludingQuery(t *testing.T) {
	// Clear hash cache to ensure deterministic test results
	low.ClearHashCache()

	left := `get:
  summary: Get v1
post:
  summary: Post v1
query:
  summary: Query v1`

	right := `get:
  summary: Get v2
post:
  summary: Post v2
query:
  summary: Query v2`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	lIdx := index.NewSpecIndexWithConfig(&lNode, index.CreateOpenAPIIndexConfig())
	rIdx := index.NewSpecIndexWithConfig(&rNode, index.CreateOpenAPIIndexConfig())
	ctx := context.Background()

	var lPath, rPath v3.PathItem
	_ = low.BuildModel(&lNode, &lPath)
	_ = low.BuildModel(&rNode, &rPath)

	_ = lPath.Build(ctx, nil, lNode.Content[0], lIdx)
	_ = rPath.Build(ctx, nil, rNode.Content[0], rIdx)

	// Compare paths
	changes := ComparePathItems(&lPath, &rPath)

	assert.NotNil(t, changes)
	assert.NotNil(t, changes.GetChanges)
	assert.NotNil(t, changes.PostChanges)
	assert.NotNil(t, changes.QueryChanges)

	// Each operation should have changes
	assert.True(t, changes.GetChanges.TotalChanges() > 0)
	assert.True(t, changes.PostChanges.TotalChanges() > 0)
	assert.True(t, changes.QueryChanges.TotalChanges() > 0)
}

// TestPathItemChanges_QueryChangesNotNil ensures we hit the branches where QueryChanges is not nil
// This test covers lines 63, 108, and 150 in path_item.go
func TestPathItemChanges_QueryChangesNotNil(t *testing.T) {
	// Create a PathItemChanges with QueryChanges present
	pc := &PathItemChanges{
		PropertyChanges: &PropertyChanges{
			Changes: []*Change{
				{
					ChangeType: PropertyAdded,
					Property:   "description",
					Breaking:   false,
				},
			},
		},
		QueryChanges: &OperationChanges{
			PropertyChanges: &PropertyChanges{
				Changes: []*Change{
					{
						ChangeType: Modified,
						Property:   "summary",
						Breaking:   false,
					},
					{
						ChangeType: PropertyRemoved,
						Property:   "operationId",
						Breaking:   true,
					},
				},
			},
		},
		TraceChanges: &OperationChanges{
			PropertyChanges: &PropertyChanges{
				Changes: []*Change{
					{
						ChangeType: Modified,
						Property:   "deprecated",
						Breaking:   false,
					},
				},
			},
		},
	}

	// Test GetAllChanges() - this covers line 63 where QueryChanges is not nil
	allChanges := pc.GetAllChanges()
	assert.NotNil(t, allChanges)
	assert.Equal(t, 4, len(allChanges)) // 1 from PropertyChanges + 2 from QueryChanges + 1 from TraceChanges
	
	// Verify QueryChanges were included
	queryChangeCount := 0
	for _, change := range allChanges {
		if (change.Property == "summary" && change.ChangeType == Modified) ||
		   (change.Property == "operationId" && change.ChangeType == PropertyRemoved) {
			queryChangeCount++
		}
	}
	assert.Equal(t, 2, queryChangeCount, "Both QueryChanges should be in the result")

	// Test TotalChanges() - this covers line 108 where QueryChanges is not nil
	total := pc.TotalChanges()
	assert.Equal(t, 4, total) // Should include all changes

	// Test TotalBreakingChanges() - this covers line 150 where QueryChanges is not nil
	breaking := pc.TotalBreakingChanges()
	assert.Equal(t, 1, breaking) // Only the operationId removal is breaking
}

// TestPathItemChanges_QueryChangesNil ensures we hit the branches where QueryChanges is nil
// This test covers the nil checks at lines 62, 107, and 149 in path_item.go
func TestPathItemChanges_QueryChangesNil(t *testing.T) {
	// Create a PathItemChanges with QueryChanges nil
	pc := &PathItemChanges{
		PropertyChanges: &PropertyChanges{
			Changes: []*Change{
				{
					ChangeType: PropertyAdded,
					Property:   "summary",
					Breaking:   false,
				},
			},
		},
		QueryChanges: nil, // Explicitly nil
		TraceChanges: &OperationChanges{
			PropertyChanges: &PropertyChanges{
				Changes: []*Change{
					{
						ChangeType: Modified,
						Property:   "tags",
						Breaking:   false,
					},
				},
			},
		},
	}

	// Test GetAllChanges() with nil QueryChanges - covers the if statement at line 62
	allChanges := pc.GetAllChanges()
	assert.NotNil(t, allChanges)
	assert.Equal(t, 2, len(allChanges)) // 1 from PropertyChanges + 1 from TraceChanges (QueryChanges skipped)

	// Test TotalChanges() with nil QueryChanges - covers the if statement at line 107
	total := pc.TotalChanges()
	assert.Equal(t, 2, total) // Should not include QueryChanges

	// Test TotalBreakingChanges() with nil QueryChanges - covers the if statement at line 149
	breaking := pc.TotalBreakingChanges()
	assert.Equal(t, 0, breaking) // No breaking changes in our test data
}