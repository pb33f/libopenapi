// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
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
x-shoes: old
default:
  description: default response`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Responses
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "default response", n.Default.Value.Description.Value)

	ok := n.FindResponseByCode("200")
	assert.NotNil(t, ok.Value)
	assert.Equal(t, "some response", ok.Value.Description.Value)

	var xGut string
	_ = ok.Value.FindExtension("x-gut").Value.Decode(&xGut)
	assert.Equal(t, "rot", xGut)

	con := ok.Value.FindContent("nice/rice")
	assert.NotNil(t, con.Value)
	assert.Equal(t, "this is some content.", con.Value.Schema.Value.Schema().Description.Value)

	head := ok.Value.FindHeader("header1")
	assert.NotNil(t, head.Value)
	assert.Equal(t, "some header", head.Value.Description.Value)

	link := ok.Value.FindLink("someLink")
	assert.NotNil(t, link.Value)
	assert.Equal(t, "a link", link.Value.Description.Value)

	// check hash
	assert.Equal(t, "37ae6a91f2260031e22bd6fbf2d286928dd910b14cb75d4239fb80651ac5ecff",
		low.GenerateHashString(&n))
}

func TestResponses_NoDefault(t *testing.T) {
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
x-shoes: old`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Responses
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)

	// check hash
	assert.Equal(t, "3da5051dcd82a06f8e4c7698cdec03550ae1988ee54d96d4c4a90a5c8f9d7b2b",
		low.GenerateHashString(&n))

	assert.Equal(t, 1, orderedmap.Len(n.FindResponseByCode("200").Value.GetExtensions()))
	assert.Equal(t, 1, orderedmap.Len(n.GetExtensions()))
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

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
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

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
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

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
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

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
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

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
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

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestResponses_Build_AllowXPrefixHeader(t *testing.T) {
	yml := `"200":
  headers:
    x-header1:
      schema:
        type: string`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Responses
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)

	assert.Equal(t, "string",
		n.FindResponseByCode("200").Value.FindHeader("x-header1").Value.Schema.Value.Schema().Type.Value.A)

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
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

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
	_ = n2.Build(context.Background(), nil, idxNode2.Content[0], idx2)

	// hash
	assert.Equal(t, n.Hash(), n2.Hash())
}

//
//func TestResponses_Default(t *testing.T) {
//
//	yml := `"200":
//  description: some response
//  headers:
//    header1:
//      description: some header
//  content:
//    nice/rice:
//      schema:
//        type: string
//        description: this is some content.
//  links:
//    someLink:
//      description: a link
//  x-gut: rot
//default:
//  description: default response`
//
//	var idxNode yaml.Node
//	_ = yaml.Unmarshal([]byte(yml), &idxNode)
//	idx := index.NewSpecIndex(&idxNode)
//
//	var n Responses
//	err := low.BuildModel(&idxNode, &n)
//	assert.NoError(t, err)
//
//	err = n.Build(idxNode.Content[0], idx)
//	assert.NoError(t, err)
//	assert.Equal(t, "default response", n.Default.Value.Description.Value)
//
//	ok := n.FindResponseByCode("200")
//	assert.NotNil(t, ok.Value)
//	assert.Equal(t, "some response", ok.Value.Description.Value)
//	assert.Equal(t, "rot", ok.Value.FindExtension("x-gut").Value)
//
//	con := ok.Value.FindContent("nice/rice")
//	assert.NotNil(t, con.Value)
//	assert.Equal(t, "this is some content.", con.Value.Schema.Value.Schema().Description.Value)
//
//	head := ok.Value.FindHeader("header1")
//	assert.NotNil(t, head.Value)
//	assert.Equal(t, "some header", head.Value.Description.Value)
//
//	link := ok.Value.FindLink("someLink")
//	assert.NotNil(t, link.Value)
//	assert.Equal(t, "a link", link.Value.Description.Value)
//
//}
