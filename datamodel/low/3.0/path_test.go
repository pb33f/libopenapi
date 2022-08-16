// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestPaths_Build(t *testing.T) {

	yml := `"/some/path":
  get:
    description: get method
  post:
    description: post method
  put:
    description: put method    
  delete:
    description: delete method
  options:
    description: options method
  patch:
    description: patch method  
  head:
    description: head method  
  trace:
    description: trace method
  servers:
    url: https://pb33f.io
  parameters:
    - name: hello
  x-cake: yummy
x-milk: cold`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Paths
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.NoError(t, err)

	path := n.FindPath("/some/path").Value
	assert.NotNil(t, path)
	assert.Equal(t, "get method", path.Get.Value.Description.Value)
	assert.Equal(t, "yummy", path.FindExtension("x-cake").Value)
	assert.Equal(t, "post method", path.Post.Value.Description.Value)
	assert.Equal(t, "put method", path.Put.Value.Description.Value)
	assert.Equal(t, "patch method", path.Patch.Value.Description.Value)
	assert.Equal(t, "delete method", path.Delete.Value.Description.Value)
	assert.Equal(t, "head method", path.Head.Value.Description.Value)
	assert.Equal(t, "trace method", path.Trace.Value.Description.Value)
	assert.Len(t, path.Parameters.Value, 1)
	assert.Equal(t, "cold", n.FindExtension("x-milk").Value)
	assert.Equal(t, "hello", path.Parameters.Value[0].Value.Name.Value)
}

func TestPaths_Build_Fail(t *testing.T) {

	yml := `"/some/path":
  $ref: $bork`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Paths
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.Error(t, err)

}

func TestPaths_Build_FailRef(t *testing.T) {

	// this is kinda nuts, and, it's completely illegal, but you never know!
	yml := `"/some/path":
 description: this is some path
 get:
   description: bloody dog ate my biscuit.
 post:
   description: post method
"/another/path":
 $ref: '#/~1some~1path'`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Paths
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.NoError(t, err)

	somePath := n.FindPath("/some/path").Value
	anotherPath := n.FindPath("/another/path").Value
	badPath := n.FindPath("/does/not/exist")
	assert.NotNil(t, somePath)
	assert.NotNil(t, anotherPath)
	assert.Nil(t, badPath)
	assert.Equal(t, "this is some path", somePath.Description.Value)
	assert.Equal(t, "bloody dog ate my biscuit.", somePath.Get.Value.Description.Value)
	assert.Equal(t, "post method", somePath.Post.Value.Description.Value)
	assert.Equal(t, "bloody dog ate my biscuit.", anotherPath.Get.Value.Description.Value)
}

func TestPaths_Build_FailRefDeadEnd(t *testing.T) {

	// this is nuts.
	yml := `"/no/path":
  get:
    $ref: '#/nowhere'
"/some/path":
  get:
    $ref: '#/~1some~1path/get'    
"/another/path":
  $ref: '#/~1some~1path'`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Paths
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestPaths_Build_SuccessRef(t *testing.T) {

	// this is kinda nuts, it's also not illegal, however the mechanics still need to work.
	yml := `"/some/path":
 description: this is some path
 get:
   $ref: '#/~1another~1path/get'
 post:
   description: post method
"/another/path":
 description: this is another path of some kind.
 get:
   description: get method from /another/path`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Paths
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.NoError(t, err)

	somePath := n.FindPath("/some/path").Value
	anotherPath := n.FindPath("/another/path").Value
	badPath := n.FindPath("/does/not/exist")
	assert.NotNil(t, somePath)
	assert.NotNil(t, anotherPath)
	assert.Nil(t, badPath)
	assert.Equal(t, "this is some path", somePath.Description.Value)
	assert.Equal(t, "get method from /another/path", somePath.Get.Value.Description.Value)
	assert.Equal(t, "post method", somePath.Post.Value.Description.Value)
	assert.Equal(t, "get method from /another/path", anotherPath.Get.Value.Description.Value)
}

func TestPaths_Build_BadParams(t *testing.T) {

	yml := `"/some/path":
  parameters:
    this: shouldFail`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Paths
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestPaths_Build_BadRef(t *testing.T) {

	// this is kinda nuts, it's also not illegal, however the mechanics still need to work.
	yml := `"/some/path":
 description: this is some path
 get:
   $ref: '#/no-where'
 post:
   description: post method
"/another/path":
 description: this is another path of some kind.
 get:
   description: get method from /another/path`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Paths
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestPathItem_Build_GoodRef(t *testing.T) {

	// this is kinda nuts, it's also not illegal, however the mechanics still need to work.
	yml := `"/some/path":
 description: this is some path
 get:
   $ref: '#/~1another~1path/get'
 post:
   description: post method
"/another/path":
 description: this is another path of some kind.
 get:
   $ref: '#/~1cakes/get'
"/cakes":
 description: cakes are awesome 
 get:
   description: get method from /cakes`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Paths
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.NoError(t, err)
}

func TestPathItem_Build_BadRef(t *testing.T) {

	// this is kinda nuts, it's also not illegal, however the mechanics still need to work.
	yml := `"/some/path":
 description: this is some path
 get:
   $ref: '#/~1another~1path/get'
 post:
   description: post method
"/another/path":
 description: this is another path of some kind.
 get:
   $ref: '#/~1cakes/NotFound'
"/cakes":
 description: cakes are awesome 
 get:
   description: get method from /cakes`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Paths
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestPathNoOps(t *testing.T) {

	// this is kinda nuts, it's also not illegal, however the mechanics still need to work.
	yml := `"/some/path":
"/cakes":`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Paths
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.NoError(t, err)
}

func TestPathItem_Build_Using_Ref(t *testing.T) {

	// first we need an index.
	doc := `paths:
 '/something/here':
   post:
    description: there is something here!`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(doc), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	yml := `"/some/path":
 description: this is some path
 get:
   $ref: '#/paths/~1something~1here/post'`

	var rootNode yaml.Node
	mErr = yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	var n Paths
	err := low.BuildModel(rootNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(rootNode.Content[0], idx)
	assert.NoError(t, err)

	somePath := n.FindPath("/a/path")
	assert.Nil(t, somePath)

	somePath = n.FindPath("/some/path")
	assert.NotNil(t, somePath.Value)
	assert.Equal(t, "this is some path", somePath.Value.Description.Value)
	assert.Equal(t, "there is something here!", somePath.Value.Get.Value.Description.Value)
}
