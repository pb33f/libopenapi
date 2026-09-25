// Copyright 2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package bundler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

const issue607Root = `openapi: 3.0.2
info:
  title: t
  version: 1.0.0
tags:
  $ref: "tags.yaml#/tags"
paths:
  /x:
    get:
      responses:
        "200":
          description: ok
`

func issue607TagsFile(count int) string {
	var b strings.Builder
	b.WriteString("tags:\n")
	for i := 0; i < count; i++ {
		fmt.Fprintf(&b, "- name: Tag%d\n  description: d%d\n", i, i)
	}
	return b.String()
}

func issue607Value(t *testing.T, node *yaml.Node, path ...string) *yaml.Node {
	t.Helper()
	for _, key := range path {
		require.Equal(t, yaml.MappingNode, node.Kind, "parent of %q", key)
		var next *yaml.Node
		for i := 0; i+1 < len(node.Content); i += 2 {
			if node.Content[i].Value == key {
				next = node.Content[i+1]
				break
			}
		}
		require.NotNil(t, next, "missing key %q", key)
		node = next
	}
	return node
}

func issue607Node(t *testing.T, src string) *yaml.Node {
	t.Helper()
	var doc yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(src), &doc))
	return doc.Content[0]
}

// https://github.com/pb33f/libopenapi/issues/607
func TestBundleBytesComposed_Issue607_SequenceRefDoesNotPanic(t *testing.T) {
	for _, count := range []int{1, 2, 3, 63, 64, 65} {
		t.Run(fmt.Sprintf("%d tags", count), func(t *testing.T) {
			tmpDir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "tags.yaml"), []byte(issue607TagsFile(count)), 0o644))

			config := datamodel.NewDocumentConfiguration()
			config.BasePath = tmpDir

			var bundled []byte
			var err error
			require.NotPanics(t, func() {
				bundled, err = BundleBytesComposed([]byte(issue607Root), config, nil)
			})
			require.NoError(t, err)
			assert.Contains(t, string(bundled), "/x:")
		})
	}
}

func TestBundleDocumentComposed_Issue607_InlinedSequenceRefBecomesSequence(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "tags.yaml"), []byte(issue607TagsFile(3)), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "optags.yaml"), []byte("list:\n  tags: [a, b, c]\n"), 0o644))

	root := strings.Replace(issue607Root, "    get:\n", "    get:\n      tags:\n        $ref: \"optags.yaml#/list/tags\"\n", 1)

	config := datamodel.NewDocumentConfiguration()
	config.BasePath = tmpDir
	doc, err := libopenapi.NewDocumentWithConfiguration([]byte(root), config)
	require.NoError(t, err)
	v3Doc, err := doc.BuildV3Model()
	require.NoError(t, err)

	_, err = BundleDocumentComposed(&v3Doc.Model, nil)
	require.NoError(t, err)

	rootNode := v3Doc.Index.GetRootNode().Content[0]

	tags := issue607Value(t, rootNode, "tags")
	assert.Equal(t, yaml.SequenceNode, tags.Kind)
	assert.Equal(t, "!!seq", tags.Tag)
	require.Len(t, tags.Content, 3)
	for i, tag := range tags.Content {
		assert.Equal(t, fmt.Sprintf("Tag%d", i), issue607Value(t, tag, "name").Value)
	}

	opTags := issue607Value(t, rootNode, "paths", "/x", "get", "tags")
	out, err := yaml.Marshal(opTags)
	require.NoError(t, err)
	assert.Equal(t, "[a, b, c]\n", string(out))
}

func TestInlineRefNode_Issue607(t *testing.T) {
	newRef := func() *yaml.Node {
		ref := issue607Node(t, "{$ref: 'other.yaml#/thing'}")
		ref.HeadComment = "# keep me"
		return ref
	}

	t.Run("sequence target", func(t *testing.T) {
		ref, target := newRef(), issue607Node(t, "[a, b, c]")
		inlineRefNode(ref, target)

		assert.Equal(t, yaml.SequenceNode, ref.Kind)
		assert.Equal(t, "!!seq", ref.Tag)
		assert.Equal(t, yaml.FlowStyle, ref.Style)
		assert.Equal(t, target.Content, ref.Content)
	})

	t.Run("scalar target", func(t *testing.T) {
		ref, target := newRef(), issue607Node(t, `"hello"`)
		inlineRefNode(ref, target)

		assert.Equal(t, yaml.ScalarNode, ref.Kind)
		assert.Equal(t, "!!str", ref.Tag)
		assert.Equal(t, yaml.DoubleQuotedStyle, ref.Style)
		assert.Equal(t, "hello", ref.Value)
		assert.Empty(t, ref.Content)
	})

	t.Run("alias target", func(t *testing.T) {
		ref, target := newRef(), issue607Node(t, "base: &b [x]\nalias: *b\n").Content[3]
		require.Equal(t, yaml.AliasNode, target.Kind)
		inlineRefNode(ref, target)

		assert.Equal(t, yaml.AliasNode, ref.Kind)
		assert.Same(t, target.Alias, ref.Alias)
		assert.Equal(t, "b", ref.Value)
	})

	t.Run("mapping target only swaps content", func(t *testing.T) {
		ref, target := newRef(), issue607Node(t, "name: pet\ntype: object\n")
		inlineRefNode(ref, target)

		assert.Equal(t, yaml.MappingNode, ref.Kind)
		assert.Equal(t, "!!map", ref.Tag)
		assert.Equal(t, yaml.FlowStyle, ref.Style)
		assert.Equal(t, "# keep me", ref.HeadComment)
		assert.Equal(t, target.Content, ref.Content)
	})
}
