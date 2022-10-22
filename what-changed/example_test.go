// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package what_changed

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestCompareExamples_SummaryModified(t *testing.T) {

	left := `summary: magic herbs`
	right := `summary: cure all`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc base.Example
	var rDoc base.Example
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	extChanges := CompareExamples(&lDoc, &rDoc)

	assert.Equal(t, extChanges.TotalChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, Modified, extChanges.Changes[0].ChangeType)
	assert.Equal(t, v3.SummaryLabel, extChanges.Changes[0].Property)
	assert.Equal(t, "magic herbs", extChanges.Changes[0].Original)
	assert.Equal(t, "cure all", extChanges.Changes[0].New)
}

func TestCompareExamples_SummaryAdded(t *testing.T) {

	left := `summary: magic herbs`
	right := `summary: magic herbs
description: cure all`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc base.Example
	var rDoc base.Example
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	extChanges := CompareExamples(&lDoc, &rDoc)

	assert.Equal(t, extChanges.TotalChanges(), 1)
	assert.Equal(t, PropertyAdded, extChanges.Changes[0].ChangeType)
	assert.Equal(t, v3.DescriptionLabel, extChanges.Changes[0].Property)
	assert.Equal(t, "cure all", extChanges.Changes[0].New)
}

func TestCompareExamples_ExtensionAdded(t *testing.T) {

	left := `summary: magic herbs`
	right := `summary: magic herbs
x-herbs: cure all`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc base.Example
	var rDoc base.Example
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	extChanges := CompareExamples(&lDoc, &rDoc)

	assert.Equal(t, extChanges.TotalChanges(), 1)
	assert.Equal(t, ObjectAdded, extChanges.ExtensionChanges.Changes[0].ChangeType)
	assert.Equal(t, "x-herbs", extChanges.ExtensionChanges.Changes[0].Property)
	assert.Equal(t, "cure all", extChanges.ExtensionChanges.Changes[0].New)
}

func TestCompareExamples_Identical(t *testing.T) {

	left := `summary: magic herbs`
	right := `summary: magic herbs`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc base.Example
	var rDoc base.Example
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	extChanges := CompareExamples(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}
