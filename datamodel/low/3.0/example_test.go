package v3

import (
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
	idx := index.NewSpecIndex(&idxNode)

	var n Example
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
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
	idx := index.NewSpecIndex(&idxNode)

	var n Example
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
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
	idx := index.NewSpecIndex(&idxNode)

	var n Example
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
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
	idx := index.NewSpecIndex(&idxNode)

	var n Example
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
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
