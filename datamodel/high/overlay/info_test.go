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

func TestNewInfo(t *testing.T) {
	yml := `title: My Overlay
version: 1.0.0
x-custom: value`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var lowInfo lowoverlay.Info
	err = low.BuildModel(node.Content[0], &lowInfo)
	require.NoError(t, err)
	err = lowInfo.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	highInfo := NewInfo(&lowInfo)

	assert.Equal(t, "My Overlay", highInfo.Title)
	assert.Equal(t, "1.0.0", highInfo.Version)
	assert.NotNil(t, highInfo.Extensions)
	assert.Equal(t, 1, highInfo.Extensions.Len())
}

func TestNewInfo_Minimal(t *testing.T) {
	yml := `title: Minimal`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var lowInfo lowoverlay.Info
	err = low.BuildModel(node.Content[0], &lowInfo)
	require.NoError(t, err)
	err = lowInfo.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	highInfo := NewInfo(&lowInfo)

	assert.Equal(t, "Minimal", highInfo.Title)
	assert.Empty(t, highInfo.Version)
}

func TestInfo_GoLow(t *testing.T) {
	yml := `title: Test`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	var lowInfo lowoverlay.Info
	_ = low.BuildModel(node.Content[0], &lowInfo)
	_ = lowInfo.Build(context.Background(), nil, node.Content[0], nil)

	highInfo := NewInfo(&lowInfo)

	assert.Equal(t, &lowInfo, highInfo.GoLow())
	assert.Equal(t, &lowInfo, highInfo.GoLowUntyped())
}

func TestInfo_Render(t *testing.T) {
	yml := `title: My Overlay
version: 1.0.0`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	var lowInfo lowoverlay.Info
	_ = low.BuildModel(node.Content[0], &lowInfo)
	_ = lowInfo.Build(context.Background(), nil, node.Content[0], nil)

	highInfo := NewInfo(&lowInfo)

	rendered, err := highInfo.Render()
	require.NoError(t, err)
	assert.Contains(t, string(rendered), "title: My Overlay")
	assert.Contains(t, string(rendered), "version: 1.0.0")
}

func TestInfo_MarshalYAML(t *testing.T) {
	yml := `title: Test
version: 2.0.0
x-custom: value`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	var lowInfo lowoverlay.Info
	_ = low.BuildModel(node.Content[0], &lowInfo)
	_ = lowInfo.Build(context.Background(), nil, node.Content[0], nil)

	highInfo := NewInfo(&lowInfo)

	result, err := highInfo.MarshalYAML()
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestInfo_MarshalYAML_Empty(t *testing.T) {
	var lowInfo lowoverlay.Info
	highInfo := NewInfo(&lowInfo)

	result, err := highInfo.MarshalYAML()
	require.NoError(t, err)
	assert.NotNil(t, result)
}
