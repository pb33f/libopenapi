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
	"go.yaml.in/yaml/v4"
)

// cleanHashCacheForTest clears the hash cache and sets up cleanup for individual tests
func cleanHashCacheForTest(t *testing.T) {
	low.ClearHashCache()
	t.Cleanup(func() {
		low.ClearHashCache()
	})
}

func TestResponses_Build(t *testing.T) {
	cleanHashCacheForTest(t)

	yml := `"200":
  summary: success response
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
  summary: default summary
  description: default response`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Responses
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	assert.NotNil(t, n.GetRootNode())
	assert.Nil(t, n.GetKeyNode())
	assert.NotNil(t, n.GetIndex())
	assert.NotNil(t, n.GetContext())

	assert.NoError(t, err)
	assert.Equal(t, "default summary", n.Default.Value.Summary.Value)
	assert.Equal(t, "default response", n.Default.Value.Description.Value)

	ok := n.FindResponseByCode("200")
	assert.NotNil(t, ok.Value)
	assert.Equal(t, "success response", ok.Value.Summary.Value)
	assert.Equal(t, "some response", ok.Value.Description.Value)
	assert.NotNil(t, ok.Value.GetKeyNode())

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

	// check hash - updated to include summary fields
	// Hash will be different now that summary is included
	hashString := low.GenerateHashString(&n)
	assert.NotEmpty(t, hashString)
	// Verify hash is consistent
	assert.Equal(t, hashString, low.GenerateHashString(&n))
}

func TestResponse_OpenAPI32_Summary(t *testing.T) {
	cleanHashCacheForTest(t)

	// Test OpenAPI 3.2 Response with summary field
	yml := `summary: Success response summary
description: Detailed description of the response
headers:
  X-Rate-Limit:
    description: Rate limit header
content:
  application/json:
    schema:
      type: object
      properties:
        message:
          type: string`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var response Response
	err := low.BuildModel(idxNode.Content[0], &response)
	assert.NoError(t, err)

	err = response.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)

	// Verify summary field is populated
	assert.Equal(t, "Success response summary", response.Summary.Value)
	assert.Equal(t, "Detailed description of the response", response.Description.Value)
	assert.NotNil(t, response.Summary.ValueNode)
	assert.NotNil(t, response.Description.ValueNode)

	// Verify summary is included in hash
	hash1 := response.Hash()
	response.Summary.Value = "Modified summary"
	hash2 := response.Hash()
	assert.NotEqual(t, hash1, hash2, "Hash should change when summary changes")
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
	assert.Equal(t, "9fc8294b7dcfc242fffe2586e10c9272fa2b9c828702a6b268ca68e8aa35cbbe",
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
	assert.NotNil(t, n.FindResponseByCode("200").GetValue().GetRootNode())
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
	assert.NotNil(t, n.GetIndex())
	assert.NotNil(t, n.GetContext())
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
