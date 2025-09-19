// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

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
