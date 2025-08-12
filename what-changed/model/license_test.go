// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"context"
	"testing"

	"github.com/pkg-base/libopenapi/datamodel/low"
	lowbase "github.com/pkg-base/libopenapi/datamodel/low/base"
	"github.com/pkg-base/yaml"
	"github.com/stretchr/testify/assert"
)

func TestCompareLicense_URLAdded(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `name: buckaroo`

	right := `name: buckaroo
url: https://pb33f.io`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc lowbase.License
	var rDoc lowbase.License
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareLicense(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, PropertyAdded, extChanges.Changes[0].ChangeType)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
}

func TestCompareLicense_URLRemoved(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `name: buckaroo
url: https://pb33f.io`

	right := `name: buckaroo`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc lowbase.License
	var rDoc lowbase.License
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareLicense(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, PropertyRemoved, extChanges.Changes[0].ChangeType)
}

func TestCompareLicense_NameAdded(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `url: https://pb33f.io`

	right := `url: https://pb33f.io
name: buckaroo`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc lowbase.License
	var rDoc lowbase.License
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareLicense(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, PropertyAdded, extChanges.Changes[0].ChangeType)
}

func TestCompareLicense_NameRemoved(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `url: https://pb33f.io`

	right := `url: https://pb33f.io
name: buckaroo`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc lowbase.License
	var rDoc lowbase.License
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareLicense(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, PropertyAdded, extChanges.Changes[0].ChangeType)
}

func TestCompareLicense_URLModified(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `url: https://pb33f.io`

	right := `url: https://pb33f.io/new`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc lowbase.License
	var rDoc lowbase.License
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareLicense(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, Modified, extChanges.Changes[0].ChangeType)
}

func TestCompareLicense_URLModifiedAndMoved(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `name: buckaroo
url: https://pb33f.io`

	right := `url: https://pb33f.io/new`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc lowbase.License
	var rDoc lowbase.License
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareLicense(&lDoc, &rDoc)
	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 2)
	assert.Equal(t, Modified, extChanges.Changes[0].ChangeType)
	assert.Equal(t, PropertyRemoved, extChanges.Changes[1].ChangeType)
}

func TestCompareLicense_Identical(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `name: buckaroo
url: https://pb33f.io`

	right := `name: buckaroo
url: https://pb33f.io`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc lowbase.License
	var rDoc lowbase.License
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareLicense(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareLicense_Identifier(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `name: buckaroo
identifier: https://pb33f.io`

	right := `name: buckaroo
identifier: https://pb33f.io2`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc lowbase.License
	var rDoc lowbase.License
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareLicense(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, Modified, extChanges.Changes[0].ChangeType)
	assert.Equal(t, "identifier", extChanges.Changes[0].Property)
}

func TestCompareLicense_extension(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `name: buckaroo
identifier: https://pb33f.io
x-truc: hello`

	right := `name: buckaroo
identifier: https://pb33f.io
x-truc: hey`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc lowbase.License
	var rDoc lowbase.License
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareLicense(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
}
