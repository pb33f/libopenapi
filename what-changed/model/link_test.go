// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestCompareLinks(t *testing.T) {

	left := `operationId: someOperation
requestBody: expression-says-what
description: a nice link
server:
  url: https://pb33f.io
parameters:
  nice: rice`

	right := left

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Link
	var rDoc v3.Link
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareLinks(&lDoc, &rDoc)
	assert.Nil(t, extChanges)

}

func TestCompareLinks_ModifyExtension(t *testing.T) {

	left := `operationId: someOperation
requestBody: expression-says-what
description: a nice link
server:
  url: https://pb33f.io
parameters:
  nice: rice
x-cake: tasty`

	right := `operationId: someOperation
requestBody: expression-says-what
description: a nice link
server:
  url: https://pb33f.io
parameters:
  nice: rice
x-cake: very tasty`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Link
	var rDoc v3.Link
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareLinks(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, Modified, extChanges.ExtensionChanges.Changes[0].ChangeType)

}

func TestCompareLinks_ModifyServer(t *testing.T) {

	left := `operationId: someOperation
requestBody: expression-says-what
description: a nice link
server:
  url: https://pb33f.io
parameters:
  nice: rice`

	right := `operationId: someOperation
requestBody: expression-says-what
description: a nice link
server:
  url: https://pb33f.io/changed
parameters:
  nice: rice`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Link
	var rDoc v3.Link
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareLinks(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, Modified, extChanges.ServerChanges.Changes[0].ChangeType)
}

func TestCompareLinks_AddServer(t *testing.T) {

	left := `operationId: someOperation
requestBody: expression-says-what
description: a nice link
parameters:
  nice: rice`

	right := `operationId: someOperation
requestBody: expression-says-what
description: a nice link
server:
  url: https://pb33f.io/changed
parameters:
  nice: rice`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Link
	var rDoc v3.Link
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareLinks(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, PropertyAdded, extChanges.Changes[0].ChangeType)
}

func TestCompareLinks_RemoveServer(t *testing.T) {

	left := `operationId: someOperation
requestBody: expression-says-what
description: a nice link
parameters:
  nice: rice`

	right := `operationId: someOperation
requestBody: expression-says-what
description: a nice link
server:
  url: https://pb33f.io/changed
parameters:
  nice: rice`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Link
	var rDoc v3.Link
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareLinks(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, PropertyRemoved, extChanges.Changes[0].ChangeType)
}

func TestCompareLinks_ModifyParam(t *testing.T) {

	left := `operationId: someOperation
requestBody: expression-says-what
description: a nice link
server:
  url: https://pb33f.io
parameters:
  nice: cake`

	right := `operationId: someOperation
requestBody: expression-says-what
description: a nice link
server:
  url: https://pb33f.io
parameters:
  nice: rice`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Link
	var rDoc v3.Link
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareLinks(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, Modified, extChanges.Changes[0].ChangeType)
	assert.Equal(t, "nice", extChanges.Changes[0].NewObject)
	assert.Equal(t, "cake", extChanges.Changes[0].Original)
	assert.Equal(t, "rice", extChanges.Changes[0].New)
}

func TestCompareLinks_AddParam(t *testing.T) {

	left := `operationId: someOperation
requestBody: expression-says-what
description: a nice link
server:
  url: https://pb33f.io
parameters:
  nice: cake`

	right := `operationId: someOperation
requestBody: expression-says-what
description: a nice link
server:
  url: https://pb33f.io
parameters:
  nice: cake
  hot: pizza`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Link
	var rDoc v3.Link
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareLinks(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, extChanges.Changes[0].ChangeType)
	assert.Equal(t, "hot", extChanges.Changes[0].NewObject)
	assert.Equal(t, "pizza", extChanges.Changes[0].New)
}

func TestCompareLinks_RemoveParam(t *testing.T) {

	left := `operationId: someOperation
requestBody: expression-says-what
description: a nice link
server:
  url: https://pb33f.io
parameters:
  nice: cake`

	right := `operationId: someOperation
requestBody: expression-says-what
description: a nice link
server:
  url: https://pb33f.io
parameters:
  nice: cake
  hot: pizza`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Link
	var rDoc v3.Link
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareLinks(&rDoc, &lDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, "hot", extChanges.Changes[0].OriginalObject)
	assert.Equal(t, "pizza", extChanges.Changes[0].Original)
}
