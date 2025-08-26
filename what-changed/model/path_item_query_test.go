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