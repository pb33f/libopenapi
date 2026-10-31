package index

import (
	"context"
	"strings"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
)

func TestExtractRefsStopsAtAliasLimit(t *testing.T) {
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("type: string\n"), &root))
	idx := NewSpecIndex(&root)
	// The parser starts checking the alias ratio after 1,000 node visits.
	walk := &extractRefsTraversal{
		ancestry: make(map[*yaml.Node]struct{}),
		nodes:    1000, aliasNodes: 1000, aliasDepth: 1,
	}
	require.Nil(t, idx.extractRefs(context.Background(), root.Content[0], &root, []string{"x-shared"}, 0, false, "", walk))
	require.True(t, walk.stopped)
	require.Len(t, idx.GetReferenceIndexErrors(), 1)
	err := idx.GetReferenceIndexErrors()[0].(*ResolvingError)
	require.ErrorContains(t, err, "document contains excessive aliasing")
	require.Same(t, root.Content[0], err.Node)
	require.Equal(t, "$.x-shared", err.Path)
	// Once stopped, later siblings do not continue work or add duplicate errors.
	require.Nil(t, idx.extractRefs(context.Background(), root.Content[0], &root, nil, 0, false, "", walk))
	require.Len(t, idx.GetReferenceIndexErrors(), 1)
	require.Equal(t, 1001, walk.nodes)
}

func TestExtractRefsAbsentAliasTarget(t *testing.T) {
	idx := NewSpecIndex(nil)
	for _, node := range []*yaml.Node{nil, {Kind: yaml.AliasNode}} {
		require.Nil(t, idx.ExtractRefs(context.Background(), node, nil, nil, 0, false, ""))
	}
	require.Empty(t, idx.GetReferenceIndexErrors())
}

func TestExtractRefsAllowsOrdinaryAliases(t *testing.T) {
	const source = `openapi: 3.1.1
info: {title: Aliases, version: '1'}
paths: {}
x-meta: &meta {description: shared metadata}
x-ref: &ref {$ref: '#/components/schemas/Target'}
components:
  schemas:
    Target: {type: string}
    First: {type: object, properties: {value: *ref}, x-meta: *meta}
    Second: {type: object, properties: {value: *ref}, x-meta: *meta}
`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(source), &root))
	idx := NewSpecIndex(&root)
	require.Empty(t, idx.GetReferenceIndexErrors())
	paths := make(map[string]bool)
	for _, ref := range idx.GetRawReferencesSequenced() {
		paths[strings.Join(ref.SourcePath, "/")] = true
	}
	require.Contains(t, paths, "components/schemas/First/properties/value")
	require.Contains(t, paths, "components/schemas/Second/properties/value")
}
