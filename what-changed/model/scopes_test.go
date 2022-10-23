// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	v2 "github.com/pb33f/libopenapi/datamodel/low/v2"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestCompareScopes(t *testing.T) {

	left := `pizza: pie
lemon: sky
x-nugget: chicken`
	right := `pizza: pie
lemon: sky
x-nugget: chicken`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Scopes
	var rDoc v2.Scopes
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareScopes(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareScopes_Modified(t *testing.T) {

	left := `pizza: sky
lemon: sky
x-nugget: chicken`
	right := `pizza: pie
lemon: sky
x-nugget: chicken`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Scopes
	var rDoc v2.Scopes
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareScopes(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, Modified, extChanges.Changes[0].ChangeType)
	assert.Equal(t, "pie", extChanges.Changes[0].New)
}

func TestCompareScopes_Added(t *testing.T) {

	left := `pizza: sky
x-nugget: chicken`
	right := `pizza: sky
lemon: sky
x-nugget: chicken`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Scopes
	var rDoc v2.Scopes
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareScopes(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, extChanges.Changes[0].ChangeType)
	assert.Equal(t, "sky", extChanges.Changes[0].New)
	assert.Equal(t, "lemon", extChanges.Changes[0].NewObject)
}

func TestCompareScopes_Removed_ChangeExt(t *testing.T) {

	left := `pizza: sky
x-nugget: chicken`
	right := `pizza: sky
lemon: sky
x-nugget: soup`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Scopes
	var rDoc v2.Scopes
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareScopes(&rDoc, &lDoc)
	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, "sky", extChanges.Changes[0].Original)
	assert.Equal(t, "lemon", extChanges.Changes[0].OriginalObject)
}
