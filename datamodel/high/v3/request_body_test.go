// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestRequestBody_MarshalYAML(t *testing.T) {
	ext := orderedmap.New[string, *yaml.Node]()
	ext.Set("x-high-gravity", utils.CreateStringNode("why not?"))

	rb := true
	req := &RequestBody{
		Description: "beer",
		Required:    &rb,
		Extensions:  ext,
	}

	rend, _ := req.Render()

	desired := `description: beer
required: true
x-high-gravity: why not?`

	assert.Equal(t, desired, strings.TrimSpace(string(rend)))
}

func TestRequestBody_MarshalYAMLInline(t *testing.T) {
	ext := orderedmap.New[string, *yaml.Node]()
	ext.Set("x-high-gravity", utils.CreateStringNode("why not?"))

	rb := true
	req := &RequestBody{
		Description: "beer",
		Required:    &rb,
		Extensions:  ext,
	}

	rend, _ := req.RenderInline()

	desired := `description: beer
required: true
x-high-gravity: why not?`

	assert.Equal(t, desired, strings.TrimSpace(string(rend)))
}

func TestRequestBody_MarshalNoRequired(t *testing.T) {
	ext := orderedmap.New[string, *yaml.Node]()
	ext.Set("x-high-gravity", utils.CreateStringNode("why not?"))

	rb := false
	req := &RequestBody{
		Description: "beer",
		Required:    &rb,
		Extensions:  ext,
	}

	rend, _ := req.Render()

	desired := `description: beer
required: false
x-high-gravity: why not?`

	assert.Equal(t, desired, strings.TrimSpace(string(rend)))
}

func TestRequestBody_MarshalRequiredNil(t *testing.T) {
	ext := orderedmap.New[string, *yaml.Node]()
	ext.Set("x-high-gravity", utils.CreateStringNode("why not?"))

	req := &RequestBody{
		Description: "beer",
		Extensions:  ext,
	}

	rend, _ := req.Render()

	desired := `description: beer
x-high-gravity: why not?`

	assert.Equal(t, desired, strings.TrimSpace(string(rend)))
}

func TestCreateRequestBodyRef(t *testing.T) {
	ref := "#/components/requestBodies/UserInput"
	rb := CreateRequestBodyRef(ref)

	assert.True(t, rb.IsReference())
	assert.Equal(t, ref, rb.GetReference())
	assert.Nil(t, rb.GoLow())
}

func TestRequestBody_MarshalYAML_Reference(t *testing.T) {
	rb := CreateRequestBodyRef("#/components/requestBodies/UserInput")

	node, err := rb.MarshalYAML()
	assert.NoError(t, err)

	yamlNode, ok := node.(*yaml.Node)
	assert.True(t, ok)
	assert.Equal(t, yaml.MappingNode, yamlNode.Kind)
	assert.Equal(t, 2, len(yamlNode.Content))
	assert.Equal(t, "$ref", yamlNode.Content[0].Value)
	assert.Equal(t, "#/components/requestBodies/UserInput", yamlNode.Content[1].Value)
}

func TestRequestBody_MarshalYAMLInline_Reference(t *testing.T) {
	rb := CreateRequestBodyRef("#/components/requestBodies/UserInput")

	node, err := rb.MarshalYAMLInline()
	assert.NoError(t, err)

	yamlNode, ok := node.(*yaml.Node)
	assert.True(t, ok)
	assert.Equal(t, "$ref", yamlNode.Content[0].Value)
}

func TestRequestBody_Reference_TakesPrecedence(t *testing.T) {
	// When both Reference and content are set, Reference should take precedence
	rb := &RequestBody{
		Reference:   "#/components/requestBodies/foo",
		Description: "shouldBeIgnored",
	}

	assert.True(t, rb.IsReference())

	node, err := rb.MarshalYAML()
	assert.NoError(t, err)

	// Should render as $ref only, not full request body
	rendered, _ := yaml.Marshal(node)
	assert.Contains(t, string(rendered), "$ref")
	assert.NotContains(t, string(rendered), "shouldBeIgnored")
}

func TestRequestBody_Render_Reference(t *testing.T) {
	rb := CreateRequestBodyRef("#/components/requestBodies/UserInput")

	rendered, err := rb.Render()
	assert.NoError(t, err)

	assert.Contains(t, string(rendered), "$ref")
	assert.Contains(t, string(rendered), "#/components/requestBodies/UserInput")
}

func TestRequestBody_IsReference_False(t *testing.T) {
	rb := &RequestBody{
		Description: "A request body",
	}
	assert.False(t, rb.IsReference())
	assert.Equal(t, "", rb.GetReference())
}
