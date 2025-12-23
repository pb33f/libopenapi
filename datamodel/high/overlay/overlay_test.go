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

func TestNewOverlay(t *testing.T) {
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

	var lowOverlay lowoverlay.Overlay
	err = low.BuildModel(node.Content[0], &lowOverlay)
	require.NoError(t, err)
	err = lowOverlay.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	highOverlay := NewOverlay(&lowOverlay)

	assert.Equal(t, "1.0.0", highOverlay.Overlay)
	assert.NotNil(t, highOverlay.Info)
	assert.Equal(t, "My Overlay", highOverlay.Info.Title)
	assert.Equal(t, "1.0.0", highOverlay.Info.Version)
	assert.Equal(t, "https://example.com/openapi.yaml", highOverlay.Extends)
	assert.Len(t, highOverlay.Actions, 2)
	assert.Equal(t, "$.info.title", highOverlay.Actions[0].Target)
	assert.Equal(t, "$.info.description", highOverlay.Actions[1].Target)
}

func TestNewOverlay_Minimal(t *testing.T) {
	yml := `overlay: 1.0.0
info:
  title: Minimal
  version: 1.0.0
actions:
  - target: $.info
    update: {}`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var lowOverlay lowoverlay.Overlay
	err = low.BuildModel(node.Content[0], &lowOverlay)
	require.NoError(t, err)
	err = lowOverlay.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	highOverlay := NewOverlay(&lowOverlay)

	assert.Equal(t, "1.0.0", highOverlay.Overlay)
	assert.Empty(t, highOverlay.Extends)
	assert.Len(t, highOverlay.Actions, 1)
}

func TestNewOverlay_NoActions(t *testing.T) {
	yml := `overlay: 1.0.0
info:
  title: No Actions
  version: 1.0.0`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var lowOverlay lowoverlay.Overlay
	err = low.BuildModel(node.Content[0], &lowOverlay)
	require.NoError(t, err)
	err = lowOverlay.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	highOverlay := NewOverlay(&lowOverlay)

	assert.Empty(t, highOverlay.Actions)
}

func TestNewOverlay_WithExtensions(t *testing.T) {
	yml := `overlay: 1.0.0
info:
  title: Extended
  version: 1.0.0
actions: []
x-custom: value`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var lowOverlay lowoverlay.Overlay
	err = low.BuildModel(node.Content[0], &lowOverlay)
	require.NoError(t, err)
	err = lowOverlay.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	highOverlay := NewOverlay(&lowOverlay)

	assert.NotNil(t, highOverlay.Extensions)
	assert.Equal(t, 1, highOverlay.Extensions.Len())
}

func TestOverlay_GoLow(t *testing.T) {
	yml := `overlay: 1.0.0
info:
  title: Test
  version: 1.0.0
actions: []`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	var lowOverlay lowoverlay.Overlay
	_ = low.BuildModel(node.Content[0], &lowOverlay)
	_ = lowOverlay.Build(context.Background(), nil, node.Content[0], nil)

	highOverlay := NewOverlay(&lowOverlay)

	assert.Equal(t, &lowOverlay, highOverlay.GoLow())
	assert.Equal(t, &lowOverlay, highOverlay.GoLowUntyped())
}

func TestOverlay_Render(t *testing.T) {
	yml := `overlay: 1.0.0
info:
  title: My Overlay
  version: 1.0.0
actions:
  - target: $.info
    update: {}`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	var lowOverlay lowoverlay.Overlay
	_ = low.BuildModel(node.Content[0], &lowOverlay)
	_ = lowOverlay.Build(context.Background(), nil, node.Content[0], nil)

	highOverlay := NewOverlay(&lowOverlay)

	rendered, err := highOverlay.Render()
	require.NoError(t, err)
	assert.Contains(t, string(rendered), "overlay: 1.0.0")
}

func TestOverlay_MarshalYAML(t *testing.T) {
	yml := `overlay: 1.0.0
info:
  title: Test
  version: 1.0.0
extends: https://example.com/spec.yaml
actions:
  - target: $.info
    update: {}
x-custom: value`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	var lowOverlay lowoverlay.Overlay
	_ = low.BuildModel(node.Content[0], &lowOverlay)
	_ = lowOverlay.Build(context.Background(), nil, node.Content[0], nil)

	highOverlay := NewOverlay(&lowOverlay)

	result, err := highOverlay.MarshalYAML()
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestOverlay_MarshalYAML_Empty(t *testing.T) {
	var lowOverlay lowoverlay.Overlay
	highOverlay := NewOverlay(&lowOverlay)

	result, err := highOverlay.MarshalYAML()
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestOverlay_MarshalYAML_NoInfo(t *testing.T) {
	yml := `overlay: 1.0.0
actions: []`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	var lowOverlay lowoverlay.Overlay
	_ = low.BuildModel(node.Content[0], &lowOverlay)
	_ = lowOverlay.Build(context.Background(), nil, node.Content[0], nil)

	highOverlay := NewOverlay(&lowOverlay)

	result, err := highOverlay.MarshalYAML()
	require.NoError(t, err)
	assert.NotNil(t, result)
}
