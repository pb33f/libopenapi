// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestParameter_Build(t *testing.T) {
	yml := `description: michelle, meddy and maddy
required: true
deprecated: false
name: happy
in: path
allowEmptyValue: false
style: beautiful
explode: true
allowReserved: true
schema:
  type: object
  description: my triple M, my loves
  properties:
    michelle:
      type: string
      description: she is my heart.
    meddy:
      type: string
      description: she is my song.
    maddy:
      type: string
      description: he is my champion.    
x-family-love: strong
example:
  michelle: my love.
  maddy: my champion.
  meddy: my song.
content:
  family/love:
    schema: 
      type: string
      description: family love.`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Parameter
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "michelle, meddy and maddy", n.Description.Value)
	assert.True(t, n.AllowReserved.Value)
	assert.True(t, n.Explode.Value)
	assert.True(t, n.Required.Value)
	assert.False(t, n.Deprecated.Value)
	assert.Equal(t, "happy", n.Name.Value)
	assert.Equal(t, "path", n.In.Value)
	assert.NotNil(t, n.Schema.Value)
	assert.Equal(t, "my triple M, my loves", n.Schema.Value.Description.Value)
	assert.NotNil(t, n.Schema.Value.Properties.Value)
	assert.Equal(t, "she is my heart.", n.Schema.Value.FindProperty("michelle").Value.Description.Value)
	assert.Equal(t, "she is my song.", n.Schema.Value.FindProperty("meddy").Value.Description.Value)
	assert.Equal(t, "he is my champion.", n.Schema.Value.FindProperty("maddy").Value.Description.Value)

	if v, ok := n.Example.Value.(map[string]interface{}); ok {
		assert.Equal(t, "my love.", v["michelle"])
		assert.Equal(t, "my song.", v["meddy"])
		assert.Equal(t, "my champion.", v["maddy"])
	} else {
		assert.Fail(t, "should not fail")
	}

	con := n.FindContent("family/love").Value
	assert.NotNil(t, con)
	assert.Equal(t, "family love.", con.Schema.Value.Description.Value)
	assert.Nil(t, n.FindContent("unknown"))

	ext := n.FindExtension("x-family-love").Value
	assert.Equal(t, "strong", ext)
}

func TestParameter_Build_Success_Examples(t *testing.T) {
	yml := `examples:
  family:
    value:
      michelle: my love.
      maddy: my champion.
      meddy: my song.`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Parameter
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.NoError(t, err)

	exp := n.FindExample("family").Value
	assert.NotNil(t, exp)

	if v, ok := exp.Value.Value.(map[string]interface{}); ok {
		assert.Equal(t, "my love.", v["michelle"])
		assert.Equal(t, "my song.", v["meddy"])
		assert.Equal(t, "my champion.", v["maddy"])
	} else {
		assert.Fail(t, "should not fail")
	}
}

func TestParameter_Build_Fail_Examples(t *testing.T) {
	yml := `examples:
  family:
    $ref: I AM BORKED`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Parameter
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestParameter_Build_Fail_Schema(t *testing.T) {
	yml := `schema:
  $ref: I will fail.`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Parameter
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestParameter_Build_Fail_Content(t *testing.T) {
	yml := `content:
  ohMyStars:
    $ref: fail!`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Parameter
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.Error(t, err)
}
