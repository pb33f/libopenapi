// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"fmt"
	"os"
	"reflect"
	"strings"
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
	sortNodeLines(lines)

	assert.Same(t, second, lookupNodeLines(lines, 4, 2))
	assert.Len(t, lines[4], 1)
}

func TestSpecIndex_MapNodes_RecordsEachNodeOnce(t *testing.T) {
	petstore, err := os.ReadFile("../test_specs/petstorev3.json")
	assert.NoError(t, err)
	var root yaml.Node
	assert.NoError(t, yaml.Unmarshal(petstore, &root))

	// Check before sorting: deduplication must not hide duplicate traversal writes.
	lines := mapNodesRecursive(&root, nil)
	writes := make(map[*yaml.Node]int)
	for _, entries := range lines {
		for _, entry := range entries {
			writes[entry.node]++
		}
	}
	var check func(*yaml.Node)
	check = func(node *yaml.Node) {
		assert.Equal(t, 1, writes[node], "node at %d:%d", node.Line, node.Column)
		for _, child := range node.Content {
			check(child)
		}
	}
	check(root.Content[0])
}

func TestSpecIndex_MapNodes_PostOrderCollisions(t *testing.T) {
	child := &yaml.Node{Line: 1, Column: 1}
	parent := &yaml.Node{Kind: yaml.SequenceNode, Line: 1, Column: 1, Content: []*yaml.Node{child}}
	first := &yaml.Node{Line: 2, Column: 1}
	last := &yaml.Node{Line: 2, Column: 1}
	root := &yaml.Node{Kind: yaml.SequenceNode, Line: 0, Content: []*yaml.Node{parent, first, last}}
	index := &SpecIndex{nodeMapCompleted: make(chan struct{})}
	index.MapNodes(root)

	node, ok := index.GetNode(1, 1)
	assert.True(t, ok)
	assert.Same(t, parent, node, "parents win over children")
	node, ok = index.GetNode(2, 1)
	assert.True(t, ok)
	assert.Same(t, last, node, "later siblings win over earlier siblings")
}

func TestSpecIndex_SortNodeLines_OrdersAndDedupes(t *testing.T) {
	// entries arrive in document (DFS) order, not column order. after sorting, lookups
	// must work for every column and the last write for a column must win.
	nodes := []*yaml.Node{
		{Line: 1, Column: 30, Value: "c"},
		{Line: 1, Column: 10, Value: "a"},
		{Line: 1, Column: 20, Value: "b-child"},
		{Line: 1, Column: 20, Value: "b-parent"},
		{Line: 1, Column: 5, Value: "first"},
	}
	var lines [][]nodeLineEntry
	for _, n := range nodes {
		lines = addNodeLineEntry(lines, n)
	}
	sortNodeLines(lines)

	assert.Len(t, lines[1], 4)
	assert.Equal(t, "first", lines[1][0].node.Value, "first entry on a line is the leftmost node")
	assert.Same(t, nodes[4], lookupNodeLines(lines, 1, 5))
	assert.Same(t, nodes[1], lookupNodeLines(lines, 1, 10))
	assert.Same(t, nodes[3], lookupNodeLines(lines, 1, 20))
	assert.Same(t, nodes[0], lookupNodeLines(lines, 1, 30))
	assert.Nil(t, lookupNodeLines(lines, 1, 25))
}

// singleLineJSONSpec builds a minified OpenAPI document with roughly n schema nodes,
// all on line 1, mirroring specs published as compact JSON (e.g. very large public APIs).
func singleLineJSONSpec(n int) []byte {
	var b strings.Builder
	b.WriteString(`{"openapi":"3.1.0","info":{"title":"dense","version":"1"},"paths":{},"components":{"schemas":{`)
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, `"s%d":{"type":"object","properties":{"id":{"type":"integer"},"ref":{"$ref":"#/components/schemas/s%d"}}}`, i, (i+1)%n)
	}
	b.WriteString(`}}}`)
	return []byte(b.String())
}

func TestSpecIndex_MapNodes_SingleLineDocument(t *testing.T) {
	spec := singleLineJSONSpec(2000)
	var rootNode yaml.Node
	assert.NoError(t, yaml.Unmarshal(spec, &rootNode))

	index := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())
	<-index.nodeMapCompleted

	// every node in the document lives on line 1 and must be retrievable by position.
	total := 0
	var walk func(n *yaml.Node)
	walk = func(n *yaml.Node) {
		for _, c := range n.Content {
			walk(c)
		}
		if n.Kind == yaml.DocumentNode {
			return
		}
		total++
		assert.Equal(t, 1, n.Line)
		found, ok := index.GetNode(n.Line, n.Column)
		assert.True(t, ok)
		assert.NotNil(t, found)
	}
	walk(&rootNode)
	assert.Greater(t, total, 2000*8)

	// the line index and the legacy map must agree.
	legacy := index.GetNodeMap()
	assert.Len(t, legacy, 1)
	assert.Len(t, legacy[1], len(index.nodeLines[1]))
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

// BenchmarkSpecIndex_MapNodes_SingleLine guards against per-line work becoming linear
// again: with 20k schemas (~200k nodes on one line) a linear scan per insert or lookup
// takes minutes, the sorted index takes milliseconds.
func BenchmarkSpecIndex_MapNodes_SingleLine(b *testing.B) {
	spec := singleLineJSONSpec(20000)
	var rootNode yaml.Node
	_ = yaml.Unmarshal(spec, &rootNode)
	probe := rootNode.Content[0].Content[1] // "info" key node
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		index := NewSpecIndexWithConfig(&rootNode, CreateOpenAPIIndexConfig())
		<-index.nodeMapCompleted
		found, ok := index.GetNode(probe.Line, probe.Column)
		if !ok || found != probe {
			b.Fatal("probe node not found")
		}
	}
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
