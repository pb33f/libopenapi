// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
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
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(nil, idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "michelle, meddy and maddy", n.Description.Value)
	assert.True(t, n.AllowReserved.Value)
	assert.True(t, n.Explode.Value)
	assert.True(t, n.Required.Value)
	assert.False(t, n.Deprecated.Value)
	assert.Equal(t, "happy", n.Name.Value)
	assert.Equal(t, "path", n.In.Value)
	assert.NotNil(t, n.Schema.Value)
	assert.Equal(t, "my triple M, my loves", n.Schema.Value.Schema().Description.Value)
	assert.NotNil(t, n.Schema.Value.Schema().Properties.Value)
	assert.Equal(t, "she is my heart.", n.Schema.Value.Schema().FindProperty("michelle").Value.Schema().Description.Value)
	assert.Equal(t, "she is my song.", n.Schema.Value.Schema().FindProperty("meddy").Value.Schema().Description.Value)
	assert.Equal(t, "he is my champion.", n.Schema.Value.Schema().FindProperty("maddy").Value.Schema().Description.Value)

	if v, ok := n.Example.Value.(map[string]interface{}); ok {
		assert.Equal(t, "my love.", v["michelle"])
		assert.Equal(t, "my song.", v["meddy"])
		assert.Equal(t, "my champion.", v["maddy"])
	} else {
		assert.Fail(t, "should not fail")
	}

	con := n.FindContent("family/love").Value
	assert.NotNil(t, con)
	assert.Equal(t, "family love.", con.Schema.Value.Schema().Description.Value)
	assert.Nil(t, n.FindContent("unknown"))

	ext := n.FindExtension("x-family-love").Value
	assert.Equal(t, "strong", ext)
	assert.Len(t, n.GetExtensions(), 1)
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

	err = n.Build(nil, idxNode.Content[0], idx)
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

	err = n.Build(nil, idxNode.Content[0], idx)
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

	err = n.Build(nil, idxNode.Content[0], idx)
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

	err = n.Build(nil, idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestParameter_Hash_n_grab(t *testing.T) {

	yml := `description: michelle, meddy and maddy
required: true
deprecated: false
name: happy
in: path
allowEmptyValue: false
style: beautiful
explode: true
allowReserved: true
examples:
  beautiful:
    description: baby girl
  handsome:
    description: baby boy
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
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Parameter
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(nil, idxNode.Content[0], idx)

	yml2 := `description: michelle, meddy and maddy
required: true
deprecated: false
name: happy
in: path
examples:
  beautiful:
    description: baby girl
  handsome:
    description: baby boy
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

	var idxNode2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml2), &idxNode2)
	idx2 := index.NewSpecIndex(&idxNode2)

	var n2 Parameter
	_ = low.BuildModel(idxNode2.Content[0], &n2)
	_ = n2.Build(nil, idxNode2.Content[0], idx2)

	// hash
	assert.Equal(t, n.Hash(), n2.Hash())

	// n grab
	assert.Equal(t, "happy", n.GetName().Value)
	assert.Equal(t, "path", n.GetIn().Value)
	assert.Equal(t, "michelle, meddy and maddy", n.GetDescription().Value)
	assert.True(t, n.GetRequired().Value)
	assert.False(t, n.GetDeprecated().Value)
	assert.False(t, n.GetAllowEmptyValue().Value)
	assert.Len(t, n.GetSchema().Value.(*base.SchemaProxy).Schema().Properties.Value, 3)
	assert.Equal(t, "beautiful", n.GetStyle().Value)
	assert.True(t, n.GetAllowReserved().Value)
	assert.True(t, n.GetExplode().Value)
	assert.NotNil(t, n.GetExample().Value)
	assert.Len(t, n.GetExamples().Value.(map[low.KeyReference[string]]low.ValueReference[*base.Example]), 2)
	assert.Len(t, n.GetContent().Value.(map[low.KeyReference[string]]low.ValueReference[*MediaType]), 1)
}
