// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/v2"
	"github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestComparePaths_v2(t *testing.T) {

	left := `
/fresh/cake:
  get:
    description: a thing?
/battered/fish:
  post:
    description: a thong?
/crispy/chips:
  head:
    description: a thang?`

	right := left

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Paths
	var rDoc v2.Paths
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(nil, lNode.Content[0], nil)
	_ = rDoc.Build(nil, rNode.Content[0], nil)

	// compare.
	extChanges := ComparePaths(&rDoc, &lDoc)
	assert.Nil(t, extChanges)

}

func TestComparePaths_v2_ModifyOp(t *testing.T) {

	left := `/fresh/cake:
  get:
    description: a thing?
x-windows: dirty
/battered/fish:
  post:
    description: a thong?
/crispy/chips:
  head:
    description: a thang?`

	right := `/fresh/cake:
  get:
    description: well, it's nice to be updated
/battered/fish:
  post:
    description: love being edited
x-windows: washed
/crispy/chips:
  head:
    description: any one for tennis?`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Paths
	var rDoc v2.Paths
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(nil, lNode.Content[0], nil)
	_ = rDoc.Build(nil, rNode.Content[0], nil)

	// compare.
	extChanges := ComparePaths(&lDoc, &rDoc)
	assert.Equal(t, 4, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 4)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
}

func TestComparePaths_v2_AddPath(t *testing.T) {

	left := `/fresh/cake:
  get:
    description: a thing?
x-windows: dirty
/battered/fish:
  post:
    description: a thong?`

	right := `/fresh/cake:
  get:
    description: a thing?
x-windows: dirty
/battered/fish:
  post:
    description: a thong?
/crispy/chips:
  head:
    description: a thang?`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Paths
	var rDoc v2.Paths
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(nil, lNode.Content[0], nil)
	_ = rDoc.Build(nil, rNode.Content[0], nil)

	// compare.
	extChanges := ComparePaths(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, extChanges.Changes[0].ChangeType)
	assert.Equal(t, "/crispy/chips", extChanges.Changes[0].New)
}

func TestComparePaths_v2_RemovePath(t *testing.T) {

	left := `/fresh/cake:
  get:
    description: a thing?
x-windows: dirty
/battered/fish:
  post:
    description: a thong?`

	right := `/fresh/cake:
  get:
    description: a thing?
x-windows: dirty
/battered/fish:
  post:
    description: a thong?
/crispy/chips:
  head:
    description: a thang?`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Paths
	var rDoc v2.Paths
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(nil, lNode.Content[0], nil)
	_ = rDoc.Build(nil, rNode.Content[0], nil)

	// compare.
	extChanges := ComparePaths(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, "/crispy/chips", extChanges.Changes[0].Original)
}

func TestComparePaths_v3(t *testing.T) {

	left := `/fresh/cake:
  get:
    description: a thing?
/battered/fish:
  post:
    description: a thong?
/crispy/chips:
  head:
    description: a thang?`

	right := left

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Paths
	var rDoc v3.Paths
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(nil, lNode.Content[0], nil)
	_ = rDoc.Build(nil, rNode.Content[0], nil)

	// compare.
	extChanges := ComparePaths(&rDoc, &lDoc)
	assert.Nil(t, extChanges)

}

func TestComparePaths_v3_ModifyOp(t *testing.T) {

	left := `/fresh/cake:
  get:
    description: a thing?
x-windows: dirty
/battered/fish:
  post:
    description: a thong?
/crispy/chips:
  head:
    description: a thang?`

	right := `/fresh/cake:
  get:
    description: well, it's nice to be updated
/battered/fish:
  post:
    description: love being edited
x-windows: washed
/crispy/chips:
  head:
    description: any one for tennis?`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Paths
	var rDoc v3.Paths
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(nil, lNode.Content[0], nil)
	_ = rDoc.Build(nil, rNode.Content[0], nil)

	// compare.
	extChanges := ComparePaths(&lDoc, &rDoc)
	assert.Equal(t, 4, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 4)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
}

func TestComparePaths_v3_AddPath(t *testing.T) {

	left := `/fresh/cake:
  get:
    description: a thing?
x-windows: dirty
/battered/fish:
  post:
    description: a thong?
/crispy/chips:
  head:
    description: a thang?`

	right := `/fresh/cake:
  get:
    description: a thing?
x-windows: dirty
/battered/fish:
  post:
    description: a thong?
/crispy/chips:
  head:
    description: a thang?
/mushy/peas:
  post:
    description: love being edited
x-windows: dirty`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Paths
	var rDoc v3.Paths
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(nil, lNode.Content[0], nil)
	_ = rDoc.Build(nil, rNode.Content[0], nil)

	// compare.
	extChanges := ComparePaths(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, extChanges.Changes[0].ChangeType)
	assert.Equal(t, "/mushy/peas", extChanges.Changes[0].New)
}

func TestComparePaths_v3_RemovePath(t *testing.T) {

	left := `/fresh/cake:
  get:
    description: a thing?
x-windows: dirty
/battered/fish:
  post:
    description: a thong?
/crispy/chips:
  head:
    description: a thang?`

	right := `/fresh/cake:
  get:
    description: a thing?
x-windows: dirty
/battered/fish:
  post:
    description: a thong?
/crispy/chips:
  head:
    description: a thang?
/mushy/peas:
  post:
    description: love being edited
x-windows: dirty`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Paths
	var rDoc v3.Paths
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(nil, lNode.Content[0], nil)
	_ = rDoc.Build(nil, rNode.Content[0], nil)

	// compare.
	extChanges := ComparePaths(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, "/mushy/peas", extChanges.Changes[0].Original)
}
