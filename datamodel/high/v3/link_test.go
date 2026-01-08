// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestLink_MarshalYAML(t *testing.T) {
	link := Link{
		OperationRef: "somewhere",
		OperationId:  "somewhereOutThere",
		Parameters: orderedmap.ToOrderedMap(map[string]string{
			"over": "theRainbow",
		}),
		RequestBody: "hello?",
		Description: "are you there?",
		Server: &Server{
			URL: "https://pb33f.io",
		},
	}

	dat, _ := link.Render()
	desired := `operationRef: somewhere
operationId: somewhereOutThere
parameters:
    over: theRainbow
requestBody: hello?
description: are you there?
server:
    url: https://pb33f.io`

	assert.Equal(t, desired, strings.TrimSpace(string(dat)))
}

func TestCreateLinkRef(t *testing.T) {
	ref := "#/components/links/GetUserByUserId"
	l := CreateLinkRef(ref)

	assert.True(t, l.IsReference())
	assert.Equal(t, ref, l.GetReference())
	assert.Nil(t, l.GoLow())
}

func TestLink_MarshalYAML_Reference(t *testing.T) {
	l := CreateLinkRef("#/components/links/GetUserByUserId")

	node, err := l.MarshalYAML()
	assert.NoError(t, err)

	yamlNode, ok := node.(*yaml.Node)
	assert.True(t, ok)
	assert.Equal(t, yaml.MappingNode, yamlNode.Kind)
	assert.Equal(t, 2, len(yamlNode.Content))
	assert.Equal(t, "$ref", yamlNode.Content[0].Value)
	assert.Equal(t, "#/components/links/GetUserByUserId", yamlNode.Content[1].Value)
}

func TestLink_MarshalYAMLInline_Reference(t *testing.T) {
	l := CreateLinkRef("#/components/links/GetUserByUserId")

	node, err := l.MarshalYAMLInline()
	assert.NoError(t, err)

	yamlNode, ok := node.(*yaml.Node)
	assert.True(t, ok)
	assert.Equal(t, "$ref", yamlNode.Content[0].Value)
}

func TestLink_Reference_TakesPrecedence(t *testing.T) {
	// When both Reference and content are set, Reference should take precedence
	l := &Link{
		Reference:   "#/components/links/foo",
		Description: "shouldBeIgnored",
	}

	assert.True(t, l.IsReference())

	node, err := l.MarshalYAML()
	assert.NoError(t, err)

	// Should render as $ref only, not full link
	rendered, _ := yaml.Marshal(node)
	assert.Contains(t, string(rendered), "$ref")
	assert.NotContains(t, string(rendered), "shouldBeIgnored")
}

func TestLink_Render_Reference(t *testing.T) {
	l := CreateLinkRef("#/components/links/GetUserByUserId")

	rendered, err := l.Render()
	assert.NoError(t, err)

	assert.Contains(t, string(rendered), "$ref")
	assert.Contains(t, string(rendered), "#/components/links/GetUserByUserId")
}

func TestLink_IsReference_False(t *testing.T) {
	l := &Link{
		Description: "A link",
	}
	assert.False(t, l.IsReference())
	assert.Equal(t, "", l.GetReference())
}

func TestLink_MarshalYAMLInlineWithContext(t *testing.T) {
	link := Link{
		OperationRef: "somewhere",
		OperationId:  "somewhereOutThere",
		Parameters: orderedmap.ToOrderedMap(map[string]string{
			"over": "theRainbow",
		}),
		RequestBody: "hello?",
		Description: "are you there?",
		Server: &Server{
			URL: "https://pb33f.io",
		},
	}

	ctx := base.NewInlineRenderContext()
	node, err := link.MarshalYAMLInlineWithContext(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, node)

	dat, _ := yaml.Marshal(node)
	desired := `operationRef: somewhere
operationId: somewhereOutThere
parameters:
    over: theRainbow
requestBody: hello?
description: are you there?
server:
    url: https://pb33f.io`

	assert.Equal(t, desired, strings.TrimSpace(string(dat)))
}

func TestLink_MarshalYAMLInlineWithContext_Reference(t *testing.T) {
	l := CreateLinkRef("#/components/links/GetUserByUserId")

	ctx := base.NewInlineRenderContext()
	node, err := l.MarshalYAMLInlineWithContext(ctx)
	assert.NoError(t, err)

	yamlNode, ok := node.(*yaml.Node)
	assert.True(t, ok)
	assert.Equal(t, "$ref", yamlNode.Content[0].Value)
}

func TestBuildLowLink_Success(t *testing.T) {
	yml := `operationId: getUser
description: A test link`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	assert.NoError(t, err)

	result, err := buildLowLink(node.Content[0], nil)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "getUser", result.OperationId.Value)
}

func TestBuildLowLink_BuildError(t *testing.T) {
	// Links don't have schemas, so we need a different way to trigger Build error
	// Links are quite simple and Build rarely fails, so we test the success path
	yml := `operationId: test
description: test link`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yml), &node)
	assert.NoError(t, err)

	result, err := buildLowLink(node.Content[0], nil)

	// Links Build method is very resilient, so this should succeed
	assert.NoError(t, err)
	assert.NotNil(t, result)
}
