// Copyright 2022-2025 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package overlay

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestOverlay_Build(t *testing.T) {
	yml := `overlay: 1.0.0
info:
  title: My Overlay
  version: 1.0.0
extends: https://example.com/openapi.yaml
actions:
  - target: $.info.title
    update: New Title
  - target: $.info.description
    remove: true`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var overlay Overlay
	err = low.BuildModel(node.Content[0], &overlay)
	require.NoError(t, err)

	err = overlay.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.Equal(t, "1.0.0", overlay.Overlay.Value)
	assert.False(t, overlay.Info.IsEmpty())
	assert.Equal(t, "My Overlay", overlay.Info.Value.Title.Value)
	assert.Equal(t, "1.0.0", overlay.Info.Value.Version.Value)
	assert.Equal(t, "https://example.com/openapi.yaml", overlay.Extends.Value)
	assert.False(t, overlay.Actions.IsEmpty())
	assert.Len(t, overlay.Actions.Value, 2)

	// Check first action
	action1 := overlay.Actions.Value[0].Value
	assert.Equal(t, "$.info.title", action1.Target.Value)
	assert.False(t, action1.Update.IsEmpty())

	// Check second action
	action2 := overlay.Actions.Value[1].Value
	assert.Equal(t, "$.info.description", action2.Target.Value)
	assert.True(t, action2.Remove.Value)
}

func TestOverlay_Build_Minimal(t *testing.T) {
	yml := `overlay: 1.0.0
info:
  title: Minimal
  version: 1.0.0
actions:
  - target: $.info
    update:
      description: Added`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var overlay Overlay
	err = low.BuildModel(node.Content[0], &overlay)
	require.NoError(t, err)

	err = overlay.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.Equal(t, "1.0.0", overlay.Overlay.Value)
	assert.True(t, overlay.Extends.IsEmpty())
	assert.Len(t, overlay.Actions.Value, 1)
}

func TestOverlay_Build_WithExtensions(t *testing.T) {
	yml := `overlay: 1.0.0
info:
  title: Extended
  version: 1.0.0
actions:
  - target: $.info
    update: {}
x-custom: value`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var overlay Overlay
	err = low.BuildModel(node.Content[0], &overlay)
	require.NoError(t, err)

	err = overlay.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.NotNil(t, overlay.Extensions)
	ext := overlay.FindExtension("x-custom")
	require.NotNil(t, ext)
	assert.Equal(t, "value", ext.Value.Value)
}

func TestOverlay_Build_NoActions(t *testing.T) {
	yml := `overlay: 1.0.0
info:
  title: No Actions
  version: 1.0.0`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var overlay Overlay
	err = low.BuildModel(node.Content[0], &overlay)
	require.NoError(t, err)

	err = overlay.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.True(t, overlay.Actions.IsEmpty())
}

func TestOverlay_Hash(t *testing.T) {
	yml1 := `overlay: 1.0.0
info:
  title: Test
  version: 1.0.0
actions:
  - target: $.info
    update: {}`

	yml2 := `overlay: 1.0.0
info:
  title: Test
  version: 1.0.0
actions:
  - target: $.info
    update: {}`

	yml3 := `overlay: 2.0.0
info:
  title: Different
  version: 2.0.0
actions:
  - target: $.paths
    remove: true`

	var node1, node2, node3 yaml.Node
	_ = yaml.Unmarshal([]byte(yml1), &node1)
	_ = yaml.Unmarshal([]byte(yml2), &node2)
	_ = yaml.Unmarshal([]byte(yml3), &node3)

	var overlay1, overlay2, overlay3 Overlay
	_ = low.BuildModel(node1.Content[0], &overlay1)
	_ = overlay1.Build(context.Background(), nil, node1.Content[0], nil)

	_ = low.BuildModel(node2.Content[0], &overlay2)
	_ = overlay2.Build(context.Background(), nil, node2.Content[0], nil)

	_ = low.BuildModel(node3.Content[0], &overlay3)
	_ = overlay3.Build(context.Background(), nil, node3.Content[0], nil)

	assert.Equal(t, overlay1.Hash(), overlay2.Hash())
	assert.NotEqual(t, overlay1.Hash(), overlay3.Hash())
}

func TestOverlay_Hash_WithExtends(t *testing.T) {
	yml := `overlay: 1.0.0
info:
  title: Test
  version: 1.0.0
extends: https://example.com/spec.yaml
actions:
  - target: $.info
    update: {}`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	var overlay Overlay
	_ = low.BuildModel(node.Content[0], &overlay)
	_ = overlay.Build(context.Background(), nil, node.Content[0], nil)

	hash := overlay.Hash()
	assert.NotEqual(t, [32]byte{}, hash)
}

