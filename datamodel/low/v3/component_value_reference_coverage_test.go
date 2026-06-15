// Copyright 2022-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

type refAwareLowModel interface {
	Build(context.Context, *yaml.Node, *yaml.Node, *index.SpecIndex) error
	GetKeyNode() *yaml.Node
	GetReference() string
	GetReferenceNode() *yaml.Node
	GetRootNode() *yaml.Node
	IsReference() bool
}

type scalarRootLowModel interface {
	Build(context.Context, *yaml.Node, *yaml.Node, *index.SpecIndex) error
	GetKeyNode() *yaml.Node
	GetNodes() map[int][]*yaml.Node
	GetRootNode() *yaml.Node
}

func TestComponentValueReferenceBuilders_PreserveRootRef(t *testing.T) {
	ref := "./components.yaml#/shared/Thing"
	keyNode, rootNode := componentValueRefNode(t, ref)

	tests := []struct {
		name  string
		build func(*testing.T, *yaml.Node, *yaml.Node) refAwareLowModel
	}{
		{
			name: "Callback",
			build: func(t *testing.T, keyNode, rootNode *yaml.Node) refAwareLowModel {
				return buildRootRefModel(t, keyNode, rootNode, &Callback{})
			},
		},
		{
			name: "Header",
			build: func(t *testing.T, keyNode, rootNode *yaml.Node) refAwareLowModel {
				return buildRootRefModel(t, keyNode, rootNode, &Header{})
			},
		},
		{
			name: "Link",
			build: func(t *testing.T, keyNode, rootNode *yaml.Node) refAwareLowModel {
				return buildRootRefModel(t, keyNode, rootNode, &Link{})
			},
		},
		{
			name: "Parameter",
			build: func(t *testing.T, keyNode, rootNode *yaml.Node) refAwareLowModel {
				return buildRootRefModel(t, keyNode, rootNode, &Parameter{})
			},
		},
		{
			name: "PathItem",
			build: func(t *testing.T, keyNode, rootNode *yaml.Node) refAwareLowModel {
				return buildRootRefModel(t, keyNode, rootNode, &PathItem{})
			},
		},
		{
			name: "RequestBody",
			build: func(t *testing.T, keyNode, rootNode *yaml.Node) refAwareLowModel {
				return buildRootRefModel(t, keyNode, rootNode, &RequestBody{})
			},
		},
		{
			name: "Response",
			build: func(t *testing.T, keyNode, rootNode *yaml.Node) refAwareLowModel {
				return buildRootRefModel(t, keyNode, rootNode, &Response{})
			},
		},
		{
			name: "SecurityScheme",
			build: func(t *testing.T, keyNode, rootNode *yaml.Node) refAwareLowModel {
				return buildRootRefModel(t, keyNode, rootNode, &SecurityScheme{})
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := test.build(t, keyNode, rootNode)

			assert.True(t, model.IsReference())
			assert.Equal(t, ref, model.GetReference())
			assert.Same(t, rootNode, model.GetReferenceNode())
			assert.Same(t, rootNode, model.GetRootNode())
			assert.Same(t, keyNode, model.GetKeyNode())
		})
	}
}

func TestScalarRootBuilders_RetainScalarNode(t *testing.T) {
	rootNode := scalarRootNode(t, "scalar-root")

	tests := []struct {
		name  string
		build func(*testing.T, *yaml.Node) scalarRootLowModel
	}{
		{
			name: "Link",
			build: func(t *testing.T, rootNode *yaml.Node) scalarRootLowModel {
				return buildScalarRootModel(t, rootNode, &Link{})
			},
		},
		{
			name: "MediaType",
			build: func(t *testing.T, rootNode *yaml.Node) scalarRootLowModel {
				return buildScalarRootModel(t, rootNode, &MediaType{})
			},
		},
		{
			name: "Parameter",
			build: func(t *testing.T, rootNode *yaml.Node) scalarRootLowModel {
				return buildScalarRootModel(t, rootNode, &Parameter{})
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := test.build(t, rootNode)
			nodes := model.GetNodes()

			assert.Same(t, rootNode, model.GetRootNode())
			assert.Nil(t, model.GetKeyNode())
			require.Len(t, nodes[rootNode.Line], 1)
			assert.Same(t, rootNode, nodes[rootNode.Line][0])
			assert.Equal(t, "scalar-root", nodes[rootNode.Line][0].Value)
		})
	}
}

func TestPathItem_Build_IgnoresUnknownScalarFields(t *testing.T) {
	yml := `summary: supported metadata
purge: disabled
get:
  description: supported operation`

	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yml), &root))
	idx := index.NewSpecIndex(&root)
	rootNode := root.Content[0]

	var pathItem PathItem
	require.NoError(t, low.BuildModel(rootNode, &pathItem))
	require.NoError(t, pathItem.Build(context.Background(), nil, rootNode, idx))

	require.NotNil(t, pathItem.Get.Value)
	assert.Equal(t, "supported operation", pathItem.Get.Value.Description.Value)
	assert.Nil(t, pathItem.AdditionalOperations.Value)
}

func componentValueRefNode(t *testing.T, ref string) (*yaml.Node, *yaml.Node) {
	t.Helper()

	var root yaml.Node
	yml := "thing:\n  $ref: '" + ref + "'\n"
	require.NoError(t, yaml.Unmarshal([]byte(yml), &root))
	require.NotEmpty(t, root.Content)
	require.Len(t, root.Content[0].Content, 2)
	return root.Content[0].Content[0], root.Content[0].Content[1]
}

func scalarRootNode(t *testing.T, value string) *yaml.Node {
	t.Helper()

	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(value), &root))
	require.NotEmpty(t, root.Content)
	return root.Content[0]
}

func buildRootRefModel[T refAwareLowModel](t *testing.T, keyNode, rootNode *yaml.Node, model T) T {
	t.Helper()

	idx := index.NewSpecIndexWithConfig(rootNode, &index.SpecIndexConfig{SkipExternalRefResolution: true})
	require.NoError(t, low.BuildModel(rootNode, model))
	require.NoError(t, model.Build(context.Background(), keyNode, rootNode, idx))
	return model
}

func buildScalarRootModel[T scalarRootLowModel](t *testing.T, rootNode *yaml.Node, model T) T {
	t.Helper()

	require.NoError(t, low.BuildModel(rootNode, model))
	require.NoError(t, model.Build(context.Background(), nil, rootNode, nil))
	return model
}
