// Copyright 2022-2025 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package overlay

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	highoverlay "github.com/pb33f/libopenapi/datamodel/high/overlay"
	lowoverlay "github.com/pb33f/libopenapi/datamodel/low/overlay"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func parseOverlay(t *testing.T, yml string) *highoverlay.Overlay {
	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var lowOv lowoverlay.Overlay
	err = low.BuildModel(node.Content[0], &lowOv)
	require.NoError(t, err)
	err = lowOv.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	return highoverlay.NewOverlay(&lowOv)
}

func TestApply_UpdateTitle(t *testing.T) {
	targetYAML := `openapi: 3.0.0
info:
  title: Original Title
  version: 1.0.0
paths: {}`

	overlayYAML := `overlay: 1.0.0
info:
  title: Test Overlay
  version: 1.0.0
actions:
  - target: $.info
    update:
      title: Updated Title`

	overlay := parseOverlay(t, overlayYAML)

	result, err := Apply([]byte(targetYAML), overlay)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, string(result.Bytes), "Updated Title")
	assert.Len(t, result.Warnings, 0)
}

func TestApply_RemoveDescription(t *testing.T) {
	targetYAML := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
  description: This should be removed
paths: {}`

	overlayYAML := `overlay: 1.0.0
info:
  title: Test Overlay
  version: 1.0.0
actions:
  - target: $.info.description
    remove: true`

	overlay := parseOverlay(t, overlayYAML)

	result, err := Apply([]byte(targetYAML), overlay)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotContains(t, string(result.Bytes), "This should be removed")
}

func TestApply_AddDescription(t *testing.T) {
	targetYAML := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
paths: {}`

	overlayYAML := `overlay: 1.0.0
info:
  title: Test Overlay
  version: 1.0.0
actions:
  - target: $.info
    update:
      description: Added description`

	overlay := parseOverlay(t, overlayYAML)

	result, err := Apply([]byte(targetYAML), overlay)
	require.NoError(t, err)
	assert.Contains(t, string(result.Bytes), "Added description")
}

func TestApply_MultipleActions(t *testing.T) {
	targetYAML := `openapi: 3.0.0
info:
  title: Original
  version: 1.0.0
  description: Remove me
paths: {}`

	overlayYAML := `overlay: 1.0.0
info:
  title: Test Overlay
  version: 1.0.0
actions:
  - target: $.info
    update:
      title: Updated
  - target: $.info.description
    remove: true`

	overlay := parseOverlay(t, overlayYAML)

	result, err := Apply([]byte(targetYAML), overlay)
	require.NoError(t, err)
	assert.Contains(t, string(result.Bytes), "Updated")
	assert.NotContains(t, string(result.Bytes), "Remove me")
}

