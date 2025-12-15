// Copyright 2024-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

// TestQueryMethodConfigurableBreakingRules tests that the Query method's breaking change
// behavior can be configured via the breaking rules system.
func TestQueryMethodConfigurableBreakingRules(t *testing.T) {
	// ensure clean state
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	low.ClearHashCache()
	defer func() {
		ResetActiveBreakingRulesConfig()
		ResetDefaultBreakingRules()
	}()

	// Test that removing a Query operation is breaking by default
	left := `query:
  summary: Query resources
  operationId: queryResources`

	right := `get:
  summary: Get resources`

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

	// Default behavior: removing Query should be breaking
	changes := ComparePathItems(&lPath, &rPath)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalBreakingChanges(), "Removing query should be breaking by default")

	// Now configure Query removal as non-breaking
	customConfig := &BreakingRulesConfig{
		PathItem: &PathItemRules{
			Query: &BreakingChangeRule{
				Added:    boolPtr(false),
				Modified: boolPtr(false),
				Removed:  boolPtr(false), // Override: removal is not breaking
			},
		},
	}
	SetActiveBreakingRulesConfig(customConfig)

	// Re-run comparison with custom config
	changes2 := ComparePathItems(&lPath, &rPath)
	assert.NotNil(t, changes2)
	assert.Equal(t, 0, changes2.TotalBreakingChanges(), "With custom config, removing query should not be breaking")
}

// TestQueryMethodAddedConfigurable tests that adding a Query operation can be configured
// to be breaking or non-breaking.
func TestQueryMethodAddedConfigurable(t *testing.T) {
	// ensure clean state
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	low.ClearHashCache()
	defer func() {
		ResetActiveBreakingRulesConfig()
		ResetDefaultBreakingRules()
	}()

	left := `get:
  summary: Get resources`

	right := `get:
  summary: Get resources
query:
  summary: Query resources
  operationId: queryResources`

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

	// Default behavior: adding Query should not be breaking
	changes := ComparePathItems(&lPath, &rPath)
	assert.NotNil(t, changes)
	assert.Equal(t, 0, changes.TotalBreakingChanges(), "Adding query should not be breaking by default")

	// Now configure Query addition as breaking
	customConfig := &BreakingRulesConfig{
		PathItem: &PathItemRules{
			Query: &BreakingChangeRule{
				Added:    boolPtr(true), // Override: addition is breaking
				Modified: boolPtr(false),
				Removed:  boolPtr(true),
			},
		},
	}
	SetActiveBreakingRulesConfig(customConfig)

	// Re-run comparison with custom config
	changes2 := ComparePathItems(&lPath, &rPath)
	assert.NotNil(t, changes2)
	assert.Equal(t, 1, changes2.TotalBreakingChanges(), "With custom config, adding query should be breaking")
}

// TestAdditionalOperationsConfigurableBreakingRules tests that AdditionalOperations
// breaking change behavior can be configured.
func TestAdditionalOperationsConfigurableBreakingRules(t *testing.T) {
	// ensure clean state
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	low.ClearHashCache()
	defer func() {
		ResetActiveBreakingRulesConfig()
		ResetDefaultBreakingRules()
	}()

	// Test with additionalOperations present then removed
	left := `additionalOperations:
  CUSTOM:
    summary: Custom operation`

	right := `get:
  summary: Get resources`

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

	// Default behavior: removing additionalOperations should be breaking
	changes := ComparePathItems(&lPath, &rPath)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalBreakingChanges(), "Removing additionalOperations should be breaking by default")

	// Now configure as non-breaking
	customConfig := &BreakingRulesConfig{
		PathItem: &PathItemRules{
			AdditionalOperations: &BreakingChangeRule{
				Added:    boolPtr(false),
				Modified: boolPtr(false),
				Removed:  boolPtr(false), // Override: removal is not breaking
			},
		},
	}
	SetActiveBreakingRulesConfig(customConfig)

	// Re-run comparison with custom config
	changes2 := ComparePathItems(&lPath, &rPath)
	assert.NotNil(t, changes2)
	assert.Equal(t, 0, changes2.TotalBreakingChanges(), "With custom config, removing additionalOperations should not be breaking")
}

