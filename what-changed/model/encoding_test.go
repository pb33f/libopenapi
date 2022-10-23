// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestCompareEncoding(t *testing.T) {

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
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareEncoding(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareEncoding_Modified(t *testing.T) {

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
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareEncoding(&lDoc, &rDoc)
	assert.NotNil(t, extChanges)
	assert.Equal(t, 4, extChanges.TotalChanges())
	assert.Equal(t, 2, extChanges.TotalBreakingChanges())

}

func TestCompareEncoding_Added(t *testing.T) {

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
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareEncoding(&lDoc, &rDoc)
	assert.NotNil(t, extChanges)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
	assert.Equal(t, ObjectAdded, extChanges.Changes[0].ChangeType)
	assert.Equal(t, v3.HeadersLabel, extChanges.Changes[0].Property)
}

func TestCompareEncoding_Removed(t *testing.T) {

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
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareEncoding(&rDoc, &lDoc)
	assert.NotNil(t, extChanges)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())

}