func TestApply_NoMatchWarning(t *testing.T) {
	targetYAML := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
paths: {}`

	overlayYAML := `overlay: 1.0.0
info:
  title: Test Overlay
  version: 1.0.0
actions:
  - target: $.nonexistent
    update:
      value: test`

	overlay := parseOverlay(t, overlayYAML)

	result, err := Apply([]byte(targetYAML), overlay)
	require.NoError(t, err)
	assert.Len(t, result.Warnings, 1)
	assert.Equal(t, "$.nonexistent", result.Warnings[0].Target)
	assert.Contains(t, result.Warnings[0].Message, "zero nodes")
}

func TestApply_NilOverlay(t *testing.T) {
	targetYAML := `openapi: 3.0.0`

	result, err := Apply([]byte(targetYAML), nil)
	assert.ErrorIs(t, err, ErrInvalidOverlay)
	assert.Nil(t, result)
}

func TestApply_MissingOverlayField(t *testing.T) {
	targetYAML := `openapi: 3.0.0`

	// Create overlay without the overlay field
	overlay := &highoverlay.Overlay{
		Info: &highoverlay.Info{
			Title:   "Test",
			Version: "1.0.0",
		},
		Actions: []*highoverlay.Action{
			{Target: "$.info"},
		},
	}

	result, err := Apply([]byte(targetYAML), overlay)
	assert.ErrorIs(t, err, ErrMissingOverlayField)
	assert.Nil(t, result)
}

func TestApply_MissingInfo(t *testing.T) {
	targetYAML := `openapi: 3.0.0`

	overlay := &highoverlay.Overlay{
		Overlay: "1.0.0",
		Actions: []*highoverlay.Action{
			{Target: "$.info"},
		},
	}

	result, err := Apply([]byte(targetYAML), overlay)
	assert.ErrorIs(t, err, ErrMissingInfo)
	assert.Nil(t, result)
}

func TestApply_EmptyActions(t *testing.T) {
	targetYAML := `openapi: 3.0.0`

	overlay := &highoverlay.Overlay{
		Overlay: "1.0.0",
		Info: &highoverlay.Info{
			Title:   "Test",
			Version: "1.0.0",
		},
		Actions: []*highoverlay.Action{},
	}

	result, err := Apply([]byte(targetYAML), overlay)
	assert.ErrorIs(t, err, ErrEmptyActions)
	assert.Nil(t, result)
}

func TestApply_InvalidTarget_UpdateScalar(t *testing.T) {
	targetYAML := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
paths: {}`

	// Create overlay that tries to update a scalar value with an object
	// This is invalid because you can't merge an object into a scalar
	updateNode := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "key"},
			{Kind: yaml.ScalarNode, Value: "value"},
		},
	}

	overlay := &highoverlay.Overlay{
		Overlay: "1.0.0",
		Info: &highoverlay.Info{
			Title:   "Test",
			Version: "1.0.0",
		},
		Actions: []*highoverlay.Action{
			{Target: "$.info.title", Update: updateNode},
		},
	}

	result, err := Apply([]byte(targetYAML), overlay)
	// $.info.title points to a scalar, which is invalid for update
	assert.ErrorIs(t, err, ErrPrimitiveTarget)
	assert.Nil(t, result)
}

