// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package what_changed

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/v2"
	"github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestCompareParameters_V3(t *testing.T) {

	left := `name: a param`
	right := `name: a parama`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Parameter
	var rDoc v3.Parameter
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareParameters(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
}

func TestCompareParameters_V3_Schema(t *testing.T) {

	left := `schema:
  description: something new`
	right := `schema:
  description: a changed thing`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Parameter
	var rDoc v3.Parameter
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareParameters(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, 1, extChanges.SchemaChanges.TotalChanges())

}

func TestCompareParameters_V3_SchemaAdd(t *testing.T) {

	left := `description: hello`
	right := `description: hello
schema:
  description: a changed thing`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Parameter
	var rDoc v3.Parameter
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareParameters(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, extChanges.Changes[0].ChangeType)

}

func TestCompareParameters_V3_SchemaRemove(t *testing.T) {

	left := `description: hello`
	right := `description: hello
schema:
  description: a changed thing`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Parameter
	var rDoc v3.Parameter
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareParameters(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, extChanges.Changes[0].ChangeType)

}

func TestCompareParameters_V3_Extensions(t *testing.T) {

	left := `x-thing: thang`
	right := `x-thing: dang`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Parameter
	var rDoc v3.Parameter
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareParameters(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, 1, extChanges.ExtensionChanges.TotalChanges())

}

//func TestCompareParameters_V3_ExampleChange(t *testing.T) {
//
//	left := `example: a string`
//	right := `example:
//  now: an object`
//
//	var lNode, rNode yaml.Node
//	_ = yaml.Unmarshal([]byte(left), &lNode)
//	_ = yaml.Unmarshal([]byte(right), &rNode)
//
//	// create low level objects
//	var lDoc v3.Parameter
//	var rDoc v3.Parameter
//	_ = low.BuildModel(&lNode, &lDoc)
//	_ = low.BuildModel(&rNode, &rDoc)
//	_ = lDoc.Build(lNode.Content[0], nil)
//	_ = rDoc.Build(rNode.Content[0], nil)
//
//	// compare.
//	extChanges := CompareParameters(&lDoc, &rDoc)
//	assert.Equal(t, 1, extChanges.TotalChanges())
//	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
//	assert.Equal(t, 1, extChanges.ExtensionChanges.TotalChanges())
//}

func TestCompareParameters_V3_ExampleEqual(t *testing.T) {

	left := `example: a string`
	right := `example: a string`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Parameter
	var rDoc v3.Parameter
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareParameters(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareParameters_V3_ExampleAdd(t *testing.T) {

	left := `description: something`
	right := `description: something
example: a string`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Parameter
	var rDoc v3.Parameter
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare
	extChanges := CompareParameters(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, PropertyAdded, extChanges.Changes[0].ChangeType)
}

func TestCompareParameters_V3_ExampleRemove(t *testing.T) {

	left := `description: something`
	right := `description: something
example: a string`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Parameter
	var rDoc v3.Parameter
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare
	extChanges := CompareParameters(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, PropertyRemoved, extChanges.Changes[0].ChangeType)
}

func TestCompareParameters_V2(t *testing.T) {

	left := `name: a param`
	right := `name: a parama`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Parameter
	var rDoc v2.Parameter
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareParameters(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
}

//func TestCompareParameters_V2_Extensions(t *testing.T) {
//
//	left := `x-thing: thang`
//	right := `x-thing: dang`
//
//	var lNode, rNode yaml.Node
//	_ = yaml.Unmarshal([]byte(left), &lNode)
//	_ = yaml.Unmarshal([]byte(right), &rNode)
//
//	// create low level objects
//	var lDoc v2.Parameter
//	var rDoc v2.Parameter
//	_ = low.BuildModel(&lNode, &lDoc)
//	_ = low.BuildModel(&rNode, &rDoc)
//	_ = lDoc.Build(lNode.Content[0], nil)
//	_ = rDoc.Build(rNode.Content[0], nil)
//
//	// compare.
//	extChanges := CompareParameters(&lDoc, &rDoc)
//	assert.Equal(t, 1, extChanges.TotalChanges())
//	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
//	assert.Equal(t, Modified, extChanges.Changes[0].ChangeType)
//	assert.Equal(t, 1, extChanges.ExtensionChanges.TotalChanges())
//
//}
