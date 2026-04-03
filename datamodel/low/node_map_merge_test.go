// Copyright 2022-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package low

import (
	"sync"
	"testing"

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

func TestNodeMapMergeHelpers(t *testing.T) {
	MergeRecursiveNodesIfLineAbsent(nil, nil)
	AppendRecursiveNodes(nil, nil)
	walkRecursiveNodes(nil, nil)

	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("example:\n  nested:\n    value: ok\n"), &root))
	node := root.Content[0]

	var dst sync.Map
	blockedLine := node.Content[0].Line
	dst.Store(blockedLine, []*yaml.Node{{Value: "existing"}})

	MergeRecursiveNodesIfLineAbsent(&dst, node)

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
	AppendRecursiveNodes(collector, node)
	assert.NotEmpty(t, collector.lines)

	var walked []int
	walkRecursiveNodes(node, func(current *yaml.Node) {
		walked = append(walked, current.Line)
	})
	assert.NotEmpty(t, walked)
}
