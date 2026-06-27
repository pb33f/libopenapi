// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package utils

import (
	"testing"

	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestCloneYAMLNode_Nil(t *testing.T) {
	assert.Nil(t, CloneYAMLNode(nil))
	assert.Nil(t, CloneYAMLNodeWithFlags(nil, YAMLNodeCloneStripAnchors))
}

func TestCloneYAMLNode_ClonesScalarMetadata(t *testing.T) {
	original := &yaml.Node{
		Kind:        yaml.ScalarNode,
		Style:       yaml.SingleQuotedStyle,
		Tag:         "!!str",
		Value:       "value",
		Anchor:      "anchor",
		Line:        10,
		Column:      3,
		HeadComment: "head",
		LineComment: "line",
		FootComment: "foot",
	}

	cloned := CloneYAMLNode(original)

	require.NotSame(t, original, cloned)
	assert.Equal(t, original.Kind, cloned.Kind)
	assert.Equal(t, original.Style, cloned.Style)
	assert.Equal(t, original.Tag, cloned.Tag)
	assert.Equal(t, original.Value, cloned.Value)
	assert.Equal(t, original.Anchor, cloned.Anchor)
	assert.Equal(t, original.Line, cloned.Line)
	assert.Equal(t, original.Column, cloned.Column)
	assert.Equal(t, original.HeadComment, cloned.HeadComment)
	assert.Equal(t, original.LineComment, cloned.LineComment)
	assert.Equal(t, original.FootComment, cloned.FootComment)
}

func TestCloneYAMLNode_ClonesContentRecursively(t *testing.T) {
	original := &yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "key"},
			{
				Kind: yaml.SequenceNode,
				Tag:  "!!seq",
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Tag: "!!str", Value: "value"},
				},
			},
		},
	}

	cloned := CloneYAMLNode(original)

	require.NotSame(t, original, cloned)
	require.Len(t, cloned.Content, 2)
	require.NotSame(t, original.Content[0], cloned.Content[0])
	require.NotSame(t, original.Content[1], cloned.Content[1])
	require.Len(t, cloned.Content[1].Content, 1)
	require.NotSame(t, original.Content[1].Content[0], cloned.Content[1].Content[0])

	cloned.Content[1].Content[0].Value = "changed"
	assert.Equal(t, "value", original.Content[1].Content[0].Value)
}

func TestCloneYAMLNode_PreservesNilContentChildren(t *testing.T) {
	original := &yaml.Node{
		Kind:    yaml.SequenceNode,
		Tag:     "!!seq",
		Content: []*yaml.Node{nil},
	}

	cloned := CloneYAMLNode(original)

	require.NotSame(t, original, cloned)
	require.Len(t, cloned.Content, 1)
	assert.Nil(t, cloned.Content[0])
}

func TestCloneYAMLNode_ClonesAliasTargets(t *testing.T) {
	target := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "target", Anchor: "target"}
	original := &yaml.Node{
		Kind: yaml.SequenceNode,
		Tag:  "!!seq",
		Content: []*yaml.Node{
			{Kind: yaml.AliasNode, Alias: target},
			{Kind: yaml.AliasNode, Alias: target},
		},
	}

	cloned := CloneYAMLNode(original)

	firstAlias := cloned.Content[0]
	secondAlias := cloned.Content[1]
	require.NotSame(t, original.Content[0], firstAlias)
	require.NotSame(t, target, firstAlias.Alias)
	require.Same(t, firstAlias.Alias, secondAlias.Alias)

	firstAlias.Alias.Value = "changed"
	assert.Equal(t, "target", target.Value)
	assert.Equal(t, "changed", secondAlias.Alias.Value)
}

func TestCloneYAMLNode_ReusesSeenContentNodes(t *testing.T) {
	target := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "target"}
	original := &yaml.Node{
		Kind:    yaml.SequenceNode,
		Tag:     "!!seq",
		Content: []*yaml.Node{target, target},
	}

	cloned := CloneYAMLNode(original)

	require.NotSame(t, target, cloned.Content[0])
	require.Same(t, cloned.Content[0], cloned.Content[1])
}

func TestCloneYAMLNode_ClonesAliasCycles(t *testing.T) {
	root := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Anchor: "root"}
	root.Content = []*yaml.Node{
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "self"},
		{Kind: yaml.AliasNode, Alias: root},
	}

	cloned := CloneYAMLNode(root)

	require.NotSame(t, root, cloned)
	require.Len(t, cloned.Content, 2)
	require.NotSame(t, root.Content[1], cloned.Content[1])
	require.Same(t, cloned, cloned.Content[1].Alias)
}

func TestCloneYAMLNodeWithFlags_StripsAnchors(t *testing.T) {
	original := &yaml.Node{
		Kind:   yaml.MappingNode,
		Tag:    "!!map",
		Anchor: "root",
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "key", Anchor: "key"},
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "value", Anchor: "value"},
		},
	}

	cloned := CloneYAMLNodeWithFlags(original, YAMLNodeCloneStripAnchors)

	assert.Empty(t, cloned.Anchor)
	assert.Empty(t, cloned.Content[0].Anchor)
	assert.Empty(t, cloned.Content[1].Anchor)
	assert.Equal(t, "root", original.Anchor)
	assert.Equal(t, "key", original.Content[0].Anchor)
	assert.Equal(t, "value", original.Content[1].Anchor)
}

func TestCloneYAMLNodeWithFlags_UnwrapsDocument(t *testing.T) {
	child := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	document := &yaml.Node{
		Kind:    yaml.DocumentNode,
		Content: []*yaml.Node{child},
	}

	cloned := CloneYAMLNodeWithFlags(document, YAMLNodeCloneUnwrapDocument)

	require.NotSame(t, child, cloned)
	assert.Equal(t, yaml.MappingNode, cloned.Kind)
}

func TestCloneYAMLNodeWithFlags_DoesNotUnwrapEmptyDocument(t *testing.T) {
	document := &yaml.Node{Kind: yaml.DocumentNode}

	cloned := CloneYAMLNodeWithFlags(document, YAMLNodeCloneUnwrapDocument)

	require.NotSame(t, document, cloned)
	assert.Equal(t, yaml.DocumentNode, cloned.Kind)
}
