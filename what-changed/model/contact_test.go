// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/what-changed/core"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestCompareContact_URLAdded(t *testing.T) {

	left := `name: buckaroo`

	right := `name: buckaroo
url: https://pb33f.io`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc lowbase.Contact
	var rDoc lowbase.Contact
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareContact(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, core.PropertyAdded, extChanges.Changes[0].ChangeType)

}

func TestCompareContact_URLRemoved(t *testing.T) {

	left := `name: buckaroo
url: https://pb33f.io`

	right := `name: buckaroo`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc lowbase.Contact
	var rDoc lowbase.Contact
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareContact(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, core.PropertyRemoved, extChanges.Changes[0].ChangeType)

}

func TestCompareContact_NameAdded(t *testing.T) {

	left := `url: https://pb33f.io`

	right := `url: https://pb33f.io
name: buckaroo`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc lowbase.Contact
	var rDoc lowbase.Contact
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareContact(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, core.PropertyAdded, extChanges.Changes[0].ChangeType)

}

func TestCompareContact_NameRemoved(t *testing.T) {

	left := `url: https://pb33f.io
name: buckaroo`

	right := `url: https://pb33f.io`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc lowbase.Contact
	var rDoc lowbase.Contact
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareContact(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, core.PropertyRemoved, extChanges.Changes[0].ChangeType)
}

func TestCompareContact_EmailAdded(t *testing.T) {

	left := `url: https://pb33f.io`

	right := `url: https://pb33f.io
email: buckaroo@pb33f.io`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc lowbase.Contact
	var rDoc lowbase.Contact
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareContact(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, core.PropertyAdded, extChanges.Changes[0].ChangeType)

}

func TestCompareContact_EmailRemoved(t *testing.T) {

	left := `url: https://pb33f.io
email: buckaroo@pb33f.io`

	right := `url: https://pb33f.io`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc lowbase.Contact
	var rDoc lowbase.Contact
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareContact(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, core.PropertyRemoved, extChanges.Changes[0].ChangeType)
}

func TestCompareContact_EmailModified(t *testing.T) {

	left := `url: https://pb33f.io
email: buckaroo@pb33f.io`

	right := `url: https://pb33f.io
email: dave@quobix.com`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc lowbase.Contact
	var rDoc lowbase.Contact
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareContact(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, core.Modified, extChanges.Changes[0].ChangeType)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
}

func TestCompareContact_EmailModifiedAndMoved(t *testing.T) {

	left := `email: buckaroo@pb33f.io
url: https://pb33f.io`

	right := `url: https://pb33f.io
email: dave@quobix.com`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc lowbase.Contact
	var rDoc lowbase.Contact
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareContact(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
}

func TestCompareContact_Identical(t *testing.T) {

	left := `email: buckaroo@pb33f.io
url: https://pb33f.io`

	right := `email: buckaroo@pb33f.io
url: https://pb33f.io`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc lowbase.Contact
	var rDoc lowbase.Contact
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareContact(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}
