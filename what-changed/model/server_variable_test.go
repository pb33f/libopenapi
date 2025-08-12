// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"testing"

	"github.com/pkg-base/libopenapi/datamodel/low"
	v3 "github.com/pkg-base/libopenapi/datamodel/low/v3"
	"github.com/pkg-base/yaml"
	"github.com/stretchr/testify/assert"
)

func TestCompareServerVariables(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `description: hi
default: hello
enum:
  - one
  - two`

	right := `description: hi
default: hello
enum:
  - one
  - two`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.ServerVariable
	var rDoc v3.ServerVariable
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)

	// compare.
	extChanges := CompareServerVariables(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareServerVariables_EnumRemoved(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `description: hi
default: hello
enum:
  - one
  - two`

	right := `description: hi
default: hello
enum:
  - one`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.ServerVariable
	var rDoc v3.ServerVariable
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)

	// compare.
	extChanges := CompareServerVariables(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, extChanges.Changes[0].ChangeType)
}

func TestCompareServerVariables_Modified(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `description: hi
default: hello
enum:
  - one
  - two`

	right := `description: hi
default: hello
enum:
  - one
  - two
  - three`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.ServerVariable
	var rDoc v3.ServerVariable
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)

	// compare.
	extChanges := CompareServerVariables(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
}

func TestCompareServerVariables_Added(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `description: hi
enum:
  - one
  - two`

	right := `default: hello
description: hi
enum:
  - one
  - two`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.ServerVariable
	var rDoc v3.ServerVariable
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)

	// compare.
	extChanges := CompareServerVariables(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, PropertyAdded, extChanges.Changes[0].ChangeType)
}

func TestCompareServerVariables_Removed(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `description: hi
default: hello
enum:
  - one
  - two`

	right := `description: hi
enum:
  - one
  - two`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.ServerVariable
	var rDoc v3.ServerVariable
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)

	// compare.
	extChanges := CompareServerVariables(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, PropertyAdded, extChanges.Changes[0].ChangeType)
}

func TestCompareServerVariables_EnumAddedEdgeCase(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `description: hi
default: hello
enum:
  - one`

	right := `description: hi
default: hello
enum:
  - one
  - two
  - three`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.ServerVariable
	var rDoc v3.ServerVariable
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)

	// compare.
	extChanges := CompareServerVariables(&lDoc, &rDoc)
	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 2)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, extChanges.Changes[0].ChangeType)
	assert.Equal(t, ObjectAdded, extChanges.Changes[1].ChangeType)
}
