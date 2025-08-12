// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package low

import (
	"sync"
	"testing"

	"github.com/pkg-base/yaml"
	"github.com/stretchr/testify/assert"
)

func Test_NodeMapExtractNodes(t *testing.T) {
	yml := `one: hello
two: there
three: nice one
four:
  shoes: yes
  socks: of course
`

	var root yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &root)
	var syncMap sync.Map
	nm := &NodeMap{Nodes: &syncMap}
	nm.ExtractNodes(root.Content[0], false)
	testTheThing(t, nm)
}

func testTheThing(t *testing.T, nm *NodeMap) {
	count := 0
	nm.Nodes.Range(func(key, value interface{}) bool {
		count++
		return true
	})

	assert.Equal(t, 4, count)

	nodes := nm.GetNodes()

	assert.Equal(t, 2, len(nodes[1]))
	assert.Equal(t, 2, len(nodes[2]))
	assert.Equal(t, 2, len(nodes[3]))
	assert.Equal(t, 1, len(nodes[4]))
	assert.Equal(t, "one", nodes[1][0].Value)
	assert.Equal(t, "hello", nodes[1][1].Value)
	assert.Equal(t, "two", nodes[2][0].Value)
	assert.Equal(t, "there", nodes[2][1].Value)
	assert.Equal(t, "three", nodes[3][0].Value)
	assert.Equal(t, "nice one", nodes[3][1].Value)
	assert.Equal(t, "four", nodes[4][0].Value)
}

func testTheThingUnmarshalled(t *testing.T, nm *sync.Map) {
	n := &NodeMap{Nodes: nm}
	nodes := n.GetNodes()

	assert.Equal(t, 2, len(nodes[1]))
	assert.Equal(t, 2, len(nodes[2]))
	assert.Equal(t, 2, len(nodes[3]))
	assert.Equal(t, 1, len(nodes[4]))
	assert.Equal(t, "one", nodes[1][0].Value)
	assert.Equal(t, "hello", nodes[1][1].Value)
	assert.Equal(t, "two", nodes[2][0].Value)
	assert.Equal(t, "there", nodes[2][1].Value)
	assert.Equal(t, "three", nodes[3][0].Value)
	assert.Equal(t, "nice one", nodes[3][1].Value)
	assert.Equal(t, "four", nodes[4][0].Value)
}

func TestExtractNodes(t *testing.T) {
	yml := `one: hello
two: there
three: nice one
four:
  shoes: yes
  socks: of course
`

	var root yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &root)

	nm := ExtractNodes(nil, root.Content[0])

	count := 0
	nm.Range(func(key, value interface{}) bool {
		count++
		return true
	})

	assert.Equal(t, 4, count)
	testTheThingUnmarshalled(t, nm)
}

func TestExtractNodesRecursive(t *testing.T) {
	yml := `one: hello
two: there
three: nice one
four:
  shoes: yes
  socks: of course
`

	var root yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &root)

	nm := ExtractNodesRecursive(nil, root.Content[0])

	count := 0
	nm.Range(func(key, value interface{}) bool {
		count++
		return true
	})

	assert.Equal(t, 6, count)
	testTheThingUnmarshalled(t, nm)
}

func TestExtractNodes_Nil(t *testing.T) {
	var syncMap sync.Map
	nm := &NodeMap{Nodes: &syncMap}
	nm.ExtractNodes(nil, false)

	count := 0
	nm.Nodes.Range(func(key, value interface{}) bool {
		count++
		return true
	})

	assert.Equal(t, 0, count)
}

func Test_NodeMapExtractNodes_SingleNode(t *testing.T) {
	yml := `one: hello
two: there
three: nice one
four:
  shoes: yes
  socks: of course
`

	var root yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &root)
	var syncMap sync.Map
	nm := &NodeMap{Nodes: &syncMap}

	syncMap.Store(1, root.Content[0])

	nm.ExtractNodes(root.Content[0], false)
}

func Test_NodeMapGetNodes_SingleNode(t *testing.T) {
	var syncMap sync.Map
	nm := &NodeMap{Nodes: &syncMap}

	syncMap.Store(1, &yaml.Node{})
	ex := nm.GetNodes()
	assert.Equal(t, 1, len(ex))
}

func Test_NodeMapContainsLine(t *testing.T) {
	yml := `one: hello
two: there
three: nice one
four:
  shoes: yes
  socks: of course
`

	var root yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &root)

	var syncMap sync.Map
	nm := &NodeMap{Nodes: &syncMap}

	nm.ExtractNodes(root.Content[0], true)
	assert.True(t, nm.ContainsLine(1))
	assert.True(t, nm.ContainsLine(2))
	assert.True(t, nm.ContainsLine(3))
	assert.True(t, nm.ContainsLine(4))
	assert.True(t, nm.ContainsLine(5))
	assert.True(t, nm.ContainsLine(6))
	assert.False(t, nm.ContainsLine(7))
}

func Test_NodeMapGetNodes_EmptyNode(t *testing.T) {
	var syncMap sync.Map
	nm := &NodeMap{Nodes: &syncMap}

	ex := nm.GetNodes()
	assert.Equal(t, 1, len(ex))
}

func TestExtractExtensionNodes(t *testing.T) {
	yml := `openapi: 3.1
chack: spack
x-fresh: nice
x-cakes: yes
x-socks: of course
x-rice:
  yes: no
  no: yes`

	var root yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &root)

	extensions := ExtractExtensions(root.Content[0])
	var syncMap sync.Map
	ExtractExtensionNodes(nil, extensions, &syncMap)

	count := 0
	syncMap.Range(func(key, value interface{}) bool {
		count++
		return true
	})

	assert.Equal(t, 6, count)
}

func TestExtractNodes_NoContent(t *testing.T) {
	yml := `one`

	var root yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &root)

	nm := ExtractNodes(nil, root.Content[0])

	count := 0
	nm.Range(func(key, value interface{}) bool {
		count++
		return true
	})

	assert.Equal(t, 1, count)
}