func TestApply_InvalidYAML(t *testing.T) {
	targetYAML := `invalid: yaml: content:`

	overlay := &highoverlay.Overlay{
		Overlay: "1.0.0",
		Info: &highoverlay.Info{
			Title:   "Test",
			Version: "1.0.0",
		},
		Actions: []*highoverlay.Action{
			{Target: "$.info"},
		},
	}

	result, err := Apply([]byte(targetYAML), overlay)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestApply_EmptyTarget(t *testing.T) {
	targetYAML := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0`

	overlay := &highoverlay.Overlay{
		Overlay: "1.0.0",
		Info: &highoverlay.Info{
			Title:   "Test",
			Version: "1.0.0",
		},
		Actions: []*highoverlay.Action{
			{Target: "", Update: &yaml.Node{Kind: yaml.ScalarNode, Value: "test"}},
		},
	}

	result, err := Apply([]byte(targetYAML), overlay)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestApply_DeepMerge(t *testing.T) {
	targetYAML := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
  contact:
    name: Original Name`

	overlayYAML := `overlay: 1.0.0
info:
  title: Test Overlay
  version: 1.0.0
actions:
  - target: $.info.contact
    update:
      email: new@example.com`

	overlay := parseOverlay(t, overlayYAML)

	result, err := Apply([]byte(targetYAML), overlay)
	require.NoError(t, err)
	// Should have both original name and new email
	assert.Contains(t, string(result.Bytes), "Original Name")
	assert.Contains(t, string(result.Bytes), "new@example.com")
}

func TestApply_ArrayAppend(t *testing.T) {
	targetYAML := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
tags:
  - name: existing`

	overlayYAML := `overlay: 1.0.0
info:
  title: Test Overlay
  version: 1.0.0
actions:
  - target: $.tags
    update:
      - name: new-tag`

	overlay := parseOverlay(t, overlayYAML)

	result, err := Apply([]byte(targetYAML), overlay)
	require.NoError(t, err)
	assert.Contains(t, string(result.Bytes), "existing")
	assert.Contains(t, string(result.Bytes), "new-tag")
}

func TestWarning_String(t *testing.T) {
	w := &Warning{
		Target:  "$.info.title",
		Message: "test message",
	}
	assert.Contains(t, w.String(), "$.info.title")
	assert.Contains(t, w.String(), "test message")
}

func TestOverlayError_Error(t *testing.T) {
	action := &highoverlay.Action{Target: "$.test"}
	err := &OverlayError{
		Action: action,
		Cause:  ErrPrimitiveTarget,
	}
	assert.Contains(t, err.Error(), "$.test")
	assert.Contains(t, err.Error(), "primitive")
}

func TestOverlayError_Error_NoAction(t *testing.T) {
	err := &OverlayError{
		Cause: ErrInvalidOverlay,
	}
	assert.Contains(t, err.Error(), "overlay error")
}

func TestOverlayError_Unwrap(t *testing.T) {
	err := &OverlayError{
		Cause: ErrPrimitiveTarget,
	}
	assert.ErrorIs(t, err, ErrPrimitiveTarget)
}

func TestApply_RemoveFromSequence(t *testing.T) {
	targetYAML := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
tags:
  - name: first
  - name: second
  - name: third`

	overlayYAML := `overlay: 1.0.0
info:
  title: Test Overlay
  version: 1.0.0
actions:
  - target: $.tags[1]
    remove: true`

	overlay := parseOverlay(t, overlayYAML)

	result, err := Apply([]byte(targetYAML), overlay)
	require.NoError(t, err)
	assert.Contains(t, string(result.Bytes), "first")
	assert.NotContains(t, string(result.Bytes), "second")
	assert.Contains(t, string(result.Bytes), "third")
}

func TestApply_RemoveKey(t *testing.T) {
	targetYAML := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
  contact:
    name: John
    email: john@example.com`

	// Test removing a value (contact)
	overlayYAML := `overlay: 1.0.0
info:
  title: Test Overlay
  version: 1.0.0
actions:
  - target: $.info.contact
    remove: true`

	overlay := parseOverlay(t, overlayYAML)

	result, err := Apply([]byte(targetYAML), overlay)
	require.NoError(t, err)
	assert.NotContains(t, string(result.Bytes), "contact")
	assert.NotContains(t, string(result.Bytes), "John")
}

func TestApply_UpdateWithDifferentKind(t *testing.T) {
	targetYAML := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
  contact:
    name: John`

	// Replace the contact object with a different structure
	overlayYAML := `overlay: 1.0.0
info:
  title: Test Overlay
  version: 1.0.0
actions:
  - target: $.info.contact
    update:
      email: new@example.com
      url: https://example.com`

	overlay := parseOverlay(t, overlayYAML)

	result, err := Apply([]byte(targetYAML), overlay)
	require.NoError(t, err)
	// Should have both old and new properties (merge)
	assert.Contains(t, string(result.Bytes), "John")
	assert.Contains(t, string(result.Bytes), "new@example.com")
}

func TestApply_UpdateScalarValue(t *testing.T) {
	targetYAML := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0`

	// This tests the mergeNode default case where node types match but aren't mapping/sequence
	overlayYAML := `overlay: 1.0.0
info:
  title: Test Overlay
  version: 1.0.0
actions:
  - target: $.info
    update:
      title: New Title`

	overlay := parseOverlay(t, overlayYAML)

	result, err := Apply([]byte(targetYAML), overlay)
	require.NoError(t, err)
	assert.Contains(t, string(result.Bytes), "New Title")
}

func TestApply_ReplaceWithDifferentType(t *testing.T) {
	targetYAML := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
  contact:
    name: John`

	// Replace an object with a sequence (different node kinds)
	updateNode := &yaml.Node{
		Kind: yaml.SequenceNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "item1"},
			{Kind: yaml.ScalarNode, Value: "item2"},
		},
	}

	overlay := &highoverlay.Overlay{
		Overlay: "1.0.0",
		Info: &highoverlay.Info{
			Title:   "Test",
			Version: "1.0.0",
		},
		Actions: []*highoverlay.Action{
			{Target: "$.info.contact", Update: updateNode},
		},
	}

	result, err := Apply([]byte(targetYAML), overlay)
	require.NoError(t, err)
	// When kinds differ, the entire node is replaced with a clone
	assert.Contains(t, string(result.Bytes), "item1")
	assert.Contains(t, string(result.Bytes), "item2")
	assert.NotContains(t, string(result.Bytes), "John")
}

