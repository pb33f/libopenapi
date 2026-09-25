// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package high

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/utils"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

// sameEncodedTree reports the first difference between two node trees, comparing every field.
func sameEncodedTree(want, got *yaml.Node, path string) error {
	switch {
	case want.Kind != got.Kind || want.Style != got.Style || want.Tag != got.Tag || want.Value != got.Value:
		return fmt.Errorf("%s: %v/%v/%q/%q != %v/%v/%q/%q", path, got.Kind, got.Style, got.Tag, got.Value,
			want.Kind, want.Style, want.Tag, want.Value)
	case want.Anchor != got.Anchor || want.Alias != got.Alias || want.Line != got.Line || want.Column != got.Column:
		return fmt.Errorf("%s: anchor, alias or position differ", path)
	case want.HeadComment != got.HeadComment || want.LineComment != got.LineComment ||
		want.FootComment != got.FootComment:
		return fmt.Errorf("%s: comments differ", path)
	case (want.Content == nil) != (got.Content == nil) || len(want.Content) != len(got.Content):
		return fmt.Errorf("%s: content differs", path)
	}
	for i := range want.Content {
		if err := sameEncodedTree(want.Content[i], got.Content[i], fmt.Sprintf("%s/%d", path, i)); err != nil {
			return err
		}
	}
	return nil
}

// requireEncodesLikeYAML asserts encodeValue matches Encode on a first (uncached) and second (cached) call.
func requireEncodesLikeYAML(t *testing.T, value any) {
	t.Helper()
	var want yaml.Node
	wantErr := want.Encode(encodeSafeValue(value))
	for pass := 0; pass < 2; pass++ {
		var got yaml.Node
		err := encodeValue(&got, value)
		require.Equal(t, wantErr, err)
		require.NoError(t, sameEncodedTree(&want, &got, "value"), "pass %d", pass)
	}
}

func walkNodes(n *yaml.Node, fn func(*yaml.Node)) {
	fn(n)
	for _, c := range n.Content {
		walkNodes(c, fn)
	}
}

// Every value encodes exactly as Encode encodes it, whether or not an earlier encoding is reused.
func TestEncodeValue_MatchesEncode(t *testing.T) {
	ClearEncodeCache()
	fixtures := []string{"burgershop.openapi.yaml", "petstorev3.json", "all-the-components.yaml",
		"vendor-extensions-components.yaml", "circular-tests.yaml", "nullable-examples.openapi.yaml"}
	for _, fixture := range fixtures {
		data, err := os.ReadFile(filepath.Join("..", "..", "test_specs", fixture))
		require.NoError(t, err)
		var doc yaml.Node
		require.NoError(t, yaml.Unmarshal(data, &doc))
		walkNodes(doc.Content[0], func(n *yaml.Node) {
			if n.Kind == yaml.ScalarNode && n.Value == "" {
				return // Encode panics on an empty scalar, cached or not
			}
			requireEncodesLikeYAML(t, n)
			if n.Kind == yaml.SequenceNode {
				requireEncodesLikeYAML(t, n.Content)
				strs := make([]string, len(n.Content))
				for i, c := range n.Content {
					strs[i] = c.Value
				}
				requireEncodesLikeYAML(t, strs)
			}
		})
	}
	for _, value := range []any{
		[]string{}, []string{"", "true", "123", "a: b", "- x", "multi\nline", " padded ", "#hash", "yes"},
		[]*yaml.Node{}, []*yaml.Node{utils.CreateStringNode("one"), utils.CreateIntNode("2")},
		&yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Content: []*yaml.Node{}},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "commented", HeadComment: "# head",
			LineComment: "# line", FootComment: "# foot"},
	} {
		requireEncodesLikeYAML(t, value)
	}
}

// Values whose encoding cannot be reused are encoded directly, with the same result.
func TestEncodeValue_Uncacheable(t *testing.T) {
	ClearEncodeCache()
	anchored := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "anchored", Anchor: "a"}
	big := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	for i := 0; i <= encodeCacheMaxNodes; i++ {
		big.Content = append(big.Content, utils.CreateIntNode(fmt.Sprint(i)))
	}
	manyStrings := make([]string, encodeCacheMaxNodes+1)
	for i := range manyStrings {
		manyStrings[i] = fmt.Sprint(i)
	}
	for _, value := range []any{
		anchored,
		&yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Content: []*yaml.Node{anchored}},
		[]*yaml.Node{utils.CreateStringNode("one"), anchored},
		big,
		manyStrings,
		[]string{strings.Repeat("long ", encodeCacheMaxKeyBytes)},
		&yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{utils.CreateStringNode("doc")}},
		[]int{1, 2},
		map[string]string{"a": "b"},
	} {
		key, cacheable := appendEncodeFingerprint(nil, value)
		assert.False(t, cacheable && len(key) <= encodeCacheMaxKeyBytes, "%T should not be cached", value)
		requireEncodesLikeYAML(t, value)
	}
	encodeCache.RLock()
	assert.Empty(t, encodeCache.entries)
	encodeCache.RUnlock()

	// an alias node and a nil slice element are never fingerprinted.
	_, cacheable := appendEncodeFingerprint(nil, &yaml.Node{Kind: yaml.AliasNode, Alias: anchored})
	assert.False(t, cacheable)
	_, cacheable = appendEncodeFingerprint(nil, []*yaml.Node{nil})
	assert.False(t, cacheable)
}

// An encoding error is returned as Encode returns it, and nothing is cached.
func TestEncodeValue_Error(t *testing.T) {
	ClearEncodeCache()
	invalid := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!", Value: "no tag suffix"}
	var want, got yaml.Node
	wantErr := want.Encode(encodeSafeValue(invalid))
	require.Error(t, wantErr)
	require.Equal(t, wantErr, encodeValue(&got, invalid))
	encodeCache.RLock()
	assert.Empty(t, encodeCache.entries)
	encodeCache.RUnlock()
}

// The cache is emptied when it fills, so it stays bounded.
func TestEncodeValue_Bounded(t *testing.T) {
	ClearEncodeCache()
	for i := 0; i <= encodeCacheLimit; i++ {
		var n yaml.Node
		require.NoError(t, encodeValue(&n, []string{fmt.Sprint(i)}))
	}
	encodeCache.RLock()
	assert.Len(t, encodeCache.entries, 1)
	encodeCache.RUnlock()

	ClearEncodeCache()
	encodeCache.RLock()
	assert.Empty(t, encodeCache.entries)
	encodeCache.RUnlock()
}

// Rendering encodes copies of model nodes, so the model is never mutated by a render.
func TestNodeBuilder_RenderLeavesNodesUnchanged(t *testing.T) {
	type test struct {
		Enum    []*yaml.Node `yaml:"enum,omitempty"`
		Example *yaml.Node   `yaml:"example,omitempty"`
	}
	enum := []*yaml.Node{utils.CreateStringNode("a"), utils.CreateStringNode("b")}
	example := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map",
		Content: []*yaml.Node{utils.CreateStringNode("k"), utils.CreateStringNode("v")}}
	for pass := 0; pass < 2; pass++ {
		_, err := yaml.Marshal(NewNodeBuilder(&test{Enum: enum, Example: example}, nil).Render())
		require.NoError(t, err)
		for _, n := range append(enum, example.Content...) {
			assert.Equal(t, "!!str", n.Tag)
			assert.Equal(t, yaml.Style(0), n.Style)
		}
		assert.Equal(t, "!!map", example.Tag)
	}
}