// TestItemSchemaConfigurableBreakingRules tests that the ItemSchema field in MediaType
// has configurable breaking change behavior.
func TestItemSchemaConfigurableBreakingRules(t *testing.T) {
	// ensure clean state
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	low.ClearHashCache()
	defer func() {
		ResetActiveBreakingRulesConfig()
		ResetDefaultBreakingRules()
	}()

	// Test with itemSchema present then removed
	left := `schema:
  type: array
itemSchema:
  type: string`

	right := `schema:
  type: array`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	lIdx := index.NewSpecIndexWithConfig(&lNode, index.CreateOpenAPIIndexConfig())
	rIdx := index.NewSpecIndexWithConfig(&rNode, index.CreateOpenAPIIndexConfig())
	ctx := context.Background()

	var lMT, rMT v3.MediaType
	_ = low.BuildModel(&lNode, &lMT)
	_ = low.BuildModel(&rNode, &rMT)

	_ = lMT.Build(ctx, nil, lNode.Content[0], lIdx)
	_ = rMT.Build(ctx, nil, rNode.Content[0], rIdx)

	// Default behavior: removing itemSchema should be breaking
	changes := CompareMediaTypes(&lMT, &rMT)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalBreakingChanges(), "Removing itemSchema should be breaking by default")

	// Now configure as non-breaking
	customConfig := &BreakingRulesConfig{
		MediaType: &MediaTypeRules{
			ItemSchema: &BreakingChangeRule{
				Added:    boolPtr(false),
				Modified: boolPtr(false),
				Removed:  boolPtr(false), // Override: removal is not breaking
			},
		},
	}
	SetActiveBreakingRulesConfig(customConfig)

	// Re-run comparison with custom config
	changes2 := CompareMediaTypes(&lMT, &rMT)
	assert.NotNil(t, changes2)
	assert.Equal(t, 0, changes2.TotalBreakingChanges(), "With custom config, removing itemSchema should not be breaking")
}

// TestItemSchemaAddedConfigurable tests that adding itemSchema can be configured.
func TestItemSchemaAddedConfigurable(t *testing.T) {
	// ensure clean state
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	low.ClearHashCache()
	defer func() {
		ResetActiveBreakingRulesConfig()
		ResetDefaultBreakingRules()
	}()

	left := `schema:
  type: array`

	right := `schema:
  type: array
itemSchema:
  type: string`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	lIdx := index.NewSpecIndexWithConfig(&lNode, index.CreateOpenAPIIndexConfig())
	rIdx := index.NewSpecIndexWithConfig(&rNode, index.CreateOpenAPIIndexConfig())
	ctx := context.Background()

	var lMT, rMT v3.MediaType
	_ = low.BuildModel(&lNode, &lMT)
	_ = low.BuildModel(&rNode, &rMT)

	_ = lMT.Build(ctx, nil, lNode.Content[0], lIdx)
	_ = rMT.Build(ctx, nil, rNode.Content[0], rIdx)

	// Default behavior: adding itemSchema should be breaking
	changes := CompareMediaTypes(&lMT, &rMT)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalBreakingChanges(), "Adding itemSchema should be breaking by default")

	// Now configure as non-breaking
	customConfig := &BreakingRulesConfig{
		MediaType: &MediaTypeRules{
			ItemSchema: &BreakingChangeRule{
				Added:    boolPtr(false), // Override: addition is not breaking
				Modified: boolPtr(false),
				Removed:  boolPtr(true),
			},
		},
	}
	SetActiveBreakingRulesConfig(customConfig)

	// Re-run comparison with custom config
	changes2 := CompareMediaTypes(&lMT, &rMT)
	assert.NotNil(t, changes2)
	assert.Equal(t, 0, changes2.TotalBreakingChanges(), "With custom config, adding itemSchema should not be breaking")
}

