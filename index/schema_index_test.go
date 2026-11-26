// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
)

func TestNewSchemaIndexMapsAuthoredSourceWithoutExtractingReferences(t *testing.T) {
	var doc yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("examples:\n  - $ref: literal.json\nproperties:\n  price:\n    $ref: money.json\n    $id: urn:price\n"), &doc))
	config := CreateClosedAPIIndexConfig()
	config.SpecAbsolutePath = "schemas/order.yaml"
	config.Rolodex = NewRolodex(config)
	idx := NewSchemaIndex(&doc, config)
	require.Same(t, &doc, idx.GetRootNode())
	require.Same(t, config.Rolodex, idx.GetRolodex())
	require.Equal(t, "schemas/order.yaml", idx.GetSpecAbsolutePath())
	require.Empty(t, idx.GetAllReferences())
	require.Empty(t, idx.GetAllSchemaIds())
	example := doc.Content[0].Content[1].Content[0].Content[1]
	price := doc.Content[0].Content[3].Content[1]
	ref := price.Content[1]
	for _, value := range []*yaml.Node{example, ref} {
		found, ok := idx.GetNode(value.Line, value.Column)
		require.True(t, ok)
		require.Same(t, value, found)
	}
	entry := &SchemaIdEntry{Id: price.Content[3].Value, SchemaNode: price, Index: idx}
	require.NoError(t, idx.RegisterSchemaId(entry))
	require.Same(t, price, idx.GetSchemaById("urn:price").SchemaNode)
	idx.Release()
	require.Nil(t, idx.GetSchemaById("urn:price"))
	require.Nil(t, idx.GetRootNode())
	require.Nil(t, idx.GetNodeMap())
	_, ok := idx.GetNode(ref.Line, ref.Column)
	require.False(t, ok)
	require.Equal(t, "money.json", ref.Value)
	idx.Release()
}

func TestNewSchemaIndexRootForms(t *testing.T) {
	for _, source := range []string{"false", "{}"} {
		var doc yaml.Node
		require.NoError(t, yaml.Unmarshal([]byte("\n"+source), &doc))
		for _, input := range []*yaml.Node{&doc, doc.Content[0]} {
			idx := NewSchemaIndex(input, CreateClosedAPIIndexConfig())
			require.Same(t, input, idx.GetRootNode())
			got, ok := idx.GetNode(2, 1)
			require.True(t, ok)
			require.Same(t, doc.Content[0], got)
			idx.Release()
		}
	}
	for _, input := range []*yaml.Node{nil, {Kind: yaml.DocumentNode}} {
		idx := NewSchemaIndex(input, CreateClosedAPIIndexConfig())
		require.Same(t, input, idx.GetRootNode())
		got, ok := idx.GetNode(1, 1)
		require.Nil(t, got)
		require.False(t, ok)
		idx.Release()
	}
}
