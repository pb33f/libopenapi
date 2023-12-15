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
	"gopkg.in/yaml.v3"
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
}
