// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestCompareServers(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `url: https://pb33f.io
description: a server
variables:
  thing:
    enum:
      - choccy
      - biccy
    default: choccy`

	right := `url: https://pb33f.io
description: a server
variables:
  thing:
    enum:
      - choccy
      - biccy
    default: choccy`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Server
	var rDoc v3.Server
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareServers(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareServers_Modified(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `url: https://pb33f.io
description: a server
variables:
  thing:
    enum:
      - choccy
      - biccy
    default: choccy`

	right := `url: https://pb33f.io/hotness
description: a server that is not
variables:
  thing:
    enum:
      - choccy
      - biccy
    default: biccy`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Server
	var rDoc v3.Server
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareServers(&lDoc, &rDoc)
	assert.Equal(t, 3, extChanges.TotalChanges())

	assert.Equal(t, 2, extChanges.TotalBreakingChanges())
}

func TestCompareServers_Added(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `url: https://pb33f.io
variables:
  thing:
    enum:
      - choccy
      - biccy
    default: choccy`

	right := `url: https://pb33f.io
description: a server
variables:
  thing:
    enum:
      - choccy
      - biccy
      - tea
    default: choccy`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Server
	var rDoc v3.Server
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareServers(&lDoc, &rDoc)
	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 2)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, PropertyAdded, extChanges.Changes[0].ChangeType)
	assert.Equal(t, ObjectAdded, extChanges.ServerVariableChanges["thing"].Changes[0].ChangeType)
}

func TestCompareServers_Removed(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `url: https://pb33f.io
variables:
  thing:
    enum:
      - choccy
      - biccy
    default: choccy`

	right := `url: https://pb33f.io
description: a server
variables:
  thing:
    enum:
      - choccy
      - biccy
      - tea
    default: choccy`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Server
	var rDoc v3.Server
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareServers(&rDoc, &lDoc)
	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 2)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, PropertyRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, ObjectRemoved, extChanges.ServerVariableChanges["thing"].Changes[0].ChangeType)
}

func TestCompareServers_Extensions(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `url: https://pb33f.io
x-coffee: hot`

	right := `url: https://pb33f.io
x-coffee: cold`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Server
	var rDoc v3.Server
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareServers(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.ExtensionChanges.GetAllChanges(), 1)
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Empty(t, extChanges.PropertyChanges)
}

func TestCompareServers_NoExtensions(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `url: https://pb33f.io
description: a server`

	right := `url: https://pb33f.io
description: a server`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Server
	var rDoc v3.Server
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareServers(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareServers_ExtensionAddedRemoved(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `url: https://pb33f.io`

	right := `url: https://pb33f.io
x-custom: value`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Server
	var rDoc v3.Server
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareServers(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.NotNil(t, extChanges.ExtensionChanges)
	assert.Len(t, extChanges.ExtensionChanges.GetAllChanges(), 1)
}
