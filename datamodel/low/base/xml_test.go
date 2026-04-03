// Copyright 2022-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestXML_Build(t *testing.T) {
	yml := `name: a thing
namespace: somewhere
wrapped: true`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n XML
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(&idxNode, idx)
	assert.NoError(t, err)
	assert.Equal(t, "a thing", n.Name.Value)
	assert.Equal(t, "somewhere", n.Namespace.Value)
	assert.True(t, n.Wrapped.Value)
	assert.NotNil(t, n.GetRootNode())
	assert.NotNil(t, n.GetIndex())
}

func TestXML_Build_WithNodeType(t *testing.T) {
	yml := `name: myElement
namespace: http://example.com/ns
nodeType: element
wrapped: false`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n XML
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(&idxNode, idx)
	assert.NoError(t, err)
	assert.Equal(t, "myElement", n.Name.Value)
	assert.Equal(t, "http://example.com/ns", n.Namespace.Value)
	assert.Equal(t, "element", n.NodeType.Value)
	assert.False(t, n.Wrapped.Value)

	// test that Hash includes nodeType
	hash1 := n.Hash()
	n.NodeType.Value = "attribute"
	hash2 := n.Hash()
	assert.NotEqual(t, hash1, hash2)
}

func TestXML_Build_WithAttributeAndNodeType(t *testing.T) {
	// test backward compatibility - both attribute and nodeType present
	yml := `name: myAttr
attribute: true
nodeType: attribute`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n XML
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(&idxNode, idx)
	assert.NoError(t, err)
	assert.Equal(t, "myAttr", n.Name.Value)
	assert.True(t, n.Attribute.Value)
	assert.Equal(t, "attribute", n.NodeType.Value)
}

func TestXML_Build_NilRoot(t *testing.T) {
	var n XML
	err := n.Build(nil, nil)
	assert.NoError(t, err)
	assert.Nil(t, n.GetRootNode())
	assert.Nil(t, n.GetExtensions())
}

func TestXML_Build_ScalarRoot(t *testing.T) {
	var scalar yaml.Node
	_ = yaml.Unmarshal([]byte("hello"), &scalar)

	var n XML
	err := low.BuildModel(scalar.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(scalar.Content[0], nil)
	assert.NoError(t, err)

	nodes := n.GetNodes()
	assert.Len(t, nodes[scalar.Content[0].Line], 1)
	assert.Equal(t, "hello", nodes[scalar.Content[0].Line][0].Value)
}
