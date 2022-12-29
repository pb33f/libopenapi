// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	v2 "github.com/pb33f/libopenapi/datamodel/low/v2"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestComparePathItem_V2(t *testing.T) {

	left := `get:
  description: get me
put:
  description: put me
post:
  description: post me
delete:
  description: delete me
patch:
  description: patch me
options:
  description: options
head:
  description: head
parameters:
  - in: head
x-thing: thang.`

	right := left

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.PathItem
	var rDoc v2.PathItem
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := ComparePathItems(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestComparePathItem_V2_ModifyOps(t *testing.T) {

	left := `get:
  description: get me
put:
  description: put me
post:
  description: post me
delete:
  description: delete me
patch:
  description: patch me
options:
  description: options
head:
  description: head
parameters:
  - in: head
x-thing: thang.`

	right := `get:
  description: get me out
put:
  description: put me in
post:
  description: post me there
delete:
  description: delete me please
patch:
  description: patch me now
options:
  description: options vested
head:
  description: heads up
parameters:
  - in: head
x-thing: ding-a-ling`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.PathItem
	var rDoc v2.PathItem
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := ComparePathItems(&lDoc, &rDoc)
	assert.Equal(t, 8, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
}

func TestComparePathItem_V2_ModifyParam(t *testing.T) {

	left := `get:
  description: get me
parameters:
  - name: cake
    in: space
  - in: head
    name: eggs`

	right := `get:
  description: get me
parameters:
  - name: cake
    in: love
  - in: code
    name: eggs`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.PathItem
	var rDoc v2.PathItem
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := ComparePathItems(&lDoc, &rDoc)
	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Equal(t, 2, extChanges.TotalBreakingChanges())
	assert.Equal(t, Modified, extChanges.ParameterChanges[0].Changes[0].ChangeType)
	assert.Equal(t, v3.InLabel, extChanges.ParameterChanges[0].Changes[0].Property)
}

func TestComparePathItem_V2_AddParam(t *testing.T) {

	left := `get:
  description: get me
parameters:
  - name: cake
    in: love
  - in: code
    name: eggs`

	right := `get:
  description: get me
parameters:
  - name: cake
    in: love
  - in: code
    name: eggs
  - in: tune
    name: melody`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.PathItem
	var rDoc v2.PathItem
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := ComparePathItems(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, extChanges.Changes[0].ChangeType)
}

func TestComparePathItem_V2_RemoveParam(t *testing.T) {

	left := `get:
  description: get me
parameters:
  - name: cake
    in: love
  - in: code
    name: eggs`

	right := `get:
  description: get me
parameters:
  - name: cake
    in: love
  - in: code
    name: eggs
  - in: tune
    name: melody`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.PathItem
	var rDoc v2.PathItem
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := ComparePathItems(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, extChanges.Changes[0].ChangeType)
}

func TestComparePathItem_V2_AddParametersToPath(t *testing.T) {

	left := `get:
  description: get me`

	right := `get:
  description: get me
parameters:
  - name: cake
    in: love
  - in: code
    name: eggs
  - in: tune
    name: melody`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.PathItem
	var rDoc v2.PathItem
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := ComparePathItems(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, PropertyAdded, extChanges.Changes[0].ChangeType)
	assert.Equal(t, v3.ParametersLabel, extChanges.Changes[0].Property)

}

func TestComparePathItem_V2_RemoveParametersToPath(t *testing.T) {

	left := `get:
  description: get me
parameters:
  - name: cake
    in: love
  - in: code
    name: eggs
  - in: tune
    name: melody`

	right := `get:
  description: get me`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.PathItem
	var rDoc v2.PathItem
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := ComparePathItems(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())

}

func TestComparePathItem_V2_AddMethods(t *testing.T) {

	left := `parameters:
  - in: head`

	right := `get:
  description: get me out
put:
  description: put me in
post:
  description: post me there
delete:
  description: delete me please
patch:
  description: patch me now
options:
  description: options vested
head:
  description: heads up
parameters:
  - in: head`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.PathItem
	var rDoc v2.PathItem
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := ComparePathItems(&lDoc, &rDoc)
	assert.Equal(t, 7, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
}
func TestComparePathItem_V2_RemoveMethods(t *testing.T) {

	left := `parameters:
  - in: head`

	right := `get:
  description: get me out
put:
  description: put me in
post:
  description: post me there
delete:
  description: delete me please
patch:
  description: patch me now
options:
  description: options vested
head:
  description: heads up
parameters:
  - in: head`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.PathItem
	var rDoc v2.PathItem
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := ComparePathItems(&rDoc, &lDoc)
	assert.Equal(t, 7, extChanges.TotalChanges())
	assert.Equal(t, 7, extChanges.TotalBreakingChanges())
}

func TestComparePathItem_V3(t *testing.T) {

	left := `summary: something
description: nice
get:
  description: get me
put:
  description: put me
post:
  description: post me
delete:
  description: delete me
patch:
  description: patch me
options:
  description: options
head:
  description: head
trace:
  description: outerspace
servers:
  - url: https://pb33f.io
parameters:
  - in: head
x-thing: thang.`

	right := left

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.PathItem
	var rDoc v3.PathItem
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := ComparePathItems(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestComparePathItem_V3_Modify(t *testing.T) {

	left := `summary: something
description: nice
get:
  description: get me
put:
  description: put me
post:
  description: post me
delete:
  description: delete me
patch:
  description: patch me
options:
  description: options
head:
  description: head
trace:
  description: outerspace
servers:
  - url: https://pb33f.io
parameters:
  - in: head
x-thing: thang.`

	right := `summary: something cute
description: nice puppy
get:
  description: get me out
put:
  description: put me in
post:
  description: post me back
delete:
  description: delete me please
patch:
  description: patch me now
options:
  description: options when
head:
  description: head down
trace:
  description: outerspace race
servers:
  - url: https://pb33f.io
    description: beefy goodness
parameters:
  - in: head
x-thing: dang.`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.PathItem
	var rDoc v3.PathItem
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := ComparePathItems(&lDoc, &rDoc)
	assert.Equal(t, 12, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
}

func TestComparePathItem_V3_AddParams(t *testing.T) {

	left := `summary: something`

	right := `summary: something
parameters:
  - in: head`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.PathItem
	var rDoc v3.PathItem
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := ComparePathItems(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, PropertyAdded, extChanges.Changes[0].ChangeType)
	assert.Equal(t, v3.ParametersLabel, extChanges.Changes[0].Property)
}

func TestComparePathItem_V3_RemoveParams(t *testing.T) {

	left := `summary: something`

	right := `summary: something
parameters:
  - in: head`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.PathItem
	var rDoc v3.PathItem
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := ComparePathItems(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, PropertyRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, v3.ParametersLabel, extChanges.Changes[0].Property)
}

func TestComparePathItem_V3_AddMethods(t *testing.T) {

	left := `summary: something`

	right := `summary: something
get:
  description: get me out
put:
  description: put me in
post:
  description: post me back
delete:
  description: delete me please
patch:
  description: patch me now
options:
  description: options when
head:
  description: head down
trace:
  description: outerspace race`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.PathItem
	var rDoc v3.PathItem
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := ComparePathItems(&lDoc, &rDoc)
	assert.Equal(t, 8, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
}

func TestComparePathItem_V3_RemoveMethods(t *testing.T) {

	left := `summary: something`

	right := `summary: something
get:
  description: get me out
put:
  description: put me in
post:
  description: post me back
delete:
  description: delete me please
patch:
  description: patch me now
options:
  description: options when
head:
  description: head down
trace:
  description: outerspace race`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.PathItem
	var rDoc v3.PathItem
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := ComparePathItems(&rDoc, &lDoc)
	assert.Equal(t, 8, extChanges.TotalChanges())
	assert.Equal(t, 8, extChanges.TotalBreakingChanges())
}

func TestComparePathItem_V3_ChangeParam(t *testing.T) {

	left := `get:
  operationId: listBurgerDressings
  parameters:
    - in: query
      name: burgerId`

	right := `get:
  operationId: listBurgerDressings
  parameters:
    - in: head
      name: burgerId`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.PathItem
	var rDoc v3.PathItem
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := ComparePathItems(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
}
