// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestCompareCallback(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `'{$request.query.queryUrl}':
    post:
      requestBody:
        description: Callback payload
        content: 
          'application/json':
            schema:
              type: int
      responses:
        '200':
          description: callback successfully processed`

	right := left

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Callback
	var rDoc v3.Callback
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareCallback(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareCallback_Add(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `'{$request.query.queryUrl}':
    post:
      requestBody:
        description: Callback payload
        content: 
          'application/json':
            schema:
              type: int
      responses:
        '200':
          description: callback successfully processed`

	right := `'{$request.query.queryUrl}':
    post:
      requestBody:
        description: Callback payload
        content: 
          'application/json':
            schema:
              type: int
      responses:
        '200':
          description: callback successfully processed
'slippers':
    post:
      description: toasty toes`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Callback
	var rDoc v3.Callback
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareCallback(&lDoc, &rDoc)
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, extChanges.Changes[0].ChangeType)
	assert.Equal(t, "slippers", extChanges.Changes[0].Property)
}

func TestCompareCallback_Modify(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `x-pizza: tasty
'{$request.query.queryUrl}':
    post:
      requestBody:
        description: Callback payload
        content: 
          'application/json':
            schema:
              type: int
      responses:
        '200':
          description: callback successfully processed`

	right := `x-pizza: cold
'{$request.query.queryUrl}':
    get:
      description: a nice new thing, for the things.
    post:
      requestBody:
        description: Callback payload
        content: 
          'application/json':
            schema:
              type: int
      responses:
        '200':
          description: callback successfully processed`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Callback
	var rDoc v3.Callback
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareCallback(&lDoc, &rDoc)
	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 2)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, PropertyAdded, extChanges.ExpressionChanges["{$request.query.queryUrl}"].Changes[0].ChangeType)
	assert.Equal(t, v3.GetLabel, extChanges.ExpressionChanges["{$request.query.queryUrl}"].Changes[0].Property)
}

func TestCompareCallback_Remove(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `'{$request.query.queryUrl}':
    post:
      requestBody:
        description: Callback payload
        content:
          'application/json':
            schema:
              type: int
      responses:
        '200':
          description: callback successfully processed`

	right := `'{$request.query.queryUrl}':
    post:
      requestBody:
        description: Callback payload
        content:
          'application/json':
            schema:
              type: int
      responses:
        '200':
          description: callback successfully processed
'slippers':
    post:
      description: toasty toes`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Callback
	var rDoc v3.Callback
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareCallback(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, "slippers", extChanges.Changes[0].Property)
}

// Test nil handling - both nil
func TestCompareCallback_BothNil(t *testing.T) {
	low.ClearHashCache()
	extChanges := CompareCallback(nil, nil)
	assert.Nil(t, extChanges)
}

// Test nil handling - callback added (left is nil)
func TestCompareCallback_Added_LeftNil(t *testing.T) {
	low.ClearHashCache()
	right := `'{$request.query.queryUrl}':
    post:
      requestBody:
        description: Callback payload
      responses:
        '200':
          description: callback successfully processed
'slippers':
    post:
      description: toasty toes`

	var rNode yaml.Node
	_ = yaml.Unmarshal([]byte(right), &rNode)

	var rDoc v3.Callback
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare with nil left - callback was added
	extChanges := CompareCallback(nil, &rDoc)
	assert.NotNil(t, extChanges)
	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 2)
	// Both expressions should be added
	for _, change := range extChanges.Changes {
		assert.Equal(t, ObjectAdded, change.ChangeType)
	}
}

// Test nil handling - callback removed (right is nil)
func TestCompareCallback_Removed_RightNil(t *testing.T) {
	low.ClearHashCache()
	left := `'{$request.query.queryUrl}':
    post:
      requestBody:
        description: Callback payload
      responses:
        '200':
          description: callback successfully processed
'slippers':
    post:
      description: toasty toes`

	var lNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)

	var lDoc v3.Callback
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)

	// compare with nil right - callback was removed
	extChanges := CompareCallback(&lDoc, nil)
	assert.NotNil(t, extChanges)
	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Equal(t, 2, extChanges.TotalBreakingChanges()) // Removals are breaking
	assert.Len(t, extChanges.GetAllChanges(), 2)
	// Both expressions should be removed
	for _, change := range extChanges.Changes {
		assert.Equal(t, ObjectRemoved, change.ChangeType)
	}
}

// Test nil handling - added callback with extensions
func TestCompareCallback_Added_WithExtensions(t *testing.T) {
	low.ClearHashCache()
	right := `x-custom: value
'{$request.query.queryUrl}':
    post:
      description: Callback handler`

	var rNode yaml.Node
	_ = yaml.Unmarshal([]byte(right), &rNode)

	var rDoc v3.Callback
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare with nil left - callback was added
	extChanges := CompareCallback(nil, &rDoc)
	assert.NotNil(t, extChanges)
	// Should have 1 expression added + 1 extension added
	assert.Equal(t, 2, extChanges.TotalChanges())
}

// Test nil handling - removed callback with extensions
func TestCompareCallback_Removed_WithExtensions(t *testing.T) {
	low.ClearHashCache()
	left := `x-custom: value
