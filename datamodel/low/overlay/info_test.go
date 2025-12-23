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

func TestInfo_Build(t *testing.T) {
	yml := `title: My Overlay
version: 1.0.0
x-custom: value`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var info Info
	err = low.BuildModel(node.Content[0], &info)
	require.NoError(t, err)

	err = info.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.Equal(t, "My Overlay", info.Title.Value)
	assert.Equal(t, "1.0.0", info.Version.Value)
	assert.NotNil(t, info.Extensions)
	assert.Equal(t, 1, info.Extensions.Len())

	ext := info.FindExtension("x-custom")
	require.NotNil(t, ext)
	assert.Equal(t, "value", ext.Value.Value)
}

func TestInfo_Build_Minimal(t *testing.T) {
	yml := `title: Minimal`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	require.NoError(t, err)

	var info Info
	err = low.BuildModel(node.Content[0], &info)
	require.NoError(t, err)

	err = info.Build(context.Background(), nil, node.Content[0], nil)
	require.NoError(t, err)

	assert.Equal(t, "Minimal", info.Title.Value)
	assert.True(t, info.Version.IsEmpty())
}

func TestInfo_Hash(t *testing.T) {
	yml1 := `title: Overlay
version: 1.0.0`

	yml2 := `title: Overlay
version: 1.0.0`

	yml3 := `title: Different
version: 2.0.0`

	var node1, node2, node3 yaml.Node
	_ = yaml.Unmarshal([]byte(yml1), &node1)
	_ = yaml.Unmarshal([]byte(yml2), &node2)
	_ = yaml.Unmarshal([]byte(yml3), &node3)

	var info1, info2, info3 Info
	_ = low.BuildModel(node1.Content[0], &info1)
	_ = info1.Build(context.Background(), nil, node1.Content[0], nil)

	_ = low.BuildModel(node2.Content[0], &info2)
	_ = info2.Build(context.Background(), nil, node2.Content[0], nil)

	_ = low.BuildModel(node3.Content[0], &info3)
	_ = info3.Build(context.Background(), nil, node3.Content[0], nil)

	assert.Equal(t, info1.Hash(), info2.Hash())
	assert.NotEqual(t, info1.Hash(), info3.Hash())
}

func TestInfo_Hash_WithExtensions(t *testing.T) {
	yml := `title: Overlay
x-ext: value`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	var info Info
	_ = low.BuildModel(node.Content[0], &info)
	_ = info.Build(context.Background(), nil, node.Content[0], nil)

	hash := info.Hash()
	assert.NotEqual(t, [32]byte{}, hash)
}

func TestInfo_GettersReturnCorrectValues(t *testing.T) {
	yml := `title: Test`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	keyNode := &yaml.Node{Value: "info"}
	var info Info
	_ = low.BuildModel(node.Content[0], &info)
	_ = info.Build(context.Background(), keyNode, node.Content[0], nil)

	assert.Equal(t, keyNode, info.GetKeyNode())
	assert.Equal(t, node.Content[0], info.GetRootNode())
	assert.Nil(t, info.GetIndex())
	assert.NotNil(t, info.GetContext())
	assert.NotNil(t, info.GetExtensions())
}

func TestInfo_FindExtension_NotFound(t *testing.T) {
	yml := `title: Test`

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	var info Info
	_ = low.BuildModel(node.Content[0], &info)
	_ = info.Build(context.Background(), nil, node.Content[0], nil)

	ext := info.FindExtension("x-nonexistent")
	assert.Nil(t, ext)
}
