// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/v2"
	"github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/what-changed/core"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func test_buildUltraGlobOfHeaders() string {

	// this is a mega glob of every header item in both swagger and openapi. the versioned models will
	// pick out the relevant bits required when parsing.
	return `description: header desc
type: string
format: something
items:
  type: string
collectionFormat: something
default: hello
maximum: 20
minimum: 10
exclusiveMinimum: true
exclusiveMaximum: true
maxLength: 200
minLength: 100
pattern: coffee
maxItems: 20
minItems: 20
uniqueItems: true
enum:
  - one
multipleOf: 5
required: true
deprecated: true
allowEmptyValue: true
style: true
explode: true
allowReserved: true
schema:
  description: a schema
  type: string
example: a thing
examples:
  something:
    description: some example
    value: nice example
content:
  application/json:
    schema:
      description: jayson says hi!
      type: string
x-beer: yummy`
}

func TestCompareHeaders_v2_identical(t *testing.T) {

	left := test_buildUltraGlobOfHeaders()

	right := left

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Header
	var rDoc v2.Header
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareHeadersV2(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareHeaders_v2_modified(t *testing.T) {

	left := test_buildUltraGlobOfHeaders()

	right := `description: header desc
type: string
format: something
items:
  type: int
collectionFormat: something
default: hello
maximum: 20
minimum: 10
exclusiveMinimum: true
exclusiveMaximum: true
maxLength: 200
minLength: 100
pattern: coffee
maxItems: 20
minItems: 20
uniqueItems: true
enum:
  - one
multipleOf: 5
x-beer: really yummy`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Header
	var rDoc v2.Header
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareHeadersV2(&lDoc, &rDoc)
	assert.NotNil(t, extChanges)
	assert.Equal(t, 2, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
}

func TestCompareHeaders_v2_addedItems(t *testing.T) {

	left := test_buildUltraGlobOfHeaders()

	right := `description: header desc
type: string
format: something
collectionFormat: something
default: hello
maximum: 20
minimum: 10
exclusiveMinimum: true
exclusiveMaximum: true
maxLength: 200
minLength: 100
pattern: coffee
maxItems: 20
minItems: 20
uniqueItems: true
enum:
  - one
multipleOf: 5
x-beer: yummy`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Header
	var rDoc v2.Header
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareHeadersV2(&rDoc, &lDoc)
	assert.NotNil(t, extChanges)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, core.ObjectAdded, extChanges.Changes[0].ChangeType)
}

func TestCompareHeaders_v2_removedItems(t *testing.T) {

	left := test_buildUltraGlobOfHeaders()

	right := `description: header desc
type: string
format: something
collectionFormat: something
default: hello
maximum: 20
minimum: 10
exclusiveMinimum: true
exclusiveMaximum: true
maxLength: 200
minLength: 100
pattern: coffee
maxItems: 20
minItems: 20
uniqueItems: true
enum:
  - one
multipleOf: 5
x-beer: yummy`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Header
	var rDoc v2.Header
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareHeadersV2(&lDoc, &rDoc)
	assert.NotNil(t, extChanges)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, 1, extChanges.TotalBreakingChanges())
	assert.Equal(t, core.ObjectRemoved, extChanges.Changes[0].ChangeType)
}

func TestCompareHeaders_v2_ItemsModified(t *testing.T) {

	left := test_buildUltraGlobOfHeaders()

	right := left

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v2.Header
	var rDoc v2.Header
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareHeadersV2(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareHeaders_v3_identical(t *testing.T) {

	left := test_buildUltraGlobOfHeaders()

	right := left

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Header
	var rDoc v3.Header
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareHeadersV3(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareHeaders_v3_modified(t *testing.T) {

	left := test_buildUltraGlobOfHeaders()

	right := `required: true
deprecated: false
allowEmptyValue: true
style: true
explode: true
allowReserved: true
schema:
  description: a schema description
  type: string
example: a thing
examples:
  something:
    description: some example description
    value: nice example
content:
  application/json:
    schema:
      description: jayson says hi again!
      type: string
x-beer: yummy`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc v3.Header
	var rDoc v3.Header
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareHeadersV3(&lDoc, &rDoc)
	assert.NotNil(t, extChanges)
	assert.Equal(t, 5, extChanges.TotalChanges())
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())

}
