package utils

import (
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestResolvedMergeNodeRetainsMetadata(t *testing.T) {
	const source = `base: &base
  inherited: first # inherited comment
  local: overridden
second: &second
  inherited: second
  extra: retained
sources: &sources [*base, *second]
merged: &merged # mapping comment
  <<: *sources
  local: chosen # local comment
alias: *merged
`
	var document yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(source), &document))
	before, err := yaml.Marshal(&document)
	require.NoError(t, err)
	root := document.Content[0]
	_, original := FindKeyNodeTop("merged", root.Content)
	_, base := FindKeyNodeTop("base", root.Content)
	key, value := FindKeyNodeTop("inherited", base.Content)
	alias := root.Content[len(root.Content)-1]
	view := ResolveMergeNodes(alias)
	require.NotSame(t, original, view)
	assert.Equal(t, original.Line, view.Line)
	assert.Equal(t, original.Column, view.Column)
	assert.Equal(t, original.Anchor, view.Anchor)
	assert.Equal(t, original.HeadComment, view.HeadComment)
	assert.Equal(t, original.LineComment, view.LineComment)
	viewKey, viewValue := FindKeyNodeTop("inherited", view.Content)
	assert.Same(t, key, viewKey)
	assert.Same(t, value, viewValue)
	_, local := FindKeyNodeTop("local", view.Content)
	assert.Equal(t, "chosen", local.Value)
	_, extra := FindKeyNodeTop("extra", view.Content)
	assert.Equal(t, "retained", extra.Value)
	assert.False(t, hasMergeKeys(view.Content))
	assert.Same(t, view, ResolveMergeNodes(view))
	after, err := yaml.Marshal(&document)
	require.NoError(t, err)
	assert.Equal(t, string(before), string(after))
}

func TestResolvedMergeNodeMappingCycleTerminates(t *testing.T) {
	node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	key := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "local"}
	value := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "retained"}
	node.Content = []*yaml.Node{
		{Kind: yaml.ScalarNode, Tag: "!!merge", Value: "<<"},
		{Kind: yaml.AliasNode, Alias: node},
		key, value,
	}
	view := ResolveMergeNodes(node)
	assert.Equal(t, []*yaml.Node{key, value}, view.Content)
	assert.Len(t, node.Content, 4)
}

func TestResolvedMergeNodeSequenceCycleTerminates(t *testing.T) {
	const source = `base: &base {inherited: first}
sequence: &sequence [*sequence, *base]
merged: {<<: *sequence, local: retained}
`
	var document yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(source), &document))
	_, original := FindKeyNodeTop("merged", document.Content[0].Content)
	view := ResolveMergeNodes(original)
	_, inherited := FindKeyNodeTop("inherited", view.Content)
	require.NotNil(t, inherited)
	assert.Equal(t, "first", inherited.Value)
	_, local := FindKeyNodeTop("local", view.Content)
	require.NotNil(t, local)
	assert.Equal(t, "retained", local.Value)
	assert.True(t, hasMergeKeys(original.Content))
}

func TestResolvedMergeNodeSharedSourcesKeepPrecedence(t *testing.T) {
	const source = `base: &base {inherited: base, local: base}
first: &first {<<: *base, inherited: first}
second: &second {<<: *base, inherited: second, extra: retained}
shared: &shared [*first, *second]
repeated: &repeated [*shared, *shared]
merged: {<<: *repeated, local: chosen}
`
	var document yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(source), &document))
	before, err := yaml.Marshal(&document)
	require.NoError(t, err)
	_, original := FindKeyNodeTop("merged", document.Content[0].Content)
	visited := make(map[*yaml.Node]struct{})
	content := expandMergeContent(original, visited)
	for key, expected := range map[string]string{
		"inherited": "first", "local": "chosen", "extra": "retained",
	} {
		_, value := FindKeyNodeTop(key, content)
		require.NotNil(t, value)
		require.Equal(t, expected, value.Value)
	}
	require.Len(t, content, 6)
	// Reaching a completed source again in this expansion does no further work.
	require.Empty(t, expandMergeContent(original, visited))
	_, repeated := FindKeyNodeTop("repeated", document.Content[0].Content)
	require.Empty(t, appendExpandedMergeContent(nil, make(map[string]struct{}), repeated, visited))
	// A new expansion still gives the same effective mapping.
	require.Equal(t, content, ResolveMergeNodes(original).Content)
	after, err := yaml.Marshal(&document)
	require.NoError(t, err)
	require.Equal(t, before, after)
}
