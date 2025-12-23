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

func TestAction_Build_Update(t *testing.T) {
	yml := `target: $.info.title
description: Update the title
update: New Title`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var action Action
	err = low.BuildModel(node.Content[0], &action)
	require.NoError(t, err)

	err = action.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.Equal(t, "$.info.title", action.Target.Value)
	assert.Equal(t, "Update the title", action.Description.Value)
	assert.False(t, action.Update.IsEmpty())
	assert.Equal(t, "New Title", action.Update.Value.Value)
	assert.True(t, action.Remove.IsEmpty())
}

func TestAction_Build_UpdateObject(t *testing.T) {
	yml := `target: $.info
update:
  title: New Title
  version: 2.0.0`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var action Action
	err = low.BuildModel(node.Content[0], &action)
	require.NoError(t, err)

	err = action.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.Equal(t, "$.info", action.Target.Value)
	assert.False(t, action.Update.IsEmpty())
	assert.Equal(t, yaml.MappingNode, action.Update.Value.Kind)
}

func TestAction_Build_Remove(t *testing.T) {
	yml := `target: $.info.description
remove: true`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var action Action
	err = low.BuildModel(node.Content[0], &action)
	require.NoError(t, err)

	err = action.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.Equal(t, "$.info.description", action.Target.Value)
	assert.True(t, action.Remove.Value)
}

func TestAction_Build_WithExtensions(t *testing.T) {
	yml := `target: $.paths
x-priority: high`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var action Action
	err = low.BuildModel(node.Content[0], &action)
	require.NoError(t, err)

	err = action.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.NotNil(t, action.Extensions)
	ext := action.FindExtension("x-priority")
	require.NotNil(t, ext)
	assert.Equal(t, "high", ext.Value.Value)
}

func TestAction_Hash(t *testing.T) {
	yml1 := `target: $.info
update:
  title: Test`

	yml2 := `target: $.info
update:
  title: Test`

	yml3 := `target: $.paths
remove: true`

	var node1, node2, node3 yaml.Node
	_ = yaml.Unmarshal([]byte(yml1), &node1)
	_ = yaml.Unmarshal([]byte(yml2), &node2)
	_ = yaml.Unmarshal([]byte(yml3), &node3)

	var action1, action2, action3 Action
	_ = low.BuildModel(node1.Content[0], &action1)
	_ = action1.Build(context.Background(), nil, node1.Content[0], nil)

	_ = low.BuildModel(node2.Content[0], &action2)
	_ = action2.Build(context.Background(), nil, node2.Content[0], nil)

	_ = low.BuildModel(node3.Content[0], &action3)
	_ = action3.Build(context.Background(), nil, node3.Content[0], nil)

	assert.Equal(t, action1.Hash(), action2.Hash())
	assert.NotEqual(t, action1.Hash(), action3.Hash())
}

func TestAction_Hash_AllFields(t *testing.T) {
	yml := `target: $.info
description: Update info
update:
  title: Test
x-ext: value`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	var action Action
	_ = low.BuildModel(node.Content[0], &action)
	_ = action.Build(context.Background(), nil, node.Content[0], nil)

	hash := action.Hash()
	assert.NotEqual(t, [32]byte{}, hash)
}

func TestAction_Hash_RemoveFalse(t *testing.T) {
	yml := `target: $.info
remove: false`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	var action Action
	_ = low.BuildModel(node.Content[0], &action)
	_ = action.Build(context.Background(), nil, node.Content[0], nil)

	hash := action.Hash()
	assert.NotEqual(t, [32]byte{}, hash)
}

func TestAction_GettersReturnCorrectValues(t *testing.T) {
	yml := `target: $.info`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	keyNode := &yaml.Node{Value: "action"}
	var action Action
	_ = low.BuildModel(node.Content[0], &action)
	_ = action.Build(context.Background(), keyNode, node.Content[0], nil)

	assert.Equal(t, keyNode, action.GetKeyNode())
	assert.Equal(t, node.Content[0], action.GetRootNode())
	assert.Nil(t, action.GetIndex())
	assert.NotNil(t, action.GetContext())
	assert.NotNil(t, action.GetExtensions())
}

func TestAction_FindExtension_NotFound(t *testing.T) {
	yml := `target: $.info`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	var action Action
	_ = low.BuildModel(node.Content[0], &action)
	_ = action.Build(context.Background(), nil, node.Content[0], nil)

	ext := action.FindExtension("x-nonexistent")
	assert.Nil(t, ext)
}

func TestAction_Build_NoUpdate(t *testing.T) {
	yml := `target: $.info
description: Just a description`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var action Action
	err = low.BuildModel(node.Content[0], &action)
	require.NoError(t, err)

	err = action.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.True(t, action.Update.IsEmpty())
}
