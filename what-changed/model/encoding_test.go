// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"context"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/low"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/testify/assert"
)

func TestCompareEncoding(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `contentType: application/json
headers:
  aHeader:
    description: a header
style: date
explode: true
allowReserved: true`

	right := `contentType: application/json
headers:
  aHeader:
    description: a header
style: date
explode: true
allowReserved: true`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Encoding
	var rDoc v3.Encoding
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareEncoding(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareEncoding_Modified(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `contentType: application/xml
headers:
  aHeader:
    description: a header description
style: date
explode: false
allowReserved: false`

	right := `contentType: application/json
headers:
  aHeader:
    description: a header
style: date
explode: true
allowReserved: true`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Encoding
	var rDoc v3.Encoding
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareEncoding(&lDoc, &rDoc)
	assert.NotNil(t, extChanges)
	assert.Equal(t, 4, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 4)
	assert.Equal(t, 2, extChanges.TotalBreakingChanges())
}

func TestCompareEncoding_Added(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `contentType: application/json
explode: true
allowReserved: true`

	right := `contentType: application/json
headers:
  aHeader:
    description: a header
style: date
explode: true
allowReserved: true`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Encoding
	var rDoc v3.Encoding
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareEncoding(&lDoc, &rDoc)
	assert.NotNil(t, extChanges)
	assert.Equal(t, 2, extChanges.TotalChanges()) // style added + header added
	assert.Len(t, extChanges.GetAllChanges(), 2)
	assert.Equal(t, 1, extChanges.TotalBreakingChanges()) // style added is breaking
}

func TestCompareEncoding_Removed(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	left := `contentType: application/json
explode: true
allowReserved: true`

	right := `contentType: application/json
headers:
  aHeader:
    description: a header
style: date
explode: true
allowReserved: true`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Encoding
	var rDoc v3.Encoding
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareEncoding(&rDoc, &lDoc)
	assert.NotNil(t, extChanges)
	assert.Equal(t, 2, extChanges.TotalChanges()) // style removed + header removed
	assert.Len(t, extChanges.GetAllChanges(), 2)
	assert.Equal(t, 2, extChanges.TotalBreakingChanges()) // both style and header removed are breaking
}

// Removing an extension is breaking, so it is counted in TotalBreakingChanges as well as TotalChanges.
func TestCompareEncoding_ExtensionRemovedIsBreaking(t *testing.T) {
	low.ClearHashCache()
	ResetActiveBreakingRulesConfig()
	build := func(yml string) *v3.Encoding {
		var node yaml.Node
		_ = yaml.Unmarshal([]byte(yml), &node)
		var enc v3.Encoding
		_ = low.BuildModel(node.Content[0], &enc)
		_ = enc.Build(context.Background(), nil, node.Content[0], nil)
		return &enc
	}
	withExt := build("contentType: image/png\nx-vendor: abc")
	withoutExt := build("contentType: image/png")

	removed := CompareEncoding(withExt, withoutExt)
	if assert.NotNil(t, removed) {
		assert.Equal(t, 1, removed.TotalChanges())
		assert.Len(t, removed.GetAllChanges(), 1)
		assert.Equal(t, 1, removed.TotalBreakingChanges())
	}

	added := CompareEncoding(withoutExt, withExt)
	if assert.NotNil(t, added) {
		assert.Equal(t, 1, added.TotalChanges())
		assert.Equal(t, 0, added.TotalBreakingChanges())
	}
}
