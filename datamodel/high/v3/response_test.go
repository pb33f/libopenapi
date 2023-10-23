// Copyright 2022-2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

// this test exists because the sample contract doesn't contain a
// response with *everything* populated, I had already written a ton of tests
// with hard coded line and column numbers in them, changing the spec above the bottom will
// create pointless test changes. So here is a standalone test. you know... for science.
func TestNewResponse(t *testing.T) {

	yml := `description: this is a response
headers:
  someHeader:
    description: a header
content:
  something/thing:
    description: a thing
x-pizza-man: pizza!
links:
  someLink:
    description: a link!        `

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())

	var n v3.Response
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	r := NewResponse(&n)

	assert.Len(t, r.Headers, 1)
	assert.Len(t, r.Content, 1)
	assert.Equal(t, "pizza!", r.Extensions["x-pizza-man"])
	assert.Len(t, r.Links, 1)
	assert.Equal(t, 1, r.GoLow().Description.KeyNode.Line)

}

func TestResponse_MarshalYAML(t *testing.T) {

	yml := `description: this is a response
headers:
    someHeader:
        description: a header
content:
    something/thing:
        example: cake
links:
    someLink:
        description: a link!`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())

	var n v3.Response
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	r := NewResponse(&n)

	rend, _ := r.Render()
	assert.Equal(t, yml, strings.TrimSpace(string(rend)))

}

func TestResponse_MarshalYAMLInline(t *testing.T) {

	yml := `description: this is a response
headers:
    someHeader:
        description: a header
content:
    something/thing:
        example: cake
links:
    someLink:
        description: a link!`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())

	var n v3.Response
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	r := NewResponse(&n)

	rend, _ := r.RenderInline()
	assert.Equal(t, yml, strings.TrimSpace(string(rend)))

}
