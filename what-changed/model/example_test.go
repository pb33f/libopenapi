// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"context"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
)

func TestCompareExamples_SummaryModified(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `summary: magic herbs`
	right := `summary: cure all`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc base.Example
	var rDoc base.Example
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	extChanges := CompareExamples(&lDoc, &rDoc)

	assert.Equal(t, extChanges.TotalChanges(), 1)
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, Modified, extChanges.Changes[0].ChangeType)
	assert.Equal(t, v3.SummaryLabel, extChanges.Changes[0].Property)
	assert.Equal(t, "magic herbs", extChanges.Changes[0].Original)
	assert.Equal(t, "cure all", extChanges.Changes[0].New)
}

func TestCompareExamples_ObjectValue(t *testing.T) {
	for _, tc := range []struct {
		name, left, right string
	}{
		{"two members modified", "{id_1: 11, id_2: 21}", "{id_1: 10, id_2: 20}"},
		{"member added", "{}", "{id: 1}"},
		{"member removed", "{id: 1}", "{}"},
		{"sequence reordered inside object", "{items: [1, 2]}", "{items: [2, 1]}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			left := buildTestExample(t, "value: "+tc.left)
			right := buildTestExample(t, "value: "+tc.right)
			changes := CompareExamples(left, right)
			require.NotNil(t, changes)
			require.Len(t, changes.GetAllChanges(), 1)
			change := changes.Changes[0]
			assert.Equal(t, v3.ValueLabel, change.Property)
			assert.Equal(t, Modified, change.ChangeType)
			assert.False(t, change.Breaking)
			assert.Equal(t, tc.left+"\n", change.OriginalEncoded)
			assert.Equal(t, tc.right+"\n", change.NewEncoded)
		})
	}

	left := buildTestExample(t, "value: {a: 1, b: 2}")
	reordered := buildTestExample(t, "value: {b: 2, a: 1}")
	assert.Nil(t, CompareExamples(left, reordered))

	absent := buildTestExample(t, "summary: example")
	present := buildTestExample(t, "summary: example\nvalue: {}")
	added := CompareExamples(absent, present).GetAllChanges()
	require.Len(t, added, 1)
	assert.Equal(t, PropertyAdded, added[0].ChangeType)
	removed := CompareExamples(present, absent).GetAllChanges()
	require.Len(t, removed, 1)
	assert.Equal(t, PropertyRemoved, removed[0].ChangeType)
}

func TestCompareExamples_SummaryAdded(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `summary: magic herbs`
	right := `summary: magic herbs
description: cure all`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc base.Example
	var rDoc base.Example
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	extChanges := CompareExamples(&lDoc, &rDoc)

	assert.Equal(t, extChanges.TotalChanges(), 1)
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, PropertyAdded, extChanges.Changes[0].ChangeType)
	assert.Equal(t, v3.DescriptionLabel, extChanges.Changes[0].Property)
	assert.Equal(t, "cure all", extChanges.Changes[0].New)
}

func TestCompareExamples_ExtensionAdded(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `summary: magic herbs`
	right := `summary: magic herbs
x-herbs: cure all`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc base.Example
	var rDoc base.Example
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	extChanges := CompareExamples(&lDoc, &rDoc)

	assert.Equal(t, extChanges.TotalChanges(), 1)
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, ObjectAdded, extChanges.ExtensionChanges.Changes[0].ChangeType)
	assert.Equal(t, "x-herbs", extChanges.ExtensionChanges.Changes[0].Property)
	assert.Equal(t, "cure all", extChanges.ExtensionChanges.Changes[0].New)
}

func TestCompareExamples_Identical(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `summary: magic herbs`
	right := `summary: magic herbs`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc base.Example
	var rDoc base.Example
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	extChanges := CompareExamples(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareExamples_Date(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `value:
  date: 2022-12-29`

	right := `value:
  date: "2022-12-29"`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc base.Example
	var rDoc base.Example
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	changes := CompareExamples(&lDoc, &rDoc)

	assert.Equal(t, 1, changes.TotalChanges())
}

func TestCompareExamples_DataValueAdded(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `summary: test example
description: basic example`

	right := `summary: test example
description: basic example
dataValue:
  name: John Doe
  age: 30`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc base.Example
	var rDoc base.Example
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	extChanges := CompareExamples(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges()) // adding dataValue is not breaking

	allChanges := extChanges.GetAllChanges()
	assert.Equal(t, PropertyAdded, allChanges[0].ChangeType)
	assert.Equal(t, base.DataValueLabel, allChanges[0].Property)
}

func TestCompareExamples_SerializedValueAdded(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `summary: test example
description: basic example`

	right := `summary: test example
description: basic example
serializedValue: '{"name":"John Doe","age":30}'`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc base.Example
	var rDoc base.Example
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	extChanges := CompareExamples(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges()) // adding serializedValue is not breaking

	allChanges := extChanges.GetAllChanges()
	assert.Equal(t, PropertyAdded, allChanges[0].ChangeType)
	assert.Equal(t, base.SerializedValueLabel, allChanges[0].Property)
}
