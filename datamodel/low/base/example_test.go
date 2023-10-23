// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestExample_Build_Success_NoValue(t *testing.T) {

	yml := `summary: hot
description: cakes
x-cake: hot`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	var n Example
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "hot", n.Summary.Value)
	assert.Equal(t, "cakes", n.Description.Value)
	assert.Nil(t, n.Value.Value)
	ext := n.FindExtension("x-cake")
	assert.NotNil(t, ext)
	assert.Equal(t, "hot", ext.Value)
}

func TestExample_Build_Success_Simple(t *testing.T) {

	yml := `summary: hot
description: cakes
value: a string example
x-cake: hot`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	var n Example
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "hot", n.Summary.Value)
	assert.Equal(t, "cakes", n.Description.Value)
	assert.Equal(t, "a string example", n.Value.Value)
	ext := n.FindExtension("x-cake")
	assert.NotNil(t, ext)
	assert.Equal(t, "hot", ext.Value)
}

func TestExample_Build_Success_Object(t *testing.T) {

	yml := `summary: hot
description: cakes
value:
  pizza: oven
  yummy: pizza`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	var n Example
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "hot", n.Summary.Value)
	assert.Equal(t, "cakes", n.Description.Value)

	if v, ok := n.Value.Value.(map[string]interface{}); ok {
		assert.Equal(t, "oven", v["pizza"])
		assert.Equal(t, "pizza", v["yummy"])
	} else {
		assert.Fail(t, "failed to decode correctly.")
	}

}

func TestExample_Build_Success_Array(t *testing.T) {

	yml := `summary: hot
description: cakes
value:
  - wow
  - such array`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	var n Example
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "hot", n.Summary.Value)
	assert.Equal(t, "cakes", n.Description.Value)

	if v, ok := n.Value.Value.([]interface{}); ok {
		assert.Equal(t, "wow", v[0])
		assert.Equal(t, "such array", v[1])
	} else {
		assert.Fail(t, "failed to decode correctly.")
	}
}

func TestExample_Build_Success_MergeNode(t *testing.T) {

	yml := `x-things: &things
  summary: hot
  description: cakes
  value:
    - wow
    - such array
<<: *things`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	var n Example
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "hot", n.Summary.Value)
	assert.Equal(t, "cakes", n.Description.Value)

	if v, ok := n.Value.Value.([]interface{}); ok {
		assert.Equal(t, "wow", v[0])
		assert.Equal(t, "such array", v[1])
	} else {
		assert.Fail(t, "failed to decode correctly.")
	}

}

func TestExample_ExtractExampleValue_Map(t *testing.T) {

	yml := `hot:
    summer: nights
    pizza: oven`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	val := ExtractExampleValue(idxNode.Content[0])
	if v, ok := val.(map[string]interface{}); ok {
		if r, rok := v["hot"].(map[string]interface{}); rok {
			assert.Equal(t, "nights", r["summer"])
			assert.Equal(t, "oven", r["pizza"])
		} else {
			assert.Fail(t, "failed to decode correctly.")
		}
	} else {
		assert.Fail(t, "failed to decode correctly.")
	}
}

func TestExample_ExtractExampleValue_Slice(t *testing.T) {

	yml := `- hot:
    summer: nights
- hotter:
    pizza: oven`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	val := ExtractExampleValue(idxNode.Content[0])
	if v, ok := val.([]interface{}); ok {
		for w := range v {
			if r, rok := v[w].(map[string]interface{}); rok {
				for k := range r {
					if k == "hotter" {
						assert.Equal(t, "oven", r[k].(map[string]interface{})["pizza"])
					}
					if k == "hot" {
						assert.Equal(t, "nights", r[k].(map[string]interface{})["summer"])
					}
				}
			} else {
				assert.Fail(t, "failed to decode correctly.")
			}
		}

	} else {
		assert.Fail(t, "failed to decode correctly.")
	}
}

func TestExample_Hash(t *testing.T) {

	left := `summary: hot
description: cakes
x-burger: nice
externalValue: cake
value:
  pizza: oven
  yummy: pizza`

	right := `externalValue: cake
summary: hot
value:
  pizza: oven
  yummy: pizza
description: cakes
x-burger: nice`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc Example
	var rDoc Example
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	assert.Equal(t, lDoc.Hash(), rDoc.Hash())
	assert.Len(t, lDoc.GetExtensions(), 1)
}

func TestExtractExampleValue(t *testing.T) {
	assert.True(t, ExtractExampleValue(&yaml.Node{Tag: "!!bool", Value: "true"}).(bool))
	assert.Equal(t, int64(10), ExtractExampleValue(&yaml.Node{Tag: "!!int", Value: "10"}).(int64))
	assert.Equal(t, 33.2, ExtractExampleValue(&yaml.Node{Tag: "!!float", Value: "33.2"}).(float64))
	assert.Equal(t, "WHAT A NICE COW", ExtractExampleValue(&yaml.Node{Tag: "!!str", Value: "WHAT A NICE COW"}))

}
