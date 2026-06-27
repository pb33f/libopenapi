// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package utils

import (
	"testing"

	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

func BenchmarkCloneYAMLNode_Scalar(b *testing.B) {
	node := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "value"}
	for b.Loop() {
		_ = CloneYAMLNode(node)
	}
}

func BenchmarkCloneYAMLNode_Schema(b *testing.B) {
	node := benchmarkCloneYAMLNodeSchema(b)
	for b.Loop() {
		_ = CloneYAMLNode(node)
	}
}

func BenchmarkCloneYAMLNode_Alias(b *testing.B) {
	target := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "target", Anchor: "target"}
	node := &yaml.Node{
		Kind: yaml.SequenceNode,
		Tag:  "!!seq",
		Content: []*yaml.Node{
			{Kind: yaml.AliasNode, Alias: target},
			{Kind: yaml.AliasNode, Alias: target},
		},
	}
	for b.Loop() {
		_ = CloneYAMLNode(node)
	}
}

func benchmarkCloneYAMLNodeSchema(b *testing.B) *yaml.Node {
	b.Helper()
	var root yaml.Node
	err := yaml.Unmarshal([]byte(`
type: object
required:
  - id
properties:
  id:
    type: string
  profile:
    allOf:
      - $ref: '#/$defs/Profile'
      - type: object
        properties:
          active:
            type: boolean
$defs:
  Profile:
    type: object
    properties:
      name:
        type: string
      tags:
        type: array
        items:
          type: string
`), &root)
	require.NoError(b, err)
	require.Len(b, root.Content, 1)
	return root.Content[0]
}
