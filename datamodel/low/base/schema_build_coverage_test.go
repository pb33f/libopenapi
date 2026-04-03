// Copyright 2022-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

type collectingAddNodes struct {
	lines []int
}

func (c *collectingAddNodes) AddNode(key int, _ *yaml.Node) {
	c.lines = append(c.lines, key)
}

func TestSchemaBuild_InvalidNestedSchemaFields(t *testing.T) {
	cases := []struct {
		name  string
		field string
		value string
	}{
		{name: "anyOf", field: "anyOf", value: "oops"},
		{name: "oneOf", field: "oneOf", value: "oops"},
		{name: "prefixItems", field: "prefixItems", value: "oops"},
		{name: "not", field: "not", value: "oops"},
		{name: "contains", field: "contains", value: "oops"},
		{name: "items", field: "items", value: "oops"},
		{name: "if", field: "if", value: "oops"},
		{name: "else", field: "else", value: "oops"},
		{name: "then", field: "then", value: "oops"},
		{name: "propertyNames", field: "propertyNames", value: "oops"},
		{name: "unevaluatedItems", field: "unevaluatedItems", value: "oops"},
		{name: "unevaluatedProperties", field: "unevaluatedProperties", value: "oops"},
		{name: "additionalProperties", field: "additionalProperties", value: "oops"},
		{name: "contentSchema", field: "contentSchema", value: "oops"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spec := fmt.Sprintf("%s: %s\n", tc.field, tc.value)
			var root yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(spec), &root))

			var schema Schema
			require.NoError(t, low.BuildModel(root.Content[0], &schema))
			err := schema.Build(context.Background(), root.Content[0], nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to build schema")
		})
	}
}

func TestResolveSchemaBuildInput_NilAndRefFailures(t *testing.T) {
	resolved, err := resolveSchemaBuildInput(context.Background(), nil, nil, "boom: %s")
	require.NoError(t, err)
	assert.Nil(t, resolved.valueNode)
	assert.Nil(t, resolved.idx)

	var missingRef yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("$ref: './missing.yaml#/Pet'"), &missingRef))

	cfg := index.CreateClosedAPIIndexConfig()
	idx := index.NewSpecIndexWithConfig(&missingRef, cfg)
	_, err = resolveSchemaBuildInput(context.Background(), missingRef.Content[0], idx, "boom: %s")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom: ./missing.yaml#/Pet")
}

func TestRecursiveSchemaNodeHelpers(t *testing.T) {
	low.MergeRecursiveNodesIfLineAbsent(nil, nil)
	low.AppendRecursiveNodes(nil, nil)

	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("example:\n  nested:\n    value: ok\n"), &root))
	node := root.Content[0]

	var dst sync.Map
	blockedLine := node.Content[0].Line
	dst.Store(blockedLine, []*yaml.Node{{Value: "existing"}})

	low.MergeRecursiveNodesIfLineAbsent(&dst, node)

	_, blocked := dst.Load(blockedLine)
	assert.True(t, blocked)

	var foundNested bool
	dst.Range(func(key, value any) bool {
		if key.(int) == node.Content[1].Content[0].Line {
			foundNested = true
		}
		assert.NotNil(t, value)
		return true
	})
	assert.True(t, foundNested)

	collector := &collectingAddNodes{}
	low.AppendRecursiveNodes(collector, node)
	assert.NotEmpty(t, collector.lines)
}
