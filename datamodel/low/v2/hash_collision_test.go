// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"context"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

type hashTestModel interface {
	Build(ctx context.Context, keyNode, root *yaml.Node, idx *index.SpecIndex) error
	Hash() uint64
}

// assertHashesDiffer builds every snippet into its own T and checks no two of them share a hash.
func assertHashesDiffer[T any, PT interface {
	*T
	hashTestModel
}](t *testing.T, snippets ...string) {
	t.Helper()

	hashes := make([]uint64, len(snippets))
	for i, snippet := range snippets {
		var root yaml.Node
		require.NoError(t, yaml.Unmarshal([]byte(snippet), &root))
		model := PT(new(T))
		require.NoError(t, low.BuildModel(root.Content[0], model))
		require.NoError(t, model.Build(context.Background(), nil, root.Content[0], index.NewSpecIndex(&root)))
		hashes[i] = model.Hash()
	}

	for i := range hashes {
		for j := i + 1; j < len(hashes); j++ {
			assert.NotEqual(t, hashes[i], hashes[j], "hash collides for:\n%s\n---\n%s", snippets[i], snippets[j])
		}
	}
}

// Items, headers and parameters carry the same validation keywords as a schema, so they need the same guarantee: a
// value hashes differently depending on the keyword that holds it, and enum values keep their type.
func TestHash_ValidationKeywordsDoNotCollide(t *testing.T) {
	t.Run("items", func(t *testing.T) {
		assertHashesDiffer[Items](t,
			"type: csv",
			"format: csv",
			"collectionFormat: csv",
			"pattern: csv",
			"enum:\n  - csv",
			"enum:\n  - 1",
			"enum:\n  - '1'",
		)
	})

	t.Run("header", func(t *testing.T) {
		assertHashesDiffer[Header](t,
			"description: csv",
			"type: csv",
			"format: csv",
			"collectionFormat: csv",
			"pattern: csv",
			"enum:\n  - csv",
			"enum:\n  - 1",
			"enum:\n  - '1'",
		)
	})

	t.Run("parameter", func(t *testing.T) {
		assertHashesDiffer[Parameter](t,
			"name: id\nin: query\ntype: csv",
			"name: id\nin: query\nformat: csv",
			"name: id\nin: query\ndescription: csv",
			"name: id\nin: query\ncollectionFormat: csv",
			"name: id\nin: query\npattern: csv",
			"name: id\nin: query\nenum:\n  - csv",
			"name: id\nin: query\nenum:\n  - 1",
			"name: id\nin: query\nenum:\n  - '1'",
		)
	})
}

// An operation that moves a value between its string fields or lists, such as a media type from consumes to
// produces, has changed, so it must not keep its hash.
func TestHash_OperationFieldsDoNotCollide(t *testing.T) {
	assertHashesDiffer[Operation](t,
		"summary: same",
		"description: same",
		"operationId: same",
		"tags:\n  - same",
		"consumes:\n  - same",
		"produces:\n  - same",
		"schemes:\n  - same",
	)
}
