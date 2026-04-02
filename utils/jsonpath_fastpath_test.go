// Copyright 2023-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package utils

import (
	"testing"
	"time"

	"github.com/pb33f/jsonpath/pkg/jsonpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestFindNodesWithoutDeserializingWithOptions_FastPathBypassesJSONPathEngine(t *testing.T) {
	root, _ := FindNodes(getPetstore(), "$")

	original := jsonPathQuery
	called := false
	jsonPathQuery = func(path *jsonpath.JSONPath, node *yaml.Node) []*yaml.Node {
		called = true
		return original(path, node)
	}
	defer func() {
		jsonPathQuery = original
	}()

	nodes, err := FindNodesWithoutDeserializingWithOptions(root[0], "$.info.contact", JSONPathLookupOptions{})
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.False(t, called)
}

func TestFindNodesWithoutDeserializingWithOptions_FastPathSupportsBracketPropertiesAndIndexes(t *testing.T) {
	spec := `paths:
  /pet:
    get:
      parameters:
        - name: id
          in: path`

	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(spec), &root))

	nodes, err := FindNodesWithoutDeserializingWithOptions(&root, "$.paths['/pet'].get.parameters[0].name", JSONPathLookupOptions{})
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, "id", nodes[0].Value)
}

func TestFindNodesWithoutDeserializingFastPath_Edges(t *testing.T) {
	results, handled := findNodesWithoutDeserializingFastPath(nil, "$.info.contact")
	assert.True(t, handled)
	assert.Nil(t, results)

	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("items:\n  - zero\n"), &root))
	results, handled = findNodesWithoutDeserializingFastPath(&root, "$.items[3]")
	assert.True(t, handled)
	assert.Nil(t, results)
}

func TestParseSimpleJSONPath_Edges(t *testing.T) {
	validCases := []struct {
		path      string
		stepCount int
		lastKind  simpleJSONPathStepKind
	}{
		{path: "$.items.0", stepCount: 2, lastKind: simpleJSONPathIndex},
		{path: "$['paths'][0]", stepCount: 2, lastKind: simpleJSONPathIndex},
	}
	for _, tc := range validCases {
		t.Run(tc.path, func(t *testing.T) {
			steps, ok := parseSimpleJSONPath(tc.path)
			require.True(t, ok)
			require.Len(t, steps, tc.stepCount)
			assert.Equal(t, tc.lastKind, steps[len(steps)-1].kind)
		})
	}

	for _, path := range []string{"", "info.contact", "$.", "$x", "$.info.*", "$[", "$['unterminated", "$[abc]", "$[]"} {
		t.Run(path, func(t *testing.T) {
			steps, ok := parseSimpleJSONPath(path)
			assert.False(t, ok)
			assert.Nil(t, steps)
		})
	}
}

func TestJSONPathFastPathNavigationHelpers(t *testing.T) {
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("info:\n  contact:\n    name: jane\nitems:\n  - zero\n"), &root))

	doc := &root
	mapNode := navigateJSONPathProperty(doc.Content[0], "info")
	require.NotNil(t, mapNode)
	assert.Nil(t, navigateJSONPathProperty(doc.Content[0], "missing"))

	itemsNode := navigateJSONPathProperty(doc.Content[0], "items")
	require.NotNil(t, itemsNode)
	assert.Nil(t, navigateJSONPathProperty(itemsNode, "info"))
	indexNode := navigateJSONPathIndex(itemsNode, 0)
	require.NotNil(t, indexNode)
	assert.Equal(t, "zero", indexNode.Value)
	assert.Nil(t, navigateJSONPathIndex(itemsNode, 9))
	assert.Nil(t, navigateJSONPathIndex(mapNode, 0))
}

func TestFindNodesWithoutDeserializingWithOptions_FallbackSuccess(t *testing.T) {
	root, _ := FindNodes(getPetstore(), "$")

	original := jsonPathQuery
	called := false
	jsonPathQuery = func(path *jsonpath.JSONPath, node *yaml.Node) []*yaml.Node {
		called = true
		return original(path, node)
	}
	defer func() {
		jsonPathQuery = original
	}()

	nodes, err := FindNodesWithoutDeserializingWithOptions(root[0], "$..contact", JSONPathLookupOptions{
		Timeout: 100 * time.Millisecond,
	})
	require.NoError(t, err)
	require.NotEmpty(t, nodes)
	assert.True(t, called)
}