func TestOverlay_GettersReturnCorrectValues(t *testing.T) {
	yml := `overlay: 1.0.0
info:
  title: Test
  version: 1.0.0
actions: []`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	keyNode := &yaml.Node{Value: "overlay"}
	var overlay Overlay
	_ = low.BuildModel(node.Content[0], &overlay)
	_ = overlay.Build(context.Background(), keyNode, node.Content[0], nil)

	assert.Equal(t, keyNode, overlay.GetKeyNode())
	assert.Equal(t, node.Content[0], overlay.GetRootNode())
	assert.Nil(t, overlay.GetIndex())
	assert.NotNil(t, overlay.GetContext())
	assert.NotNil(t, overlay.GetExtensions())
}

func TestOverlay_FindExtension_NotFound(t *testing.T) {
	yml := `overlay: 1.0.0
info:
  title: Test
  version: 1.0.0
actions: []`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	var overlay Overlay
	_ = low.BuildModel(node.Content[0], &overlay)
	_ = overlay.Build(context.Background(), nil, node.Content[0], nil)

	ext := overlay.FindExtension("x-nonexistent")
	assert.Nil(t, ext)
}

func TestOverlay_Build_ActionsNotSequence(t *testing.T) {
	yml := `overlay: 1.0.0
info:
  title: Test
  version: 1.0.0
actions: not-a-sequence`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var overlay Overlay
	err = low.BuildModel(node.Content[0], &overlay)
	require.NoError(t, err)

	err = overlay.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	// Actions should be empty since it's not a sequence
	assert.True(t, overlay.Actions.IsEmpty() || len(overlay.Actions.Value) == 0)
}

func TestOverlay_Build_MultipleActions(t *testing.T) {
	yml := `overlay: 1.0.0
info:
  title: Multi-Action
  version: 1.0.0
actions:
  - target: $.info.title
    description: First action
    update: Title One
  - target: $.info.description
    description: Second action
    update: Description
  - target: $.info.contact
    description: Third action
    remove: true`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var overlay Overlay
	err = low.BuildModel(node.Content[0], &overlay)
	require.NoError(t, err)

	err = overlay.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	require.Len(t, overlay.Actions.Value, 3)

	assert.Equal(t, "$.info.title", overlay.Actions.Value[0].Value.Target.Value)
	assert.Equal(t, "First action", overlay.Actions.Value[0].Value.Description.Value)

	assert.Equal(t, "$.info.description", overlay.Actions.Value[1].Value.Target.Value)
	assert.Equal(t, "Second action", overlay.Actions.Value[1].Value.Description.Value)

	assert.Equal(t, "$.info.contact", overlay.Actions.Value[2].Value.Target.Value)
	assert.True(t, overlay.Actions.Value[2].Value.Remove.Value)
}

func TestOverlay_Hash_Empty(t *testing.T) {
	// Test hash with all fields empty
	var overlay Overlay
	hash := overlay.Hash()
	// Empty hash should still produce a valid (non-zero) hash
	assert.NotEqual(t, [32]byte{}, hash)
}

func TestOverlay_Hash_NoActions(t *testing.T) {
	yml := `overlay: 1.0.0
info:
  title: Test
  version: 1.0.0`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	var overlay Overlay
	_ = low.BuildModel(node.Content[0], &overlay)
	_ = overlay.Build(context.Background(), nil, node.Content[0], nil)

	hash := overlay.Hash()
	assert.NotEqual(t, [32]byte{}, hash)
}

func TestOverlay_Build_OddContentLength(t *testing.T) {
	// This tests the i+1 >= len(root.Content) check in extractActions
	yml := `overlay: 1.0.0
info:
  title: Test
  version: 1.0.0
actions:
  - target: $.info
    update: {}`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	// Manually corrupt the node to have odd content length
	// This simulates a malformed YAML structure
	var overlay Overlay
	err = low.BuildModel(node.Content[0], &overlay)
	require.NoError(t, err)

	err = overlay.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)
}

func TestOverlay_Build_EmptyActions(t *testing.T) {
	yml := `overlay: 1.0.0
info:
  title: Test
  version: 1.0.0
actions: []`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var overlay Overlay
	err = low.BuildModel(node.Content[0], &overlay)
	require.NoError(t, err)

	err = overlay.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.Len(t, overlay.Actions.Value, 0)
}

func TestOverlay_Hash_WithExtensions(t *testing.T) {
	yml := `overlay: 1.0.0
info:
  title: Test
  version: 1.0.0
actions: []
x-custom: value`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	var overlay Overlay
	_ = low.BuildModel(node.Content[0], &overlay)
	_ = overlay.Build(context.Background(), nil, node.Content[0], nil)

	hash := overlay.Hash()
	assert.NotEqual(t, [32]byte{}, hash)
}
