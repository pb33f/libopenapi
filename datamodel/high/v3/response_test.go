// Copyright 2022-2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/datamodel/low"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

// this test exists because the sample contract doesn't contain a
// response with *everything* populated, I had already written a ton of tests
// with hard coded line and column numbers in them, changing the spec above the bottom will
// create pointless test changes. So here is a standalone test. you know... for science.
func TestNewResponse(t *testing.T) {
	yml := `summary: quick summary
description: this is a response
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

	assert.Equal(t, "quick summary", r.Summary)
	assert.Equal(t, "this is a response", r.Description)
	assert.Equal(t, 1, orderedmap.Len(r.Headers))
	assert.Equal(t, 1, orderedmap.Len(r.Content))

	var xPizzaMan string
	_ = r.Extensions.GetOrZero("x-pizza-man").Decode(&xPizzaMan)

	assert.Equal(t, "pizza!", xPizzaMan)
	assert.Equal(t, 1, orderedmap.Len(r.Links))
	assert.Equal(t, 2, r.GoLow().Description.KeyNode.Line)
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

func TestCreateResponseRef(t *testing.T) {
	ref := "#/components/responses/NotFound"
	r := CreateResponseRef(ref)

	assert.True(t, r.IsReference())
	assert.Equal(t, ref, r.GetReference())
	assert.Nil(t, r.GoLow())
}

func TestResponse_MarshalYAML_Reference(t *testing.T) {
	r := CreateResponseRef("#/components/responses/NotFound")

	node, err := r.MarshalYAML()
	assert.NoError(t, err)

	yamlNode, ok := node.(*yaml.Node)
	assert.True(t, ok)
	assert.Equal(t, yaml.MappingNode, yamlNode.Kind)
	assert.Equal(t, 2, len(yamlNode.Content))
	assert.Equal(t, "$ref", yamlNode.Content[0].Value)
	assert.Equal(t, "#/components/responses/NotFound", yamlNode.Content[1].Value)
}

func TestResponse_MarshalYAMLInline_Reference(t *testing.T) {
	r := CreateResponseRef("#/components/responses/NotFound")

	node, err := r.MarshalYAMLInline()
	assert.NoError(t, err)

	yamlNode, ok := node.(*yaml.Node)
	assert.True(t, ok)
	assert.Equal(t, "$ref", yamlNode.Content[0].Value)
}

func TestResponse_Reference_TakesPrecedence(t *testing.T) {
	// When both Reference and content are set, Reference should take precedence
	r := &Response{
		Reference:   "#/components/responses/foo",
		Description: "shouldBeIgnored",
	}

	assert.True(t, r.IsReference())

	node, err := r.MarshalYAML()
	assert.NoError(t, err)

	// Should render as $ref only, not full response
	rendered, _ := yaml.Marshal(node)
	assert.Contains(t, string(rendered), "$ref")
	assert.NotContains(t, string(rendered), "shouldBeIgnored")
}

func TestResponse_Render_Reference(t *testing.T) {
	r := CreateResponseRef("#/components/responses/NotFound")

	rendered, err := r.Render()
	assert.NoError(t, err)

	assert.Contains(t, string(rendered), "$ref")
	assert.Contains(t, string(rendered), "#/components/responses/NotFound")
}

func TestResponse_IsReference_False(t *testing.T) {
	r := &Response{
		Description: "A response",
	}
	assert.False(t, r.IsReference())
	assert.Equal(t, "", r.GetReference())
}

func TestResponse_MarshalYAMLInlineWithContext(t *testing.T) {
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

	ctx := base.NewInlineRenderContext()
	node, err := r.MarshalYAMLInlineWithContext(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, node)

	rendered, _ := yaml.Marshal(node)
	assert.Equal(t, yml, strings.TrimSpace(string(rendered)))
}

func TestResponse_MarshalYAMLInlineWithContext_Reference(t *testing.T) {
	r := CreateResponseRef("#/components/responses/NotFound")

	ctx := base.NewInlineRenderContext()
	node, err := r.MarshalYAMLInlineWithContext(ctx)
	assert.NoError(t, err)

	yamlNode, ok := node.(*yaml.Node)
	assert.True(t, ok)
	assert.Equal(t, "$ref", yamlNode.Content[0].Value)
}

func TestBuildLowResponse_Success(t *testing.T) {
	yml := `description: A successful response
content:
  application/json:
    schema:
      type: object`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	assert.NoError(t, err)

	result, err := buildLowResponse(node.Content[0], nil)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "A successful response", result.Description.Value)
}

func TestBuildLowResponse_BuildError(t *testing.T) {
	yml := `description: test
content:
  application/json:
    schema:
      $ref: '#/components/schemas/DoesNotExist'`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	assert.NoError(t, err)

	config := index.CreateOpenAPIIndexConfig()
	idx := index.NewSpecIndexWithConfig(&node, config)

	result, err := buildLowResponse(node.Content[0], idx)

	assert.Error(t, err)
	assert.Nil(t, result)
}
