// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
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

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.NotNil(t, n.GetRootNode())
	assert.Nil(t, n.GetKeyNode())
	assert.NotNil(t, n.GetIndex())
	assert.NotNil(t, n.GetContext())

	path := n.FindPath("/some/path").Value
	assert.NotNil(t, path)
	assert.Equal(t, "get method", path.Get.Value.Description.Value)

	var xCake string
	_ = path.FindExtension("x-cake").Value.Decode(&xCake)
	assert.Equal(t, "yummy", xCake)
	assert.Equal(t, "post method", path.Post.Value.Description.Value)
	assert.Equal(t, "put method", path.Put.Value.Description.Value)
	assert.Equal(t, "patch method", path.Patch.Value.Description.Value)
	assert.Equal(t, "delete method", path.Delete.Value.Description.Value)
	assert.Equal(t, "head method", path.Head.Value.Description.Value)
	assert.Equal(t, "trace method", path.Trace.Value.Description.Value)
	assert.Len(t, path.Parameters.Value, 1)
	assert.NotNil(t, path.GetContext())
	assert.NotNil(t, path.GetIndex())

	var xMilk string
	_ = n.FindExtension("x-milk").Value.Decode(&xMilk)
	assert.Equal(t, "cold", xMilk)
	assert.Equal(t, "hello", path.Parameters.Value[0].Value.Name.Value)
	assert.Equal(t, 1, orderedmap.Len(n.GetExtensions()))
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

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
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

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
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
    $ref: '#/no/path'
"/another/path":
  $ref: '#/~1some~1path'`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	var b []byte
	buf := bytes.NewBuffer(b)
	log := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
	cfg := index.SpecIndexConfig{
		Logger: log,
	}
	idx := index.NewSpecIndexWithConfig(&idxNode, &cfg)

	var n Paths
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	n.Build(context.Background(), nil, idxNode.Content[0], idx)

	assert.Contains(t, buf.String(), "msg=\"unable to locate reference anywhere in the rolodex\" reference=#/no/path")
	assert.Contains(t, buf.String(), "msg=\"unable to locate reference anywhere in the rolodex\" reference=#/nowhere")
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

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
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

	var b []byte
	buf := bytes.NewBuffer(b)
	log := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
	cfg := index.SpecIndexConfig{
		Logger: log,
	}
	idx := index.NewSpecIndexWithConfig(&idxNode, &cfg)

	var n Paths
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	er := buf.String()
	assert.Contains(t, er, "array build failed, input is not an array, line 3, column 5")
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

	var b []byte
	buf := bytes.NewBuffer(b)
	log := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
	cfg := index.SpecIndexConfig{
		Logger: log,
	}
	idx := index.NewSpecIndexWithConfig(&idxNode, &cfg)

	var n Paths
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.Contains(t, buf.String(), "unable to locate reference anywhere in the rolodex\" reference=#/no-where")
	assert.Contains(t, buf.String(), "error building path item: path item build failed: cannot find reference: #/no-where at line 4, col 10")
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

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
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

	var b []byte
	buf := bytes.NewBuffer(b)
	log := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
	cfg := index.SpecIndexConfig{
		Logger: log,
	}
	idx := index.NewSpecIndexWithConfig(&idxNode, &cfg)

	var n Paths
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.Contains(t, buf.String(), "unable to locate reference anywhere in the rolodex\" reference=#/~1cakes/NotFound")
	assert.Contains(t, buf.String(), "error building path item: path item build failed: cannot find reference: #/~1another~1path/get at line 4, col 10")
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

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)
}

func TestPathItem_Build_Using_Ref(t *testing.T) {
	// first we need an index.
	yml := `paths:
 '/something/here':
   post:
    description: there is something here!`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	yml = `"/some/path":
 description: this is some path
 get:
   $ref: '#/paths/~1something~1here/post'`

	var rootNode yaml.Node
	mErr = yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	var n Paths
	err := low.BuildModel(rootNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, rootNode.Content[0], idx)
	assert.NoError(t, err)

	somePath := n.FindPath("/a/path")
	assert.Nil(t, somePath)

	somePath = n.FindPath("/some/path")
	assert.NotNil(t, somePath.Value)
	assert.Equal(t, "this is some path", somePath.Value.Description.Value)
	assert.Equal(t, "there is something here!", somePath.Value.Get.Value.Description.Value)
}

func TestPath_Build_Using_CircularRef(t *testing.T) {
	// first we need an index.
	yml := `paths:
  '/something/here':
    post:
      $ref: '#/paths/~1something~1there/post'
  '/something/there':
    post:
      $ref: '#/paths/~1something~1here/post'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	yml = `"/some/path":
  $ref: '#/paths/~1something~1here/post'`

	var rootNode yaml.Node
	mErr = yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	var n Paths
	err := low.BuildModel(rootNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, rootNode.Content[0], idx)
	assert.Error(t, err)
}

