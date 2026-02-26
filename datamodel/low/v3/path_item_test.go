// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestPathItem_Hash(t *testing.T) {
	yml := `description: a path item
summary: it's another path item
servers:
  - url: https://pb33f.io
parameters: 
  - in: head
get:
  description: get me
post:
  description: post me
put:
  description: put me
patch: 
  description: patch me
delete:
  description: delete me
head:
  description: top
options:
  description: choices
trace:
  description: find me
x-byebye: boebert`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n PathItem
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	yml2 := `get:
  description: get me
post:
  description: post me
servers:
  - url: https://pb33f.io
parameters: 
  - in: head
put:
  description: put me
patch: 
  description: patch me
delete:
  description: delete me
head:
  description: top
options:
  description: choices
trace:
  description: find me
x-byebye: boebert
description: a path item
summary: it's another path item`

	var idxNode2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml2), &idxNode2)
	idx2 := index.NewSpecIndex(&idxNode2)

	var n2 PathItem
	_ = low.BuildModel(idxNode2.Content[0], &n2)
	_ = n2.Build(context.Background(), nil, idxNode2.Content[0], idx2)

	// hash
	assert.Equal(t, n.Hash(), n2.Hash())
	assert.Equal(t, 1, orderedmap.Len(n.GetExtensions()))
	assert.NotNil(t, n.GetRootNode())
	assert.Nil(t, n.GetKeyNode())
	assert.NotNil(t, n.GetContext())
	assert.NotNil(t, n.GetIndex())
}

// https://github.com/pb33f/libopenapi/issues/388
func TestPathItem_CheckExtensionWithParametersValue_NoPanic(t *testing.T) {
	yml := `x-user_extension: parameters
get:
   description: test users 
   operationId: users`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n PathItem
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	assert.NotNil(t, n.RootNode)
}

func TestPathItem_AdditionalOperations(t *testing.T) {
	yml := `get:
  description: standard get operation
post:
  description: standard post operation
purge:
  description: purge operation for cache clearing
  operationId: purgeCache
  responses:
    '204':
      description: Cache cleared successfully
lock:
  description: lock operation for resource locking
  operationId: lockResource
  parameters:
    - name: timeout
      in: query
      schema:
        type: integer`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n PathItem
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	// test standard operations
	assert.NotNil(t, n.Get.Value)
	assert.Equal(t, "standard get operation", n.Get.Value.Description.Value)
	assert.NotNil(t, n.Post.Value)
	assert.Equal(t, "standard post operation", n.Post.Value.Description.Value)

	// test additional operations
	assert.NotNil(t, n.AdditionalOperations.Value)
	assert.Equal(t, 2, n.AdditionalOperations.Value.Len())

	var purgeOp low.NodeReference[*Operation]
	for k, v := range n.AdditionalOperations.Value.FromOldest() {
		if k.Value == "purge" {
			purgeOp = v
			break
		}
	}

	assert.NotNil(t, purgeOp)
	assert.Equal(t, "purge operation for cache clearing", purgeOp.Value.Description.Value)
	assert.Equal(t, "purgeCache", purgeOp.Value.OperationId.Value)

	var lockOp low.NodeReference[*Operation]
	for k, v := range n.AdditionalOperations.Value.FromOldest() {
		if k.Value == "lock" {
			lockOp = v
			break
		}
	}
	assert.NotNil(t, lockOp)
	assert.Equal(t, "lock operation for resource locking", lockOp.Value.Description.Value)
	assert.Equal(t, "lockResource", lockOp.Value.OperationId.Value)

	// test hash includes additional operations
	hash1 := n.Hash()
	n.AdditionalOperations = low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.NodeReference[*Operation]]]{}
	hash2 := n.Hash()
	assert.NotEqual(t, hash1, hash2)
}

func TestPathItem_AdditionalOperations_InCorrectLocation(t *testing.T) {
	yml := `get:
  description: standard get operation
post:
  description: standard post operation
additionalOperations:
  purge:
    description: purge operation for cache clearing
    operationId: purgeCache
    responses:
      '204':
        description: Cache cleared successfully
  lock:
    description: lock operation for resource locking
    operationId: lockResource
    parameters:
      - name: timeout
        in: query
        schema:
          type: integer
  cycle:
    $ref: '#/get'`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n PathItem
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	// test standard operations
	assert.NotNil(t, n.Get.Value)
	assert.Equal(t, "standard get operation", n.Get.Value.Description.Value)
	assert.NotNil(t, n.Post.Value)
	assert.Equal(t, "standard post operation", n.Post.Value.Description.Value)

	// test additional operations
	assert.NotNil(t, n.AdditionalOperations.Value)
	assert.Equal(t, 3, n.AdditionalOperations.Value.Len())

	var purgeOp low.NodeReference[*Operation]
	for k, v := range n.AdditionalOperations.Value.FromOldest() {
		if k.Value == "purge" {
			purgeOp = v
			break
		}
	}

	assert.NotNil(t, purgeOp)
	assert.Equal(t, "purge operation for cache clearing", purgeOp.Value.Description.Value)
	assert.Equal(t, "purgeCache", purgeOp.Value.OperationId.Value)

	var lockOp low.NodeReference[*Operation]
	for k, v := range n.AdditionalOperations.Value.FromOldest() {
		if k.Value == "lock" {
			lockOp = v
			break
		}
	}
	assert.NotNil(t, lockOp)
	assert.Equal(t, "lock operation for resource locking", lockOp.Value.Description.Value)
	assert.Equal(t, "lockResource", lockOp.Value.OperationId.Value)

	// test hash includes additional operations
	hash1 := n.Hash()
	n.AdditionalOperations = low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.NodeReference[*Operation]]]{}
	hash2 := n.Hash()
	assert.NotEqual(t, hash1, hash2)
}

