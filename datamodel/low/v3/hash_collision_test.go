// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

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

// Each case lists objects that differ only in which field or map key holds a value. Hashing values without their
// field names or keys made each case hash the same, so what-changed skipped them as unchanged.
func TestHash_FieldsAndKeysDoNotCollide(t *testing.T) {
	t.Run("parameter", func(t *testing.T) {
		assertHashesDiffer[Parameter](t,
			"name: id\nin: query\ndescription: form",
			"name: id\nin: query\nstyle: form",
			"name: id\nin: query\ncontent:\n  application/json:\n    schema:\n      type: string",
			"name: id\nin: query\ncontent:\n  text/plain:\n    schema:\n      type: string",
			"name: id\nin: query\nexamples:\n  first:\n    value: 1",
			"name: id\nin: query\nexamples:\n  second:\n    value: 1",
		)
	})

	t.Run("header", func(t *testing.T) {
		assertHashesDiffer[Header](t,
			"description: simple",
			"style: simple",
		)
	})

	t.Run("media type", func(t *testing.T) {
		assertHashesDiffer[MediaType](t,
			"schema:\n  type: string",
			"itemSchema:\n  type: string",
			"encoding:\n  first:\n    contentType: text/plain",
			"encoding:\n  second:\n    contentType: text/plain",
			"itemEncoding:\n  first:\n    contentType: text/plain",
			"examples:\n  first:\n    value: 1",
			"examples:\n  second:\n    value: 1",
		)
	})

	t.Run("encoding", func(t *testing.T) {
		assertHashesDiffer[Encoding](t,
			"contentType: form",
			"style: form",
		)
	})

	t.Run("request body", func(t *testing.T) {
		assertHashesDiffer[RequestBody](t,
			"content:\n  application/json:\n    schema:\n      type: string",
			"content:\n  application/xml:\n    schema:\n      type: string",
		)
	})

	t.Run("callback", func(t *testing.T) {
		assertHashesDiffer[Callback](t,
			"'{$request.body#/callbackUrl}':\n  post:\n    description: sent",
			"'{$request.body#/webhookUrl}':\n  post:\n    description: sent",
		)
	})

	t.Run("operation", func(t *testing.T) {
		callback := func(name string) string {
			return "callbacks:\n  " + name + ":\n    '{$request.body#/url}':\n      post:\n        description: sent"
		}
		assertHashesDiffer[Operation](t,
			"summary: same",
			"description: same",
			"operationId: same",
			"tags:\n  - same",
			callback("onCreated"),
			callback("onDeleted"),
		)
	})

	t.Run("link", func(t *testing.T) {
		assertHashesDiffer[Link](t,
			"operationId: getUser",
			"operationRef: getUser",
			"description: getUser",
			"parameters:\n  id: $response.body#/id",
			"parameters:\n  userId: $response.body#/id",
		)
	})

	t.Run("oauth flows", func(t *testing.T) {
		flow := "\n  tokenUrl: https://example.com/token\n  scopes:\n    read: read things"
		assertHashesDiffer[OAuthFlows](t,
			"implicit:"+flow,
			"password:"+flow,
			"clientCredentials:"+flow,
			"authorizationCode:"+flow,
			DeviceLabel+":"+flow,
		)
	})

	t.Run("oauth flow", func(t *testing.T) {
		assertHashesDiffer[OAuthFlow](t,
			"authorizationUrl: https://example.com",
			"tokenUrl: https://example.com",
			"refreshUrl: https://example.com",
		)
	})

	t.Run("server", func(t *testing.T) {
		assertHashesDiffer[Server](t,
			"url: https://example.com\nvariables:\n  region:\n    default: eu",
			"url: https://example.com\nvariables:\n  zone:\n    default: eu",
			"url: https://example.com\ndescription: same",
			"url: https://example.com\nname: same",
		)
	})
}
