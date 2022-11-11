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

func TestResponses_Build(t *testing.T) {

	yml := `"200":
  description: some response
  headers:
    header1: 
      description: some header
  content: 
    nice/rice:
      schema:
        type: string
        description: this is some content.
  links:
    someLink:
      description: a link
  x-gut: rot    
default:
  description: default response`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Responses
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "default response", n.Default.Value.Description.Value)

	ok := n.FindResponseByCode("200")
	assert.NotNil(t, ok.Value)
	assert.Equal(t, "some response", ok.Value.Description.Value)
	assert.Equal(t, "rot", ok.Value.FindExtension("x-gut").Value)

	con := ok.Value.FindContent("nice/rice")
	assert.NotNil(t, con.Value)
	assert.Equal(t, "this is some content.", con.Value.Schema.Value.Schema().Description.Value)

	head := ok.Value.FindHeader("header1")
	assert.NotNil(t, head.Value)
	assert.Equal(t, "some header", head.Value.Description.Value)

	link := ok.Value.FindLink("someLink")
	assert.NotNil(t, link.Value)
	assert.Equal(t, "a link", link.Value.Description.Value)

}

func TestResponses_Build_FailCodes_WrongType(t *testing.T) {

	yml := `- "200":
  $ref: #bork`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Responses
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.Error(t, err)

}

func TestResponses_Build_FailCodes(t *testing.T) {

	yml := `"200":
  $ref: #bork`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Responses
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.Error(t, err)

}

func TestResponses_Build_FailDefault(t *testing.T) {

	yml := `- default`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Responses
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.Error(t, err)

}

func TestResponses_Build_FailBadHeader(t *testing.T) {

	yml := `"200":
  headers:
    header1: 
      $ref: borko`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Responses
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.Error(t, err)

}

func TestResponses_Build_FailBadContent(t *testing.T) {

	yml := `"200":
  content:
    flim/flam: 
      $ref: borko`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Responses
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.Error(t, err)

}

func TestResponses_Build_FailBadLinks(t *testing.T) {

	yml := `"200":
  links:
    aLink: 
      $ref: borko`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Responses
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.Error(t, err)

}

func TestResponse_Hash(t *testing.T) {

	yml := `description: nice toast
headers:
  heady:
    description: a header
  handy:
    description: a handy
content:
  nice/toast:
    schema: 
      type: int
  nice/roast:
    schema: 
      type: int
x-jam: toast
x-ham: jam
links: 
  linky:
    operationId: one two toast`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Response
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(idxNode.Content[0], idx)

	yml2 := `description: nice toast
x-ham: jam
headers:
  heady:
    description: a header
  handy:
    description: a handy
content:
  nice/toast:
    schema: 
      type: int
  nice/roast:
    schema: 
      type: int
x-jam: toast
links: 
  linky:
    operationId: one two toast`

	var idxNode2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml2), &idxNode2)
	idx2 := index.NewSpecIndex(&idxNode2)

	var n2 Response
	_ = low.BuildModel(idxNode2.Content[0], &n2)
	_ = n2.Build(idxNode2.Content[0], idx2)

	// hash
	assert.Equal(t, n.Hash(), n2.Hash())

}