// TestOAuth2MetadataUrlConfigurableBreakingRules tests that the oauth2MetadataUrl field
// in SecurityScheme has configurable breaking change behavior.
func TestOAuth2MetadataUrlConfigurableBreakingRules(t *testing.T) {
	// ensure clean state
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	low.ClearHashCache()
	defer func() {
		ResetActiveBreakingRulesConfig()
		ResetDefaultBreakingRules()
	}()

	// Include type field which is required for the comparison to work properly
	left := `type: oauth2
description: OAuth2 auth
oauth2MetadataUrl: https://example.com/.well-known/oauth-authorization-server`

	right := `type: oauth2
description: OAuth2 auth
oauth2MetadataUrl: https://example.com/v2/.well-known/oauth-authorization-server`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	ctx := context.Background()

	var lSS, rSS v3.SecurityScheme
	_ = low.BuildModel(lNode.Content[0], &lSS)
	_ = low.BuildModel(rNode.Content[0], &rSS)

	_ = lSS.Build(ctx, nil, lNode.Content[0], nil)
	_ = rSS.Build(ctx, nil, rNode.Content[0], nil)

	// Default behavior: modifying oauth2MetadataUrl should not be breaking
	changes := CompareSecuritySchemes(&lSS, &rSS)
	assert.NotNil(t, changes, "Changes should not be nil when oauth2MetadataUrl differs")
	assert.Equal(t, 1, changes.TotalChanges(), "Should detect 1 change for oauth2MetadataUrl modification")
	assert.Equal(t, 0, changes.TotalBreakingChanges(), "Modifying oauth2MetadataUrl should not be breaking by default")

	// Now configure as breaking
	customConfig := &BreakingRulesConfig{
		SecurityScheme: &SecuritySchemeRules{
			OAuth2MetadataUrl: &BreakingChangeRule{
				Added:    boolPtr(true),
				Modified: boolPtr(true), // Override: modification is breaking
				Removed:  boolPtr(true),
			},
		},
	}
	SetActiveBreakingRulesConfig(customConfig)

	// Re-run comparison with custom config
	changes2 := CompareSecuritySchemes(&lSS, &rSS)
	assert.NotNil(t, changes2, "Changes should not be nil with custom config")
	assert.Equal(t, 1, changes2.TotalBreakingChanges(), "With custom config, modifying oauth2MetadataUrl should be breaking")
}

// TestDefaultBreakingRules_Query verifies the default breaking rules for Query method
func TestDefaultBreakingRules_Query(t *testing.T) {
	ResetDefaultBreakingRules()
	config := GenerateDefaultBreakingRules()

	// Query should have same defaults as other HTTP methods
	assert.NotNil(t, config.PathItem.Query)
	assert.False(t, *config.PathItem.Query.Added, "Adding query should not be breaking by default")
	assert.False(t, *config.PathItem.Query.Modified, "Modifying query should not be breaking by default")
	assert.True(t, *config.PathItem.Query.Removed, "Removing query should be breaking by default")
}

// TestDefaultBreakingRules_AdditionalOperations verifies the default breaking rules for AdditionalOperations
func TestDefaultBreakingRules_AdditionalOperations(t *testing.T) {
	ResetDefaultBreakingRules()
	config := GenerateDefaultBreakingRules()

	assert.NotNil(t, config.PathItem.AdditionalOperations)
	assert.False(t, *config.PathItem.AdditionalOperations.Added, "Adding additionalOperations should not be breaking by default")
	assert.False(t, *config.PathItem.AdditionalOperations.Modified, "Modifying additionalOperations should not be breaking by default")
	assert.True(t, *config.PathItem.AdditionalOperations.Removed, "Removing additionalOperations should be breaking by default")
}

// TestDefaultBreakingRules_ItemSchema verifies the default breaking rules for ItemSchema
func TestDefaultBreakingRules_ItemSchema(t *testing.T) {
	ResetDefaultBreakingRules()
	config := GenerateDefaultBreakingRules()

	assert.NotNil(t, config.MediaType.ItemSchema)
	assert.True(t, *config.MediaType.ItemSchema.Added, "Adding itemSchema should be breaking by default")
	assert.False(t, *config.MediaType.ItemSchema.Modified, "Modifying itemSchema should not be breaking by default")
	assert.True(t, *config.MediaType.ItemSchema.Removed, "Removing itemSchema should be breaking by default")
}

// TestDefaultBreakingRules_ItemEncoding verifies the default breaking rules for ItemEncoding
func TestDefaultBreakingRules_ItemEncoding(t *testing.T) {
	ResetDefaultBreakingRules()
	config := GenerateDefaultBreakingRules()

	assert.NotNil(t, config.MediaType.ItemEncoding)
	assert.False(t, *config.MediaType.ItemEncoding.Added, "Adding itemEncoding should not be breaking by default")
	assert.False(t, *config.MediaType.ItemEncoding.Modified, "Modifying itemEncoding should not be breaking by default")
	assert.True(t, *config.MediaType.ItemEncoding.Removed, "Removing itemEncoding should be breaking by default")
}