func TestPath_Build_Using_CircularRefWithOp(t *testing.T) {
	// first we need an index.
	yml := `paths:
  '/something/here':
    post:
      $ref: '#/paths/~1something~1there/post'
  '/something/there':
    post:
      $ref: '#/paths/~1something~1here/post'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)

	var b []byte
	buf := bytes.NewBuffer(b)
	log := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
	cfg := index.SpecIndexConfig{
		Logger: log,
	}
	idx := index.NewSpecIndexWithConfig(&idxNode, &cfg)

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	yml = `"/some/path":
  post:
    $ref: '#/paths/~1something~1here/post'`

	var rootNode yaml.Node
	mErr = yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	var n Paths
	err := low.BuildModel(rootNode.Content[0], &n)
	assert.NoError(t, err)

	_ = n.Build(context.Background(), nil, rootNode.Content[0], idx)
	assert.Contains(t, buf.String(), "error building path item: build schema failed: circular reference 'post -> post -> post' found during lookup at line 4, column 7, It cannot be resolved")
}

func TestPaths_Build_BrokenOp(t *testing.T) {
	yml := `"/some/path":
  post:
    externalDocs:
      $ref: #bork`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	var b []byte
	buf := bytes.NewBuffer(b)
	log := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
	cfg := index.SpecIndexConfig{
		Logger: log,
	}
	idx := index.NewSpecIndexWithConfig(&idxNode, &cfg)

	var n Paths
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.Contains(t, buf.String(), "error building path item: object extraction failed: reference at line 4, column 7 is empty, it cannot be resolved")
}

func TestPaths_Hash(t *testing.T) {
	yml := `/french/toast:
  description: toast
/french/hen:
  description: chicken
/french/food:
  description: the worst.
x-france: french`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Paths
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	yml2 := `/french/toast:
  description: toast
/french/hen:
  description: chicken
/french/food:
  description: the worst.
x-france: french`

	var idxNode2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml2), &idxNode2)
	idx2 := index.NewSpecIndex(&idxNode2)

	var n2 Paths
	_ = low.BuildModel(idxNode2.Content[0], &n2)
	_ = n2.Build(context.Background(), nil, idxNode2.Content[0], idx2)

	// hash
	assert.Equal(t, n.Hash(), n2.Hash())
	a, b := n.FindPathAndKey("/french/toast")
	assert.NotNil(t, a)
	assert.NotNil(t, b)

	a, b = n.FindPathAndKey("I do not exist")
	assert.Nil(t, a)
	assert.Nil(t, b)
}

// Test parse failure among many paths.
// This stresses `TranslatePipeline`'s error handling.
func TestPaths_Build_Fail_Many(t *testing.T) {
	var yml string
	for i := 0; i < 1000; i++ {
		format := `"/fresh/code%d":
  parameters:
    $ref: break
`
		yml += fmt.Sprintf(format, i)
	}

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	var b []byte
	buf := bytes.NewBuffer(b)
	log := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
	cfg := index.SpecIndexConfig{
		Logger: log,
	}
	idx := index.NewSpecIndexWithConfig(&idxNode, &cfg)

	var n Paths
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	errors := strings.Split(buf.String(), "\n")
	assert.Len(t, errors, 1001)
}