func TestApply_RemoveNonexistentParent(t *testing.T) {
	targetYAML := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0`

	// Try to remove root (no parent)
	overlay := &highoverlay.Overlay{
		Overlay: "1.0.0",
		Info: &highoverlay.Info{
			Title:   "Test",
			Version: "1.0.0",
		},
		Actions: []*highoverlay.Action{
			{Target: "$", Remove: true},
		},
	}

	result, err := Apply([]byte(targetYAML), overlay)
	require.NoError(t, err)
	// Should complete without error, just doesn't remove root
	assert.NotNil(t, result)
}

func TestApply_EmptyUpdateNode(t *testing.T) {
	targetYAML := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0`

	// Action with empty update
	overlay := &highoverlay.Overlay{
		Overlay: "1.0.0",
		Info: &highoverlay.Info{
			Title:   "Test",
			Version: "1.0.0",
		},
		Actions: []*highoverlay.Action{
			{Target: "$.info", Update: &yaml.Node{}},
		},
	}

	result, err := Apply([]byte(targetYAML), overlay)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCloneNode_WithAlias(t *testing.T) {
	// Create a node with an alias
	alias := &yaml.Node{Kind: yaml.ScalarNode, Value: "aliased"}
	node := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Alias: alias,
	}

	cloned := cloneNode(node)
	assert.NotNil(t, cloned.Alias)
	assert.Equal(t, "aliased", cloned.Alias.Value)
}

func TestApply_InvalidJSONPath(t *testing.T) {
	targetYAML := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0`

	overlay := &highoverlay.Overlay{
		Overlay: "1.0.0",
		Info: &highoverlay.Info{
			Title:   "Test",
			Version: "1.0.0",
		},
		Actions: []*highoverlay.Action{
			{Target: "$..[[[invalid", Update: &yaml.Node{Kind: yaml.ScalarNode, Value: "test"}},
		},
	}

	result, err := Apply([]byte(targetYAML), overlay)
	assert.ErrorIs(t, err, ErrInvalidJSONPath)
	assert.Nil(t, result)
}

func TestRemoveNode_NilParent(t *testing.T) {
	// Test removeNode with a node that has no parent in the index
	// This tests the defensive nil parent check
	orphanNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "orphan"}

	// Create an empty parent index
	idx := parentIndex{}

	// removeNode should safely handle nil parent
	removeNode(idx, orphanNode)

	// No panic or error expected, just a silent no-op
	assert.Equal(t, "orphan", orphanNode.Value)
}

func TestApply_MarshalError(t *testing.T) {
	targetYAML := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0`

	// Create an overlay with an update that contains an invalid node kind
	// yaml.Marshal will fail when trying to marshal a node with kind 99
	invalidNode := &yaml.Node{
		Kind: 99, // Invalid node kind
	}

	overlay := &highoverlay.Overlay{
		Overlay: "1.0.0",
		Info: &highoverlay.Info{
			Title:   "Test",
			Version: "1.0.0",
		},
		Actions: []*highoverlay.Action{
			{Target: "$.info", Update: &yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "contact"},
					invalidNode,
				},
			}},
		},
	}

	result, err := Apply([]byte(targetYAML), overlay)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown kind")
	assert.Nil(t, result)
}

func TestApply_UpdateThenRemove_SequentialActions(t *testing.T) {
	// This test verifies that remove actions can delete nodes added by earlier update actions
	targetYAML := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0`

	// First action adds a description, second action removes it
	overlayYAML := `overlay: 1.0.0
info:
  title: Test Overlay
  version: 1.0.0
actions:
  - target: $.info
    update:
      description: This will be added then removed
  - target: $.info.description
    remove: true`

	overlay := parseOverlay(t, overlayYAML)

	result, err := Apply([]byte(targetYAML), overlay)
	require.NoError(t, err)

	// The description should NOT be in the result because it was removed
	assert.NotContains(t, string(result.Bytes), "This will be added then removed")
	assert.NotContains(t, string(result.Bytes), "description")
}