// TestDefaultBreakingRules_OAuth2MetadataUrl verifies the default breaking rules for OAuth2MetadataUrl
func TestDefaultBreakingRules_OAuth2MetadataUrl(t *testing.T) {
	ResetDefaultBreakingRules()
	config := GenerateDefaultBreakingRules()

	assert.NotNil(t, config.SecurityScheme.OAuth2MetadataUrl)
	assert.False(t, *config.SecurityScheme.OAuth2MetadataUrl.Added, "Adding oauth2MetadataUrl should not be breaking by default")
	assert.False(t, *config.SecurityScheme.OAuth2MetadataUrl.Modified, "Modifying oauth2MetadataUrl should not be breaking by default")
	assert.False(t, *config.SecurityScheme.OAuth2MetadataUrl.Removed, "Removing oauth2MetadataUrl should not be breaking by default")
}

// TestBreakingHelperFunctions_OpenAPI32 tests the BreakingAdded/Modified/Removed helper functions
// work correctly with the new OpenAPI 3.2 properties.
func TestBreakingHelperFunctions_OpenAPI32(t *testing.T) {
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	defer func() {
		ResetActiveBreakingRulesConfig()
		ResetDefaultBreakingRules()
	}()

	// Test Query
	assert.False(t, BreakingAdded(CompPathItem, PropQuery))
	assert.False(t, BreakingModified(CompPathItem, PropQuery))
	assert.True(t, BreakingRemoved(CompPathItem, PropQuery))

	// Test AdditionalOperations
	assert.False(t, BreakingAdded(CompPathItem, PropAdditionalOperations))
	assert.False(t, BreakingModified(CompPathItem, PropAdditionalOperations))
	assert.True(t, BreakingRemoved(CompPathItem, PropAdditionalOperations))

	// Test ItemSchema
	assert.True(t, BreakingAdded(CompMediaType, PropItemSchema))
	assert.False(t, BreakingModified(CompMediaType, PropItemSchema))
	assert.True(t, BreakingRemoved(CompMediaType, PropItemSchema))

	// Test ItemEncoding
	assert.False(t, BreakingAdded(CompMediaType, PropItemEncoding))
	assert.False(t, BreakingModified(CompMediaType, PropItemEncoding))
	assert.True(t, BreakingRemoved(CompMediaType, PropItemEncoding))

	// Test OAuth2MetadataUrl
	assert.False(t, BreakingAdded(CompSecurityScheme, PropOAuth2MetadataUrl))
	assert.False(t, BreakingModified(CompSecurityScheme, PropOAuth2MetadataUrl))
	assert.False(t, BreakingRemoved(CompSecurityScheme, PropOAuth2MetadataUrl))
}

// TestSchemaDescriptionAddedConfigurable tests that adding a schema description respects
// the configurable breaking rules when schema.description.added is set to true.
func TestSchemaDescriptionAddedConfigurable(t *testing.T) {
	// ensure clean state
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	low.ClearHashCache()
	defer func() {
		ResetActiveBreakingRulesConfig()
		ResetDefaultBreakingRules()
	}()

	// Original schema without description
	left := `openapi: 3.0
components:
  schemas:
    Tree:
      type: string`

	// New schema with description added
	right := `openapi: 3.0
components:
  schemas:
    Tree:
      type: string
      description: new`

	leftDoc, rightDoc := test_BuildDoc(left, right)

	lSchemaProxy := leftDoc.Components.Value.FindSchema("Tree").Value
	rSchemaProxy := rightDoc.Components.Value.FindSchema("Tree").Value

	// Test 1: By default, adding a description is NOT breaking
	changes := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Equal(t, 0, changes.TotalBreakingChanges()) // Default: not breaking

	// Test 2: With custom config where description.added = true, should be breaking
	ResetActiveBreakingRulesConfig()
	low.ClearHashCache()

	customConfig := &BreakingRulesConfig{
		Schema: &SchemaRules{
			Description: &BreakingChangeRule{
				Added:    boolPtr(true),
				Modified: boolPtr(false),
				Removed:  boolPtr(false),
			},
		},
	}
	SetActiveBreakingRulesConfig(customConfig)

	changes2 := CompareSchemas(lSchemaProxy, rSchemaProxy)
	assert.NotNil(t, changes2)
	assert.Equal(t, 1, changes2.TotalChanges())
	assert.Equal(t, 1, changes2.TotalBreakingChanges()) // Custom: IS breaking
}
