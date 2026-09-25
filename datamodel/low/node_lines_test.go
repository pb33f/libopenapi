// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package low

import (
	"sync"
	"testing"

	"github.com/pb33f/testify/assert"
	"go.yaml.in/yaml/v4"
)

// writeLines applies the same writes to a NodeLines: a stored single node that a later add turns into a
// pair, a line built up by adds, a stored value that a later add leaves alone, and a stored slice that a
// later Store replaces.
func writeLines(n *NodeLines, a, b, c *yaml.Node) {
	n.Store(1, a)
	n.add(1, b)
	n.add(2, a)
	n.add(2, b)
	n.add(2, c)
	n.Store(3, "not a node")
	n.add(3, a)
	n.Store(4, []*yaml.Node{a})
	n.Store(4, []*yaml.Node{c})
}

// Writes give the same result whether they are recorded before the line index is built or applied to it.
func TestNodeLines_WritesBeforeAndAfterIndexing(t *testing.T) {
	a, b, c := &yaml.Node{Value: "a"}, &yaml.Node{Value: "b"}, &yaml.Node{Value: "c"}

	var recorded NodeLines
	writeLines(&recorded, a, b, c)

	var applied NodeLines
	_, ok := applied.Load(1) // builds the empty index, so every write below is applied directly
	assert.False(t, ok)
	writeLines(&applied, a, b, c)

	for _, n := range []*NodeLines{&recorded, &applied} {
		one, _ := n.Load(1)
		assert.Equal(t, []*yaml.Node{a, b}, one)
		two, _ := n.Load(2)
		assert.Equal(t, []*yaml.Node{a, b, c}, two)
		three, _ := n.Load(3)
		assert.Equal(t, "not a node", three)
		four, _ := n.Load(4)
		assert.Equal(t, []*yaml.Node{c}, four)
	}
}

// Range visits lines in order, stops when asked, and lets the callback write to the nodes it ranges over.
func TestNodeLines_Range(t *testing.T) {
	var n NodeLines
	for _, line := range []int{30, 10, 20} {
		n.Store(line, &yaml.Node{Line: line})
	}

	var visited []int
	n.Range(func(line int, _ any) bool {
		visited = append(visited, line)
		n.Store(line+1, &yaml.Node{}) // writing mid-range must not deadlock
		return line < 20
	})
	assert.Equal(t, []int{10, 20}, visited)

	_, written := n.Load(11)
	assert.True(t, written)
}

// NodeLines is safe for concurrent writes and reads.
func TestNodeLines_Concurrent(t *testing.T) {
	var n NodeLines
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				n.add(i, &yaml.Node{Line: i})
				if i%10 == g {
					n.Load(i)
				}
			}
		}(g)
	}
	wg.Wait()

	lines := 0
	n.Range(func(_ int, value any) bool {
		assert.Len(t, value, 8)
		lines++
		return true
	})
	assert.Equal(t, 100, lines)
}
