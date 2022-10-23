// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/what-changed/core"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestCompareXML_NameChanged(t *testing.T) {

	left := `name: xml thing
namespace: something
prefix: another
attribute: true
wrapped: true`

	right := `namespace: something
prefix: another
name: changed xml thing
attribute: true
wrapped: true`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc base.XML
	var rDoc base.XML
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareXML(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, core.Modified, extChanges.Changes[0].ChangeType)

}

func TestCompareXML_NameRemoved(t *testing.T) {

	left := `name: xml thing
namespace: something
prefix: another
attribute: true
wrapped: true`

	right := `wrapped: true
prefix: another
attribute: true
namespace: something`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc base.XML
	var rDoc base.XML
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareXML(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, core.PropertyRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())

}

func TestCompareXML_ExtensionAdded(t *testing.T) {

	left := `name: xml thing
namespace: something
prefix: another
attribute: true
wrapped: true`

	right := `name: xml thing
namespace: something
prefix: another
attribute: true
wrapped: true
x-coffee: time`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc base.XML
	var rDoc base.XML
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareXML(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, core.ObjectAdded, extChanges.ExtensionChanges.Changes[0].ChangeType)

}

func TestCompareXML_Identical(t *testing.T) {

	left := `name: xml thing
namespace: something
prefix: another
attribute: true
wrapped: true`

	right := `name: xml thing
namespace: something
prefix: another
attribute: true
wrapped: true`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc base.XML
	var rDoc base.XML
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareXML(&lDoc, &rDoc)
	assert.Nil(t, extChanges)

}
