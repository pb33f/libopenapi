// Copyright 2022-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"testing"

	lowmodel "github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

// buildReferencedExample builds an Example whose low model is a reference to another example
// in the same document, which is what the inline renderer resolves through.
func buildReferencedExample(t *testing.T) *Example {
	t.Helper()

	const spec = `components:
  examples:
    Target:
      summary: the referenced example
      value: resolved-value
    Referenced:
      $ref: '#/components/examples/Target'
`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(spec), &root))

	idx := index.NewSpecIndexWithConfig(&root, index.CreateOpenAPIIndexConfig())

	// Locate the Referenced example node.
	examples := root.Content[0].Content[1].Content[1]
	var refNode *yaml.Node
	for i := 0; i < len(examples.Content)-1; i += 2 {
		if examples.Content[i].Value == "Referenced" {
			refNode = examples.Content[i+1]
		}
	}
	require.NotNil(t, refNode)

	low := new(lowbase.Example)
	require.NoError(t, lowmodel.BuildModel(refNode, low))
	require.NoError(t, low.Build(context.Background(), nil, refNode, idx))

	return NewExample(low)
}

// Inline rendering resolves a referenced example and renders its target in place. The high
// model carries no Reference of its own here, so resolution goes through the low model rather
// than short-circuiting to a $ref node.
func TestExample_MarshalYAMLInline_ResolvesReference(t *testing.T) {
	example := buildReferencedExample(t)
	require.NotNil(t, example)
	require.Empty(t, example.Reference, "the high model holds no reference of its own here")

	t.Run("without context", func(t *testing.T) {
		rendered, err := example.MarshalYAMLInline()
		require.NoError(t, err)
		require.NotNil(t, rendered)

		out, err := yaml.Marshal(rendered)
		require.NoError(t, err)
		assert.NotContains(t, string(out), "$ref", "the target is rendered in place")
	})

	t.Run("with context", func(t *testing.T) {
		rendered, err := example.MarshalYAMLInlineWithContext(nil)
		require.NoError(t, err)
		require.NotNil(t, rendered)

		out, err := yaml.Marshal(rendered)
		require.NoError(t, err)
		assert.NotContains(t, string(out), "$ref")
	})
}
