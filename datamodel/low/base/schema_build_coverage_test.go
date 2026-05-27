// Copyright 2022-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"unsafe"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

type collectingAddNodes struct {
	lines []int
}

//go:linkname lowBuildModelFieldCache github.com/pb33f/libopenapi/datamodel/low.buildModelFieldCache
var lowBuildModelFieldCache sync.Map

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

func TestTransformSiblingRefNode(t *testing.T) {
	var siblingRef yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("$ref: '#/components/schemas/Name'\ndeprecated: true"), &siblingRef))

	transformed, metadata, ok := transformSiblingRefNode(siblingRef.Content[0], nil)
	require.False(t, ok)
	assert.Nil(t, metadata)
	assert.Equal(t, siblingRef.Content[0], transformed)

	cfg := index.CreateOpenAPIIndexConfig()
	cfg.TransformSiblingRefs = true
	idx := index.NewSpecIndexWithConfig(&siblingRef, cfg)

	var refOnly yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("$ref: '#/components/schemas/Name'"), &refOnly))
	transformed, metadata, ok = transformSiblingRefNode(refOnly.Content[0], idx)
	require.False(t, ok)
	assert.Nil(t, metadata)
	assert.Equal(t, refOnly.Content[0], transformed)

	transformed, metadata, ok = transformSiblingRefNode(siblingRef.Content[0], idx)
	require.True(t, ok)
	require.NotNil(t, transformed)
	require.NotNil(t, metadata)
	require.Len(t, transformed.Content, 2)
	assert.Equal(t, "allOf", transformed.Content[0].Value)
	assert.Equal(t, "#/components/schemas/Name", metadata.reference)
}

func TestResolveSchemaBuildInput_TransformsSiblingRefBeforeResolution(t *testing.T) {
	spec := `openapi: 3.1.0
info:
  title: sibling refs
  version: 1.0.0
paths: {}
components:
  schemas:
    Name:
      type: string
    Container:
      type: object
      properties:
        foo:
          $ref: '#/components/schemas/Name'
          deprecated: true`

	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(spec), &root))

	cfg := index.CreateOpenAPIIndexConfig()
	cfg.TransformSiblingRefs = true
	idx := index.NewSpecIndexWithConfig(&root, cfg)

	fooNode := findNestedSchemaTestNode(t, root.Content[0], "components", "schemas", "Container", "properties", "foo")
	resolved, err := resolveSchemaBuildInput(context.Background(), fooNode, idx, "boom: %s")
	require.NoError(t, err)
	require.NotNil(t, resolved.valueNode)
	assert.Equal(t, "allOf", resolved.valueNode.Content[0].Value)
	assert.Equal(t, fooNode, resolved.scopeNode)
	assert.Nil(t, resolved.refNode)
	require.NotNil(t, resolved.transformed)
	assert.Equal(t, fooNode, resolved.transformed.referenceNode)
	assert.Empty(t, resolved.refLocation)
	assert.Equal(t, idx, resolved.idx)

	built := buildSchemaProxy(resolved.ctx, resolved.idx, fooNode, resolved.valueNode, resolved.scopeNode, resolved.refNode, resolved.transformed, resolved.refLocation)
	assert.Equal(t, fooNode, built.Value.TransformedRef)
	assert.True(t, built.Value.IsTransformedRefWithSiblings())
	assert.Equal(t, "#/components/schemas/Name", built.Value.GetTransformedRefReference())
	assert.Equal(t, "allOf", built.Value.GetTransformedRefAllOfSchema().Content[0].Value)
	require.NotNil(t, built.Value.GetTransformedRefSiblingSchema())
	require.Len(t, built.Value.GetTransformedRefSiblingSchema().Content, 2)
	assert.Equal(t, "deprecated", built.Value.GetTransformedRefSiblingSchema().Content[0].Value)
}

func findNestedSchemaTestNode(t *testing.T, node *yaml.Node, path ...string) *yaml.Node {
	t.Helper()

	current := node
	for _, key := range path {
		require.NotNil(t, current)
		require.Equal(t, yaml.MappingNode, current.Kind)

		var next *yaml.Node
		for i := 0; i+1 < len(current.Content); i += 2 {
			if current.Content[i].Value == key {
				next = current.Content[i+1]
				break
			}
		}
		require.NotNil(t, next, "missing key %q", key)
		current = next
	}
	return current
}

func TestSchemaBuild_BuildModelError(t *testing.T) {
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("type: string\n"), &root))

	var seed Schema
	require.NoError(t, low.BuildModel(root.Content[0], &seed))

	schemaType := reflect.TypeOf(Schema{})
	original, ok := lowBuildModelFieldCache.Load(schemaType)
	require.True(t, ok)

	origType := reflect.TypeOf(original)
	elemType := origType.Elem()
	replacement := reflect.MakeSlice(origType, 1, 1)
	elem := reflect.New(elemType).Elem()
	setUnexportedField(elem.FieldByName("lookupKey"), "type")
	setUnexportedField(elem.FieldByName("index"), 0)
	setUnexportedField(elem.FieldByName("kind"), reflect.Bool)
	replacement.Index(0).Set(elem)

	lowBuildModelFieldCache.Store(schemaType, replacement.Interface())
	t.Cleanup(func() {
		lowBuildModelFieldCache.Store(schemaType, original)
	})

	var schema Schema
	err := schema.Build(context.Background(), root.Content[0], nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to parse unsupported type")
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

func setUnexportedField(field reflect.Value, value any) {
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(reflect.ValueOf(value))
}