func TestPathItem_AdditionalOperations_BadRef(t *testing.T) {
	yml := `additionalOperations:
  smellyCatSmellyCat:
    $ref: '#/WhatAreTheyFeedingYou'`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n PathItem
	_ = low.BuildModel(idxNode.Content[0], &n)
	err := n.Build(context.Background(), nil, idxNode.Content[0], idx)

	assert.Error(t, err)
	assert.Nil(t, n.AdditionalOperations.Value)

}

func TestPathItem_AdditionalOperations_BadRef_AtRoot(t *testing.T) {
	yml := `smellyCatSmellyCat:
  $ref: '#/ItsNotYourFault'`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n PathItem
	_ = low.BuildModel(idxNode.Content[0], &n)
	err := n.Build(context.Background(), nil, idxNode.Content[0], idx)

	assert.Error(t, err)
	assert.Nil(t, n.AdditionalOperations.Value)

}

func TestPathItem_Build_StandardOperationUnknownYAMLKey(t *testing.T) {
	// YAML keys matching unexported fields (e.g., "context") are silently ignored
	// by BuildModel; the build succeeds since the key is simply unrecognized.
	yml := `get:
  context: nope`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n PathItem
	_ = low.BuildModel(idxNode.Content[0], &n)
	err := n.Build(context.Background(), nil, idxNode.Content[0], idx)

	assert.NoError(t, err)
}

func TestPathItem_Build_AdditionalOperationsUnknownYAMLKey(t *testing.T) {
	// YAML keys matching unexported fields (e.g., "context") are silently ignored
	// by BuildModel; the build succeeds since the key is simply unrecognized.
	yml := `additionalOperations:
  purge:
    context: nope`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n PathItem
	_ = low.BuildModel(idxNode.Content[0], &n)
	err := n.Build(context.Background(), nil, idxNode.Content[0], idx)

	assert.NoError(t, err)
}

func TestResolveOperationReference_DocumentNode(t *testing.T) {
	refValue := "#/components/operations/getOp"
	refNode := utils.CreateRefNode(refValue)
	var resolvedDoc yaml.Node
	_ = yaml.Unmarshal([]byte("description: from-document-node"), &resolvedDoc)

	idx := index.NewSpecIndex(refNode)
	idx.SetMappedReferences(map[string]*index.Reference{
		refValue: {
			FullDefinition: refValue,
			Node:           &resolvedDoc,
			Index:          idx,
		},
	})

	foundCtx, resolvedNode, isRef, foundRef, foundRefNode, err := resolveOperationReference(context.Background(), refNode, idx)
	assert.NoError(t, err)
	assert.True(t, isRef)
	assert.Equal(t, refValue, foundRef)
	assert.Equal(t, refNode, foundRefNode)
	assert.Equal(t, yaml.MappingNode, resolvedNode.Kind)
	assert.Equal(t, "description", resolvedNode.Content[0].Value)
	assert.Equal(t, "from-document-node", resolvedNode.Content[1].Value)
	assert.NotNil(t, foundCtx.Value(index.FoundIndexKey))
}

func TestResolveOperationReference_EmptyTagNode(t *testing.T) {
	refValue := "#/components/operations/emptyTag"
	refNode := utils.CreateRefNode(refValue)

	resolved := &yaml.Node{
		Kind: yaml.SequenceNode,
		Tag:  "",
		Content: []*yaml.Node{
			{
				Kind: yaml.MappingNode,
				Tag:  "!!map",
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Tag: "!!str", Value: "description"},
					{Kind: yaml.ScalarNode, Tag: "!!str", Value: "from-empty-tag-node"},
				},
			},
		},
	}

	idx := index.NewSpecIndex(refNode)
	idx.SetMappedReferences(map[string]*index.Reference{
		refValue: {
			FullDefinition: refValue,
			Node:           resolved,
			Index:          idx,
		},
	})

	foundCtx, resolvedNode, isRef, foundRef, foundRefNode, err := resolveOperationReference(context.Background(), refNode, idx)
	assert.NoError(t, err)
	assert.True(t, isRef)
	assert.Equal(t, refValue, foundRef)
	assert.Equal(t, refNode, foundRefNode)
	assert.Equal(t, yaml.MappingNode, resolvedNode.Kind)
	assert.Equal(t, "description", resolvedNode.Content[0].Value)
	assert.Equal(t, "from-empty-tag-node", resolvedNode.Content[1].Value)
	assert.NotNil(t, foundCtx.Value(index.FoundIndexKey))
}
