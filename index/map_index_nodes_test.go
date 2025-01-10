// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"github.com/pb33f/libopenapi/utils"
	"github.com/speakeasy-api/jsonpath/pkg/jsonpath"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"os"
	"reflect"
	"testing"
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
	mappedKeyNode, ok = index.GetNode(999, 999)
	assert.False(t, ok)
	assert.Nil(t, mappedKeyNode)

	mappedKeyNode, ok = index.GetNode(12, 999)
	assert.False(t, ok)
	assert.Nil(t, mappedKeyNode)

	index.nodeMap[15] = nil
	mappedKeyNode, ok = index.GetNode(15, 999)
	assert.False(t, ok)
	assert.Nil(t, mappedKeyNode)
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
