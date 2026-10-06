// Copyright 2022-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
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

func TestPathItem_Build_ScalarRoot(t *testing.T) {
	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte("nope"), &idxNode)

	var n PathItem
	_ = low.BuildModel(idxNode.Content[0], &n)
	err := n.Build(context.Background(), idxNode.Content[0], idxNode.Content[0], nil)
	assert.NoError(t, err)
	assert.NotNil(t, n.GetRootNode())
	assert.NotNil(t, n.GetKeyNode())
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

func TestPathItem_Build_NullParameters_BuildsAllOperations(t *testing.T) {
	yml := `post:
  operationId: createThing
  parameters:
put:
  operationId: replaceThing
  parameters:
delete:
  operationId: deleteThing
  parameters:`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n PathItem
	_ = low.BuildModel(idxNode.Content[0], &n)
	err := n.Build(context.Background(), nil, idxNode.Content[0], idx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "input is not an array")

	// Each failed operation initializes its reference and contributes an error.
	joined, ok := err.(interface{ Unwrap() []error })
	if assert.True(t, ok) {
		assert.Len(t, joined.Unwrap(), 3)
	}
	assert.NotNil(t, n.Post.Value.Reference)
	assert.NotNil(t, n.Put.Value.Reference)
	assert.NotNil(t, n.Delete.Value.Reference)
	assert.False(t, n.Put.Value.IsReference())
}

func TestPathItem_Build_ErrorPreservesValidSiblings(t *testing.T) {
	// The larger case exercises the parallel additional-operation builder.
	for _, tc := range []struct {
		name   string
		count  int
		nested bool
	}{
		{"additional_operations", 2, true},
		{"parallel_additional_operations", 18, true},
		{"root_custom_operations", 2, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var spec strings.Builder
			spec.WriteString(`get:
  parameters:
post:
  responses:
    '201':
      description: created
put:
  parameters:
`)
			indent := ""
			if tc.nested {
				spec.WriteString("additionalOperations:\n")
				indent = "  "
			}
			for i := 0; i < tc.count; i++ {
				fmt.Fprintf(&spec, "%sMETHOD%d:\n", indent, i)
				if i%2 == 0 {
					fmt.Fprintf(&spec, "%s  parameters:\n", indent)
				} else {
					fmt.Fprintf(&spec, "%s  responses:\n%s    '200':\n%s      description: ok\n", indent, indent, indent)
				}
			}
			var root yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(spec.String()), &root))
			idx := index.NewSpecIndex(&root)
			var item PathItem
			require.NoError(t, low.BuildModel(root.Content[0], &item))
			err := item.Build(context.Background(), nil, root.Content[0], idx)
			require.Error(t, err)
			joined, ok := err.(interface{ Unwrap() []error })
			require.True(t, ok)
			require.Len(t, joined.Unwrap(), 2+tc.count/2)
			for _, buildErr := range joined.Unwrap() {
				assert.Contains(t, buildErr.Error(), "input is not an array")
			}

			require.NotNil(t, item.Post.Value.Responses.Value)
			response := item.Post.Value.Responses.Value.FindResponseByCode("201")
			require.NotNil(t, response)
			assert.Equal(t, "created", response.Value.Description.Value)
			require.NotNil(t, item.AdditionalOperations.Value)
			require.Equal(t, tc.count, item.AdditionalOperations.Value.Len())
			if tc.nested {
				assert.Equal(t, AdditionalOperationsLabel, item.AdditionalOperations.KeyNode.Value)
			} else {
				assert.Equal(t, "METHOD0", item.AdditionalOperations.KeyNode.Value)
			}
			assert.NotNil(t, item.AdditionalOperations.ValueNode)
			i := 0
			for key, op := range item.AdditionalOperations.Value.FromOldest() {
				require.NotNil(t, op.Value.Reference, key.Value)
				if i%2 == 1 {
					require.NotNil(t, op.Value.Responses.Value, key.Value)
					response := op.Value.Responses.Value.FindResponseByCode("200")
					require.NotNil(t, response, key.Value)
					assert.Equal(t, "ok", response.Value.Description.Value)
				}
				i++
			}
		})
	}
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
