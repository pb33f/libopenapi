// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/pb33f/jsonpath/pkg/jsonpath"
	"github.com/pb33f/libopenapi/utils"
	"github.com/pb33f/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestSpecIndex_MapNodes(t *testing.T) {
	petstore, _ := os.ReadFile("../test_specs/petstorev3.json")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(petstore, &rootNode)

	index := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())

	<-index.nodeMapCompleted

	// look up a node and make sure they match exactly (same pointer)
	path, _ := jsonpath.NewPath("$.paths['/pet'].put")
	nodes := path.Query(&rootNode)

	keyNode, valueNode := utils.FindKeyNodeTop("operationId", nodes[0].Content)
	mappedKeyNode, _ := index.GetNode(keyNode.Line, keyNode.Column)
	mappedValueNode, _ := index.GetNode(valueNode.Line, valueNode.Column)

	assert.Equal(t, keyNode, mappedKeyNode)
	assert.Equal(t, valueNode, mappedValueNode)

	// make sure the pointers are the same
	p1 := reflect.ValueOf(keyNode).Pointer()
	p2 := reflect.ValueOf(mappedKeyNode).Pointer()
	assert.Equal(t, p1, p2)

	// check missing line
	var ok bool
	mappedKeyNode, ok = index.GetNode(999999, 999)
	assert.False(t, ok)
	assert.Nil(t, mappedKeyNode)

	// check missing column on an existing line
	mappedKeyNode, ok = index.GetNode(12, 999)
	assert.False(t, ok)
	assert.Nil(t, mappedKeyNode)

	// check negative line
	mappedKeyNode, ok = index.GetNode(-1, 1)
	assert.False(t, ok)
	assert.Nil(t, mappedKeyNode)
}

func TestSpecIndex_GetNode_MissDoesNotLeakReadLock(t *testing.T) {
	index := NewSpecIndexWithConfig(&yaml.Node{}, CreateOpenAPIIndexConfig())
	index.nodeLines = [][]nodeLineEntry{
		nil,
		{{column: 1, node: &yaml.Node{Value: "ok"}}},
		nil,
	}

	node, ok := index.GetNode(2, 1)
	assert.False(t, ok)
	assert.Nil(t, node)

	locked := make(chan struct{})
	go func() {
		index.nodeMapLock.Lock()
		index.nodeLines = append(index.nodeLines, nil)
		index.nodeMapLock.Unlock()
		close(locked)
	}()

	select {
	case <-locked:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("writer lock blocked after GetNode miss")
	}
}

func TestSpecIndex_MapNodes_LineZeroAndGrowth(t *testing.T) {
	// a zero-value node reports line 0; nodes can also report lines beyond any
	// preallocated hint. both must be stored and retrievable without panics.
	lines := make([][]nodeLineEntry, 1)
	zeroNode := &yaml.Node{}
	lines = addNodeLineEntry(lines, zeroNode)
	assert.Same(t, zeroNode, lookupNodeLines(lines, 0, 0))

	farNode := &yaml.Node{Line: 500, Column: 3}
	lines = addNodeLineEntry(lines, farNode)
	assert.GreaterOrEqual(t, len(lines), 501)
	assert.Same(t, farNode, lookupNodeLines(lines, 500, 3))

	// a negative line is ignored, not stored.
	negNode := &yaml.Node{Line: -1, Column: 1}
	lines = addNodeLineEntry(lines, negNode)
	assert.Nil(t, lookupNodeLines(lines, -1, 1))

	// growth that doubles instead of exact-fit: line just past the end.
	nearNode := &yaml.Node{Line: 501, Column: 9}
	lines = addNodeLineEntry(lines, nearNode)
	assert.Same(t, nearNode, lookupNodeLines(lines, 501, 9))
}

func TestSpecIndex_MapNodes_OverwriteSemantics(t *testing.T) {
	// a later write to the same line/column replaces the earlier one — parents are
	// written after children in mapNodesRecursive, so parents win collisions.
	first := &yaml.Node{Line: 4, Column: 2, Value: "child"}
	second := &yaml.Node{Line: 4, Column: 2, Value: "parent"}

	var lines [][]nodeLineEntry
	lines = addNodeLineEntry(lines, first)
	lines = addNodeLineEntry(lines, second)

	assert.Same(t, second, lookupNodeLines(lines, 4, 2))
	assert.Len(t, lines[4], 1)
}

func TestSpecIndex_GetNodeMap_LegacyMaterialization(t *testing.T) {
	petstore, _ := os.ReadFile("../test_specs/petstorev3.json")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(petstore, &rootNode)

	index := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())

	// the legacy map must never be materialized by internal build paths.
	assert.Nil(t, index.legacyNodeMap)

	legacy := index.GetNodeMap()
	assert.NotNil(t, legacy)

	// every entry in the legacy map must match the line index exactly.
	total := 0
	for line, cols := range legacy {
		for col, n := range cols {
			total++
			found, ok := index.GetNode(line, col)
			assert.True(t, ok)
			assert.Same(t, n, found)
		}
	}
	assert.Positive(t, total)

	// second call returns the cached map.
	assert.Equal(t, reflect.ValueOf(legacy).Pointer(), reflect.ValueOf(index.GetNodeMap()).Pointer())
}

func TestSpecIndex_GetNodeMap_AfterRelease(t *testing.T) {
	petstore, _ := os.ReadFile("../test_specs/petstorev3.json")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(petstore, &rootNode)

	index := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())
	index.Release()

	// after release: no legacy map, no lookups, no blocking.
	assert.Nil(t, index.GetNodeMap())
	node, ok := index.GetNode(1, 1)
	assert.False(t, ok)
	assert.Nil(t, node)
}

func BenchmarkSpecIndex_MapNodes(b *testing.B) {
	petstore, _ := os.ReadFile("../test_specs/petstorev3.json")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(petstore, &rootNode)
	path, _ := jsonpath.NewPath("$.paths['/pet'].put")

	for i := 0; i < b.N; i++ {

		index := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())

		<-index.nodeMapCompleted

		// look up a node and make sure they match exactly (same pointer)
		nodes := path.Query(&rootNode)

		keyNode, valueNode := utils.FindKeyNodeTop("operationId", nodes[0].Content)
		mappedKeyNode, _ := index.GetNode(keyNode.Line, keyNode.Column)
		mappedValueNode, _ := index.GetNode(valueNode.Line, valueNode.Column)

		assert.Equal(b, keyNode, mappedKeyNode)
		assert.Equal(b, valueNode, mappedValueNode)

		// make sure the pointers are the same
		p1 := reflect.ValueOf(keyNode).Pointer()
		p2 := reflect.ValueOf(mappedKeyNode).Pointer()
		assert.Equal(b, p1, p2)
	}
}
