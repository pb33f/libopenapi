// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestServer_Build(t *testing.T) {
	yml := `x-coffee: hot
url: https://pb33f.io
description: high quality software for developers.
variables:
  var1: 
    default: hello
    description: a var
    enum: [one, two]`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Server
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)
	assert.Nil(t, n.GetRootNode())

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.NotNil(t, n.GetRootNode())
	// maphash uses random seed per process, so just test non-empty
	assert.NotEmpty(t, low.GenerateHashString(&n))

	assert.Equal(t, "https://pb33f.io", n.URL.Value)
	assert.Equal(t, "high quality software for developers.", n.Description.Value)
	assert.Equal(t, "hello", n.FindVariable("var1").Value.Default.Value)
	assert.Equal(t, "a var", n.FindVariable("var1").Value.Description.Value)

	// test var hash - maphash uses random seed per process, so just test non-empty
	s := n.FindVariable("var1")
	assert.NotEmpty(t, low.GenerateHashString(s.Value))

	assert.Equal(t, 1, orderedmap.Len(n.GetExtensions()))
	assert.NotNil(t, n.GetContext())
	assert.NotNil(t, n.GetIndex())

	// check nodes on variables
	for v := range n.Variables.Value.ValuesFromOldest() {
		assert.NotNil(t, v.Value.GetKeyNode())
		assert.NotNil(t, v.Value.GetRootNode())
		assert.Equal(t, 0, v.Value.GetExtensions().Len())
	}
}

func TestServerWithVariableExtension_Build(t *testing.T) {
	yml := `url: https://pb33f.io
description: high quality software for developers.
variables:
  var1: 
    default: hello
    description: a var
    enum: [one, two]
    x-transforms:
        allowMissing: true`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Server
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)
	assert.Nil(t, n.GetRootNode())

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.NotNil(t, n.GetRootNode())
	// maphash uses random seed per process, so just test non-empty
	assert.NotEmpty(t, low.GenerateHashString(&n))

	assert.Equal(t, "https://pb33f.io", n.URL.Value)
	assert.Equal(t, "high quality software for developers.", n.Description.Value)

	variable := n.FindVariable("var1").Value
	assert.Equal(t, "hello", variable.Default.Value)
	assert.Equal(t, "a var", variable.Description.Value)

	var_extensions := variable.Extensions.First()
	assert.Equal(t, "x-transforms", var_extensions.Key().Value)
	variable_pair := variable.Extensions.GetPair(var_extensions.Key())
	assert.Equal(t, "allowMissing", variable_pair.Value.Value.Content[0].Value)
	assert.Equal(t, "true", variable_pair.Value.Value.Content[1].Value)

	// test var hash - maphash uses random seed per process, so just test non-empty
	s := n.FindVariable("var1")
	assert.NotEmpty(t, low.GenerateHashString(s.Value))

	assert.Equal(t, 0, orderedmap.Len(n.GetExtensions()))

	// check nodes on variables
	for v := range n.Variables.Value.ValuesFromOldest() {
		assert.NotNil(t, v.Value.GetKeyNode())
		assert.NotNil(t, v.Value.GetRootNode())
		assert.NotNil(t, v.Value.GetExtensions())
	}
}

func TestServer_Build_NoVars(t *testing.T) {
	yml := `url: https://pb33f.io
description: high quality software for developers.`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Server
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "https://pb33f.io", n.URL.Value)
	assert.Equal(t, "high quality software for developers.", n.Description.Value)
	assert.Equal(t, 0, orderedmap.Len(n.Variables.Value))
}

func TestServer_Name_OpenAPI32(t *testing.T) {
	yml := `name: Production Server
url: https://api.example.com
description: Main production API server`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Server
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "Production Server", n.Name.Value)
	assert.Equal(t, "https://api.example.com", n.URL.Value)
	assert.Equal(t, "Main production API server", n.Description.Value)
}

func TestServer_Name_WithVariables(t *testing.T) {
	yml := `name: Staging Server
url: https://{environment}.api.example.com
description: Staging environment server
variables:
  environment:
    default: staging
    enum: [staging, dev, test]
    description: The environment name`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Server
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "Staging Server", n.Name.Value)
	assert.Equal(t, "https://{environment}.api.example.com", n.URL.Value)
	assert.Equal(t, "Staging environment server", n.Description.Value)
	assert.Equal(t, "staging", n.FindVariable("environment").Value.Default.Value)
}

func TestServer_Hash_WithName(t *testing.T) {
	left := `name: API Server
url: https://api.example.com
description: Main API`

	right := `url: https://api.example.com
description: Main API
name: API Server`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	idx := index.NewSpecIndex(&lNode)

	var lDoc, rDoc Server
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], idx)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], idx)

	// Same content, different order should produce same hash
	assert.Equal(t, lDoc.Hash(), rDoc.Hash())
}

func TestServer_Hash_DifferentName(t *testing.T) {
	left := `name: Production Server
url: https://api.example.com`

	right := `name: Development Server
url: https://api.example.com`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	idx := index.NewSpecIndex(&lNode)

	var lDoc, rDoc Server
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], idx)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], idx)

	// Different names should produce different hash
	assert.NotEqual(t, lDoc.Hash(), rDoc.Hash())
}
