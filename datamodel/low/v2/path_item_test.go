// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"context"
	"errors"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestPathItem_Build_Params(t *testing.T) {
	yml := `parameters:
  $ref: break`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n PathItem
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestPathItem_Build_SkippedExternalReferenceAndSiblingOperation(t *testing.T) {
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(`swagger: '2.0'
info: {title: Pets, version: '1'}
paths:
  /pets:
    $ref: './shared.yaml#/paths/~1pets'
    get:
      operationId: listPets
      responses:
        '200': {description: Pet list}
`), &root))
	cfg := index.CreateOpenAPIIndexConfig()
	cfg.SkipExternalRefResolution = true
	idx := index.NewSpecIndexWithConfig(&root, cfg)
	_, paths := utils.FindKeyNodeTop("paths", root.Content[0].Content)
	_, source := utils.FindKeyNodeTop("/pets", paths.Content)
	var item PathItem
	require.NoError(t, item.Build(context.Background(), nil, source, idx))
	assert.Equal(t, "./shared.yaml#/paths/~1pets", item.Ref.Value)
	assert.Same(t, source.Content[0], item.Ref.KeyNode)
	assert.Same(t, source.Content[1], item.Ref.ValueNode)
	assert.Empty(t, item.Parameters.Value)
	require.NotNil(t, item.Get.Value)
	assert.Equal(t, "listPets", item.Get.Value.OperationId.Value)
}

func TestPathItem_Build_MethodFail(t *testing.T) {
	yml := `post:
  $ref: break`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n PathItem
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestPathItem_Build_MethodModelBuildFail(t *testing.T) {
	origBuildFn := buildPathItemOperationModel
	buildPathItemOperationModel = func(_ *yaml.Node, _ interface{}) error {
		return errors.New("model boom")
	}
	defer func() {
		buildPathItemOperationModel = origBuildFn
	}()

	yml := `get:
  description: hi`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n PathItem
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "model boom")
}

func TestPathItem_Hash(t *testing.T) {
	yml := `get:
  description: get me up
put:
  description: put me out
post:
  description: post me there
delete:
  description: delete me please
options:
  description: whats on the menu
parameters:
  - name: fishy
    location: sea
head:
  description: meta me up
patch:
  description: I got a boo boo
x-winter: is coming`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n PathItem
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	yml2 := `post:
  description: post me there
put:
  description: put me out
delete:
  description: delete me please
options:
  description: whats on the menu
patch:
  description: I got a boo boo
head:
  description: meta me up
x-winter: is coming
get:
  description: get me up
parameters:
  - name: fishy
    location: sea`

	var idxNode2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml2), &idxNode2)
	idx2 := index.NewSpecIndex(&idxNode2)

	var n2 PathItem
	_ = low.BuildModel(idxNode2.Content[0], &n2)
	_ = n2.Build(context.Background(), nil, idxNode2.Content[0], idx2)

	// hash
	assert.Equal(t, n.Hash(), n2.Hash())
	assert.Equal(t, 1, orderedmap.Len(n.GetExtensions()))
}
