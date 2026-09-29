// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"context"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/low"
	v2 "github.com/pb33f/libopenapi/datamodel/low/v2"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

// The comparisons below stop early when both sides hash the same. Moving a value to another field or map key used to
// leave the hash untouched, so each change here, breaking ones included, was reported as no change at all.

func buildLowModel[T any, PT interface {
	*T
	Build(ctx context.Context, keyNode, root *yaml.Node, idx *index.SpecIndex) error
}](t *testing.T, yml string) PT {
	t.Helper()
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yml), &root))
	model := PT(new(T))
	require.NoError(t, low.BuildModel(root.Content[0], model))
	require.NoError(t, model.Build(context.Background(), nil, root.Content[0], index.NewSpecIndex(&root)))
	return model
}

func assertChangeTotals(t *testing.T, changes interface {
	TotalChanges() int
	TotalBreakingChanges() int
}, total, breaking int,
) {
	t.Helper()
	require.NotNil(t, changes)
	assert.Equal(t, total, changes.TotalChanges())
	assert.Equal(t, breaking, changes.TotalBreakingChanges())
}

func TestCompareSchemas_ValueMovedToAnotherKeyword(t *testing.T) {
	doc := func(input string) string {
		return "openapi: 3.1.0\ncomponents:\n  schemas:\n    Widget:\n      type: object\n    Input:\n" + input
	}
	tests := []struct {
		name            string
		left, right     string
		total, breaking int
	}{
		{
			name:  "minimum to maximum",
			left:  "      type: integer\n      minimum: 5",
			right: "      type: integer\n      maximum: 5",
			total: 2,
		},
		{
			name:     "oneOf to allOf",
			left:     "      oneOf:\n        - $ref: '#/components/schemas/Widget'\n        - type: object",
			right:    "      allOf:\n        - $ref: '#/components/schemas/Widget'\n        - type: object",
			total:    4,
			breaking: 2,
		},
		{
			name:     "not to items",
			left:     "      not:\n        $ref: '#/components/schemas/Widget'",
			right:    "      items:\n        $ref: '#/components/schemas/Widget'",
			total:    2,
			breaking: 2,
		},
		{
			name:  "readOnly to writeOnly",
			left:  "      readOnly: true",
			right: "      writeOnly: true",
			total: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cleanHashCacheForTest(t)
			leftDoc, rightDoc := test_BuildDoc(doc(tc.left), doc(tc.right))
			changes := CompareSchemas(leftDoc.Components.Value.FindSchema("Input").Value,
				rightDoc.Components.Value.FindSchema("Input").Value)
			assertChangeTotals(t, changes, tc.total, tc.breaking)
		})
	}
}

func TestCompare_ValueMovedToAnotherFieldOrKey(t *testing.T) {
	t.Run("request body content type", func(t *testing.T) {
		cleanHashCacheForTest(t)
		changes := CompareRequestBodies(
			buildLowModel[v3.RequestBody](t, "content:\n  application/json:\n    schema:\n      type: string"),
			buildLowModel[v3.RequestBody](t, "content:\n  application/xml:\n    schema:\n      type: string"))
		assertChangeTotals(t, changes, 2, 1)
	})

	t.Run("parameter content type", func(t *testing.T) {
		cleanHashCacheForTest(t)
		changes := CompareParametersV3(
			buildLowModel[v3.Parameter](t, "name: id\nin: query\ncontent:\n  application/json:\n    schema:\n      type: string"),
			buildLowModel[v3.Parameter](t, "name: id\nin: query\ncontent:\n  text/plain:\n    schema:\n      type: string"))
		assertChangeTotals(t, changes, 2, 1)
	})

	t.Run("media type schema to itemSchema", func(t *testing.T) {
		cleanHashCacheForTest(t)
		changes := CompareMediaTypes(
			buildLowModel[v3.MediaType](t, "schema:\n  type: string"),
			buildLowModel[v3.MediaType](t, "itemSchema:\n  type: string"))
		assertChangeTotals(t, changes, 2, 2)
	})

	t.Run("oauth password flow to client credentials", func(t *testing.T) {
		cleanHashCacheForTest(t)
		flow := "\n  tokenUrl: https://example.com/token\n  scopes:\n    read: read things"
		changes := CompareOAuthFlows(
			buildLowModel[v3.OAuthFlows](t, "password:"+flow),
			buildLowModel[v3.OAuthFlows](t, "clientCredentials:"+flow))
		assertChangeTotals(t, changes, 2, 1)
	})

	t.Run("callback expression", func(t *testing.T) {
		cleanHashCacheForTest(t)
		callbacks := func(expression string) string {
			return "callbacks:\n  onEvent:\n    '" + expression + "':\n      post:\n        description: sent"
		}
		changes := CompareOperations(
			buildLowModel[v3.Operation](t, callbacks("{$request.body#/callbackUrl}")),
			buildLowModel[v3.Operation](t, callbacks("{$request.body#/webhookUrl}")))
		assertChangeTotals(t, changes, 2, 1)
	})

	t.Run("swagger consumes to produces", func(t *testing.T) {
		cleanHashCacheForTest(t)
		changes := CompareOperations(
			buildLowModel[v2.Operation](t, "consumes:\n  - application/json"),
			buildLowModel[v2.Operation](t, "produces:\n  - application/json"))
		assertChangeTotals(t, changes, 2, 1)
	})
}
