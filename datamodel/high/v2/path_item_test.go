// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	lowV2 "github.com/pb33f/libopenapi/datamodel/low/v2"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestPathItem_GetOperations(t *testing.T) {
	yml := `get:
  description: get
put:
  description: put
post:
  description: post
patch:
  description: patch
delete:
  description: delete
head:
  description: head
options:
  description: options
`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n lowV2.PathItem
	_ = low.BuildModel(&idxNode, &n)
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	r := NewPathItem(&n)

	assert.Equal(t, 7, orderedmap.Len(r.GetOperations()))
}

func TestPathItem_GetOperations_NoLow(t *testing.T) {
	pi := &PathItem{
		Delete: &Operation{},
		Post:   &Operation{},
		Get:    &Operation{},
	}
	ops := pi.GetOperations()

	expectedOrderOfOps := []string{"get", "post", "delete"}
	actualOrder := []string{}

	for op := range ops.KeysFromOldest() {
		actualOrder = append(actualOrder, op)
	}

	assert.Equal(t, expectedOrderOfOps, actualOrder)
}

func TestPathItem_GetOperations_LowWithUnsetOperations(t *testing.T) {
	pi := &PathItem{
		Delete: &Operation{},
		Post:   &Operation{},
		Get:    &Operation{},
		low:    &lowV2.PathItem{},
	}
	ops := pi.GetOperations()

	expectedOrderOfOps := []string{"get", "post", "delete"}
	actualOrder := []string{}

	for op := range ops.KeysFromOldest() {
		actualOrder = append(actualOrder, op)
	}

	assert.Equal(t, expectedOrderOfOps, actualOrder)
}

func TestPathItem_NewPathItem_WithParameters(t *testing.T) {
	pi := NewPathItem(&lowV2.PathItem{
		Parameters: low.NodeReference[[]low.ValueReference[*lowV2.Parameter]]{
			Value: []low.ValueReference[*lowV2.Parameter]{
				{
					Value: &lowV2.Parameter{},
				},
			},
			ValueNode: &yaml.Node{},
		},
	})
	assert.NotNil(t, pi.Parameters)
}
