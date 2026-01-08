// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestHeader_MarshalYAML(t *testing.T) {
	ext := orderedmap.New[string, *yaml.Node]()
	ext.Set("x-burgers", utils.CreateStringNode("why not?"))

	header := &Header{
		Description:     "A header",
		Required:        true,
		Deprecated:      true,
		AllowEmptyValue: true,
		Style:           "simple",
		Explode:         true,
		AllowReserved:   true,
		Example:         utils.CreateStringNode("example"),
		Examples: orderedmap.ToOrderedMap(map[string]*base.Example{
			"example": {Value: utils.CreateStringNode("example")},
		}),
		Extensions: ext,
	}

	rend, _ := header.Render()

	desired := `description: A header
required: true
deprecated: true
allowEmptyValue: true
style: simple
explode: true
allowReserved: true
example: example
examples:
    example:
        value: example
x-burgers: why not?`

	assert.Equal(t, desired, strings.TrimSpace(string(rend)))
}

func TestCreateHeaderRef(t *testing.T) {
	ref := "#/components/headers/X-Rate-Limit"
	h := CreateHeaderRef(ref)

	assert.True(t, h.IsReference())
	assert.Equal(t, ref, h.GetReference())
	assert.Nil(t, h.GoLow())
}

func TestHeader_MarshalYAML_Reference(t *testing.T) {
	h := CreateHeaderRef("#/components/headers/X-Rate-Limit")

	node, err := h.MarshalYAML()
	assert.NoError(t, err)

	yamlNode, ok := node.(*yaml.Node)
	assert.True(t, ok)
	assert.Equal(t, yaml.MappingNode, yamlNode.Kind)
	assert.Equal(t, 2, len(yamlNode.Content))
	assert.Equal(t, "$ref", yamlNode.Content[0].Value)
	assert.Equal(t, "#/components/headers/X-Rate-Limit", yamlNode.Content[1].Value)
}

func TestHeader_MarshalYAMLInline_Reference(t *testing.T) {
	h := CreateHeaderRef("#/components/headers/X-Rate-Limit")

	node, err := h.MarshalYAMLInline()
	assert.NoError(t, err)

	yamlNode, ok := node.(*yaml.Node)
	assert.True(t, ok)
	assert.Equal(t, "$ref", yamlNode.Content[0].Value)
}

func TestHeader_Reference_TakesPrecedence(t *testing.T) {
	// When both Reference and content are set, Reference should take precedence
	h := &Header{
		Reference:   "#/components/headers/foo",
		Description: "shouldBeIgnored",
	}

	assert.True(t, h.IsReference())

	node, err := h.MarshalYAML()
	assert.NoError(t, err)

	// Should render as $ref only, not full header
	rendered, _ := yaml.Marshal(node)
	assert.Contains(t, string(rendered), "$ref")
	assert.NotContains(t, string(rendered), "shouldBeIgnored")
}

func TestHeader_Render_Reference(t *testing.T) {
	h := CreateHeaderRef("#/components/headers/X-Rate-Limit")

	rendered, err := h.Render()
	assert.NoError(t, err)

	assert.Contains(t, string(rendered), "$ref")
	assert.Contains(t, string(rendered), "#/components/headers/X-Rate-Limit")
}

func TestHeader_IsReference_False(t *testing.T) {
	h := &Header{
		Description: "A header",
	}
	assert.False(t, h.IsReference())
	assert.Equal(t, "", h.GetReference())
}

func TestHeader_RenderInline_Reference(t *testing.T) {
	h := CreateHeaderRef("#/components/headers/X-Rate-Limit")

	rendered, err := h.RenderInline()
	assert.NoError(t, err)

	assert.Contains(t, string(rendered), "$ref")
	assert.Contains(t, string(rendered), "#/components/headers/X-Rate-Limit")
}

func TestHeader_RenderInline_NonReference(t *testing.T) {
	h := &Header{
		Description: "A rate limit header",
		Required:    true,
	}

	rendered, err := h.RenderInline()
	assert.NoError(t, err)

	assert.Contains(t, string(rendered), "description:")
	assert.Contains(t, string(rendered), "A rate limit header")
	assert.Contains(t, string(rendered), "required:")
}

func TestHeader_MarshalYAMLInlineWithContext(t *testing.T) {
	ext := orderedmap.New[string, *yaml.Node]()
	ext.Set("x-burgers", utils.CreateStringNode("why not?"))

	header := &Header{
		Description:     "A header",
		Required:        true,
		Deprecated:      true,
		AllowEmptyValue: true,
		Style:           "simple",
		Explode:         true,
		AllowReserved:   true,
		Example:         utils.CreateStringNode("example"),
		Examples: orderedmap.ToOrderedMap(map[string]*base.Example{
			"example": {Value: utils.CreateStringNode("example")},
		}),
		Extensions: ext,
	}

	ctx := base.NewInlineRenderContext()
	node, err := header.MarshalYAMLInlineWithContext(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, node)

	rend, _ := yaml.Marshal(node)

	desired := `description: A header
required: true
deprecated: true
allowEmptyValue: true
style: simple
explode: true
allowReserved: true
example: example
examples:
    example:
        value: example
x-burgers: why not?`

	assert.Equal(t, desired, strings.TrimSpace(string(rend)))
}

func TestHeader_MarshalYAMLInlineWithContext_Reference(t *testing.T) {
	h := CreateHeaderRef("#/components/headers/X-Rate-Limit")

	ctx := base.NewInlineRenderContext()
	node, err := h.MarshalYAMLInlineWithContext(ctx)
	assert.NoError(t, err)

	yamlNode, ok := node.(*yaml.Node)
	assert.True(t, ok)
	assert.Equal(t, "$ref", yamlNode.Content[0].Value)
}

func TestBuildLowHeader_Success(t *testing.T) {
	yml := `description: A test header
required: true`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	assert.NoError(t, err)

	result, err := buildLowHeader(node.Content[0], nil)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "A test header", result.Description.Value)
}

func TestBuildLowHeader_BuildError(t *testing.T) {
	yml := `description: test
schema:
  $ref: '#/components/schemas/DoesNotExist'`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	assert.NoError(t, err)

	config := index.CreateOpenAPIIndexConfig()
	idx := index.NewSpecIndexWithConfig(&node, config)

	result, err := buildLowHeader(node.Content[0], idx)

	assert.Error(t, err)
	assert.Nil(t, result)
}
