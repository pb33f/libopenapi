// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestResponses_Build_Response(t *testing.T) {

	yml := `- $ref: break`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Responses
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(nil, idxNode.Content[0], idx)
	assert.Error(t, err)

}

func TestResponses_Build_Response_Default(t *testing.T) {

	yml := `default:
  $ref: break`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Responses
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(nil, idxNode.Content[0], idx)
	assert.Error(t, err)

}

func TestResponses_Build_WrongType(t *testing.T) {

	yml := `- $ref: break`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Responses
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(nil, idxNode.Content[0], idx)
	assert.Error(t, err)

}

func TestResponses_Hash(t *testing.T) {

	yml := `default:
  description: I am a potato
200:
  description: OK
301:
  description: dont need it you're good
x-tea: warm
400:
  description: wat?
401:
  description: and you are?
404: 
  description: not here mate.`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Responses
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(nil, idxNode.Content[0], idx)

	yml2 := `401:
  description: and you are?
200:
  description: OK
default:
  description: I am a potato
400:
  description: wat?
301:
  description: dont need it you're good
404: 
  description: not here mate.
x-tea: warm`

	var idxNode2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml2), &idxNode2)
	idx2 := index.NewSpecIndex(&idxNode2)

	var n2 Responses
	_ = low.BuildModel(idxNode2.Content[0], &n2)
	_ = n2.Build(nil, idxNode2.Content[0], idx2)

	// hash
	assert.Equal(t, n.Hash(), n2.Hash())
	assert.Len(t, n.GetExtensions(), 1)

}
