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

func TestCompareExamplesV2(t *testing.T) {

    left := `summary: magic herbs`
    right := `summary: cure all`

    var lNode, rNode yaml.Node
    _ = yaml.Unmarshal([]byte(left), &lNode)
    _ = yaml.Unmarshal([]byte(right), &rNode)

    // create low level objects
    var lDoc v2.Examples
    var rDoc v2.Examples
    _ = low.BuildModel(lNode.Content[0], &lDoc)
    _ = low.BuildModel(rNode.Content[0], &rDoc)
    _ = lDoc.Build(lNode.Content[0], nil)
    _ = rDoc.Build(rNode.Content[0], nil)

    extChanges := CompareExamplesV2(&lDoc, &rDoc)
    assert.Equal(t, extChanges.TotalChanges(), 1)
    assert.Len(t, extChanges.GetAllChanges(), 1)
    assert.Equal(t, 0, extChanges.TotalBreakingChanges())
    assert.Equal(t, Modified, extChanges.Changes[0].ChangeType)
    assert.Equal(t, v3.SummaryLabel, extChanges.Changes[0].Property)
    assert.Equal(t, "magic herbs", extChanges.Changes[0].Original)
    assert.Equal(t, "cure all", extChanges.Changes[0].New)
}

func TestCompareExamplesV2_Add(t *testing.T) {

    left := `summary: magic herbs`
    right := `summary: magic herbs
yummy: coffee`

    var lNode, rNode yaml.Node
    _ = yaml.Unmarshal([]byte(left), &lNode)
    _ = yaml.Unmarshal([]byte(right), &rNode)

    // create low level objects
    var lDoc v2.Examples
    var rDoc v2.Examples
    _ = low.BuildModel(lNode.Content[0], &lDoc)
    _ = low.BuildModel(rNode.Content[0], &rDoc)
    _ = lDoc.Build(lNode.Content[0], nil)
    _ = rDoc.Build(rNode.Content[0], nil)

    extChanges := CompareExamplesV2(&lDoc, &rDoc)
    assert.Equal(t, extChanges.TotalChanges(), 1)
    assert.Len(t, extChanges.GetAllChanges(), 1)
    assert.Equal(t, 0, extChanges.TotalBreakingChanges())
    assert.Equal(t, ObjectAdded, extChanges.Changes[0].ChangeType)
}

func TestCompareExamplesV2_Remove(t *testing.T) {

    left := `summary: magic herbs`
    right := `summary: magic herbs
yummy: coffee`

    var lNode, rNode yaml.Node
    _ = yaml.Unmarshal([]byte(left), &lNode)
    _ = yaml.Unmarshal([]byte(right), &rNode)

    // create low level objects
    var lDoc v2.Examples
    var rDoc v2.Examples
    _ = low.BuildModel(lNode.Content[0], &lDoc)
    _ = low.BuildModel(rNode.Content[0], &rDoc)
    _ = lDoc.Build(lNode.Content[0], nil)
    _ = rDoc.Build(rNode.Content[0], nil)

    extChanges := CompareExamplesV2(&rDoc, &lDoc)
    assert.Equal(t, extChanges.TotalChanges(), 1)
    assert.Len(t, extChanges.GetAllChanges(), 1)
    assert.Equal(t, 0, extChanges.TotalBreakingChanges())
    assert.Equal(t, ObjectRemoved, extChanges.Changes[0].ChangeType)
}

func TestCompareExamplesV2_Identical(t *testing.T) {

    left := `summary: magic herbs`
    right := left

    var lNode, rNode yaml.Node
    _ = yaml.Unmarshal([]byte(left), &lNode)
    _ = yaml.Unmarshal([]byte(right), &rNode)

    // create low level objects
    var lDoc v2.Examples
    var rDoc v2.Examples
    _ = low.BuildModel(lNode.Content[0], &lDoc)
    _ = low.BuildModel(rNode.Content[0], &rDoc)
    _ = lDoc.Build(lNode.Content[0], nil)
    _ = rDoc.Build(rNode.Content[0], nil)

    extChanges := CompareExamplesV2(&rDoc, &lDoc)
	assert.Nil(t, extChanges)
}
