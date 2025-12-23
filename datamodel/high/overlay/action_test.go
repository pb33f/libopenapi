// Copyright 2022-2025 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package overlay

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	lowoverlay "github.com/pb33f/libopenapi/datamodel/low/overlay"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestNewAction_Update(t *testing.T) {
	yml := `target: $.info.title
description: Update the title
update: New Title`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var lowAction lowoverlay.Action
	err = low.BuildModel(node.Content[0], &lowAction)
	require.NoError(t, err)
	err = lowAction.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	highAction := NewAction(&lowAction)

	assert.Equal(t, "$.info.title", highAction.Target)
	assert.Equal(t, "Update the title", highAction.Description)
	assert.NotNil(t, highAction.Update)
	assert.Equal(t, "New Title", highAction.Update.Value)
	assert.False(t, highAction.Remove)
}

func TestNewAction_Remove(t *testing.T) {
	yml := `target: $.info.description
remove: true`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var lowAction lowoverlay.Action
	err = low.BuildModel(node.Content[0], &lowAction)
	require.NoError(t, err)
	err = lowAction.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	highAction := NewAction(&lowAction)

	assert.Equal(t, "$.info.description", highAction.Target)
	assert.True(t, highAction.Remove)
}

func TestNewAction_WithExtensions(t *testing.T) {
	yml := `target: $.paths
x-priority: high`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var lowAction lowoverlay.Action
	err = low.BuildModel(node.Content[0], &lowAction)
	require.NoError(t, err)
	err = lowAction.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	highAction := NewAction(&lowAction)

	assert.NotNil(t, highAction.Extensions)
	assert.Equal(t, 1, highAction.Extensions.Len())
}

func TestAction_GoLow(t *testing.T) {
	yml := `target: $.info`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	var lowAction lowoverlay.Action
	_ = low.BuildModel(node.Content[0], &lowAction)
	_ = lowAction.Build(context.Background(), nil, node.Content[0], nil)

	highAction := NewAction(&lowAction)

	assert.Equal(t, &lowAction, highAction.GoLow())
	assert.Equal(t, &lowAction, highAction.GoLowUntyped())
}

func TestAction_Render(t *testing.T) {
	yml := `target: $.info
update:
  title: Test`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	var lowAction lowoverlay.Action
	_ = low.BuildModel(node.Content[0], &lowAction)
	_ = lowAction.Build(context.Background(), nil, node.Content[0], nil)

	highAction := NewAction(&lowAction)

	rendered, err := highAction.Render()
	require.NoError(t, err)
	assert.Contains(t, string(rendered), "target: $.info")
}

func TestAction_MarshalYAML(t *testing.T) {
	yml := `target: $.info
description: Update info
update:
  title: Test
x-custom: value`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	var lowAction lowoverlay.Action
	_ = low.BuildModel(node.Content[0], &lowAction)
	_ = lowAction.Build(context.Background(), nil, node.Content[0], nil)

	highAction := NewAction(&lowAction)

	result, err := highAction.MarshalYAML()
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAction_MarshalYAML_Remove(t *testing.T) {
	yml := `target: $.info
remove: true`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	var lowAction lowoverlay.Action
	_ = low.BuildModel(node.Content[0], &lowAction)
	_ = lowAction.Build(context.Background(), nil, node.Content[0], nil)

	highAction := NewAction(&lowAction)

	result, err := highAction.MarshalYAML()
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAction_MarshalYAML_Empty(t *testing.T) {
	var lowAction lowoverlay.Action
	highAction := NewAction(&lowAction)

	result, err := highAction.MarshalYAML()
	require.NoError(t, err)
	assert.NotNil(t, result)
}