'{$request.query.queryUrl}':
    post:
      description: Callback handler`

	var lNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)

	var lDoc v3.Callback
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)

	// compare with nil right - callback was removed
	extChanges := CompareCallback(&lDoc, nil)
	assert.NotNil(t, extChanges)
	// Should have 1 expression removed + 1 extension removed
	assert.Equal(t, 2, extChanges.TotalChanges())
}

// Test configurable breaking rules for added callbacks
func TestCompareCallback_Added_ConfigurableBreakingRules(t *testing.T) {
	// Reset state for clean test
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	low.ClearHashCache()
	defer func() {
		ResetActiveBreakingRulesConfig()
		ResetDefaultBreakingRules()
	}()

	right := `'{$request.query.queryUrl}':
    post:
      description: Callback handler`

	var rNode yaml.Node
	_ = yaml.Unmarshal([]byte(right), &rNode)

	var rDoc v3.Callback
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// Test 1: With default config, callbacks added should NOT be breaking
	extChanges := CompareCallback(nil, &rDoc)
	assert.NotNil(t, extChanges)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges(), "default config: callbacks.added should not be breaking")

	// Test 2: Set custom config with operation.callbacks.added = true
	customConfig := &BreakingRulesConfig{
		Operation: &OperationRules{
			Callbacks: &BreakingChangeRule{
				Added: boolPtr(true),
			},
		},
	}
	SetActiveBreakingRulesConfig(customConfig)
	low.ClearHashCache()

	// Re-compare with custom config - should now be breaking
	extChanges2 := CompareCallback(nil, &rDoc)
	assert.NotNil(t, extChanges2)
	assert.Equal(t, 1, extChanges2.TotalChanges())
	assert.Equal(t, 1, extChanges2.TotalBreakingChanges(), "custom config: callbacks.added should be breaking")
}

// Test configurable breaking rules for removed callbacks
func TestCompareCallback_Removed_ConfigurableBreakingRules(t *testing.T) {
	// Reset state for clean test
	ResetDefaultBreakingRules()
	ResetActiveBreakingRulesConfig()
	low.ClearHashCache()
	defer func() {
		ResetActiveBreakingRulesConfig()
		ResetDefaultBreakingRules()
	}()

	left := `'{$request.query.queryUrl}':
    post:
      description: Callback handler`

	var lNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)

	var lDoc v3.Callback
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)

	// Test 1: With default config, callbacks removed SHOULD be breaking
	extChanges := CompareCallback(&lDoc, nil)
	assert.NotNil(t, extChanges)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges(), "default config: callbacks.removed should be breaking")

	// Test 2: Set custom config with operation.callbacks.removed = false
	customConfig := &BreakingRulesConfig{
		Operation: &OperationRules{
			Callbacks: &BreakingChangeRule{
				Removed: boolPtr(false),
			},
		},
	}
	SetActiveBreakingRulesConfig(customConfig)
	low.ClearHashCache()

	// Re-compare with custom config - should now NOT be breaking
	extChanges2 := CompareCallback(&lDoc, nil)
	assert.NotNil(t, extChanges2)
	assert.Equal(t, 1, extChanges2.TotalChanges())
	assert.Equal(t, 0, extChanges2.TotalBreakingChanges(), "custom config: callbacks.removed should not be breaking")
}

// Test edge case: empty callback with only extensions (no expressions)
func TestCompareCallback_Added_OnlyExtensions(t *testing.T) {
	low.ClearHashCache()
	defer low.ClearHashCache()

	// Callback with only extension, no expressions
	right := `x-custom: value`

	var rNode yaml.Node
	_ = yaml.Unmarshal([]byte(right), &rNode)

	var rDoc v3.Callback
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// Compare with nil left - callback was added but has no expressions
	extChanges := CompareCallback(nil, &rDoc)
	// Should have 1 extension change
	assert.NotNil(t, extChanges)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.NotNil(t, extChanges.ExtensionChanges)
}

// Test edge case: empty callback removed with only extensions (no expressions)
func TestCompareCallback_Removed_OnlyExtensions(t *testing.T) {
	low.ClearHashCache()
	defer low.ClearHashCache()

	// Callback with only extension, no expressions
	left := `x-custom: value`

	var lNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)

	var lDoc v3.Callback
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)

	// Compare with nil right - callback was removed but has no expressions
	extChanges := CompareCallback(&lDoc, nil)
	// Should have 1 extension change
	assert.NotNil(t, extChanges)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.NotNil(t, extChanges.ExtensionChanges)
}

// Test edge case: truly empty callback (no expressions, no extensions) returns nil
func TestCompareCallback_Added_TrulyEmpty(t *testing.T) {
	low.ClearHashCache()
	defer low.ClearHashCache()

	// Empty callback - this is unusual but tests the TotalChanges() <= 0 path
	right := `{}`

	var rNode yaml.Node
	_ = yaml.Unmarshal([]byte(right), &rNode)

	var rDoc v3.Callback
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// Compare with nil left - truly empty callback
	extChanges := CompareCallback(nil, &rDoc)
	// Should return nil since there are no changes to report
	assert.Nil(t, extChanges)
}

// Test edge case: truly empty callback removed returns nil
func TestCompareCallback_Removed_TrulyEmpty(t *testing.T) {
	low.ClearHashCache()
	defer low.ClearHashCache()

	// Empty callback
	left := `{}`

	var lNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)

	var lDoc v3.Callback
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)

	// Compare with nil right - truly empty callback
	extChanges := CompareCallback(&lDoc, nil)
	// Should return nil since there are no changes to report
	assert.Nil(t, extChanges)
}
