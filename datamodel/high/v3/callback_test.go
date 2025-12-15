// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestCallback_MarshalYAML(t *testing.T) {
	ext := orderedmap.New[string, *yaml.Node]()
	ext.Set("x-burgers", utils.CreateStringNode("why not?"))

	cb := &Callback{
		Expression: orderedmap.ToOrderedMap(map[string]*PathItem{
			"https://pb33f.io": {
				Get: &Operation{
					OperationId: "oneTwoThree",
				},
			},
			"https://pb33f.io/libopenapi": {
				Get: &Operation{
					OperationId: "openaypeeeye",
				},
			},
		}),
		Extensions: ext,
	}

	rend, _ := cb.Render()

	// there is no way to determine order in brand new maps, so we have to check length.
	assert.Len(t, rend, 152)

	// mutate
	cb.Expression.GetOrZero("https://pb33f.io").Get.OperationId = "blim-blam"

	ext = orderedmap.New[string, *yaml.Node]()
	ext.Set("x-burgers", utils.CreateStringNode("yes please!"))
	cb.Extensions = ext

	rend, _ = cb.Render()
	// there is no way to determine order in brand new maps, so we have to check length.
	assert.Len(t, rend, 153)

	k := `x-break-everything: please
'{$request.query.queryUrl}':
    post:
        description: Callback payload
        responses:
            "200":
                description: callback successfully processed
                content:
                    application/json:
                        schema:
                            type: string`

	var idxNode yaml.Node
	err := yaml.Unmarshal([]byte(k), &idxNode)
	assert.NoError(t, err)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())

	var n v3.Callback
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	r := NewCallback(&n)

	var xBreakEverything string
	_ = r.Extensions.GetOrZero("x-break-everything").Decode(&xBreakEverything)

	assert.Equal(t, "please", xBreakEverything)

	rend, _ = r.Render()
	assert.Equal(t, k, strings.TrimSpace(string(rend)))
}

func TestCallback_RenderInline(t *testing.T) {
	ext := orderedmap.New[string, *yaml.Node]()
	ext.Set("x-burgers", utils.CreateStringNode("why not?"))

	cb := &Callback{
		Expression: orderedmap.ToOrderedMap(map[string]*PathItem{
			"https://pb33f.io": {
				Get: &Operation{
					OperationId: "oneTwoThree",
				},
			},
			"https://pb33f.io/libopenapi": {
				Get: &Operation{
					OperationId: "openaypeeeye",
				},
			},
		}),
		Extensions: ext,
	}

	rend, _ := cb.RenderInline()
	assert.Equal(t, "x-burgers: why not?\nhttps://pb33f.io:\n    get:\n        operationId: oneTwoThree\nhttps://pb33f.io/libopenapi:\n    get:\n        operationId: openaypeeeye\n", string(rend))
}

func TestCreateCallbackRef(t *testing.T) {
	ref := "#/components/callbacks/WebhookCallback"
	cb := CreateCallbackRef(ref)

	assert.True(t, cb.IsReference())
	assert.Equal(t, ref, cb.GetReference())
	assert.Nil(t, cb.GoLow())
}

func TestCallback_MarshalYAML_Reference(t *testing.T) {
	cb := CreateCallbackRef("#/components/callbacks/WebhookCallback")

	node, err := cb.MarshalYAML()
	assert.NoError(t, err)

	yamlNode, ok := node.(*yaml.Node)
	assert.True(t, ok)
	assert.Equal(t, yaml.MappingNode, yamlNode.Kind)
	assert.Equal(t, 2, len(yamlNode.Content))
	assert.Equal(t, "$ref", yamlNode.Content[0].Value)
	assert.Equal(t, "#/components/callbacks/WebhookCallback", yamlNode.Content[1].Value)
}

func TestCallback_MarshalYAMLInline_Reference(t *testing.T) {
	cb := CreateCallbackRef("#/components/callbacks/WebhookCallback")

	node, err := cb.MarshalYAMLInline()
	assert.NoError(t, err)

	yamlNode, ok := node.(*yaml.Node)
	assert.True(t, ok)
	assert.Equal(t, "$ref", yamlNode.Content[0].Value)
}

func TestCallback_Reference_TakesPrecedence(t *testing.T) {
	// When both Reference and content are set, Reference should take precedence
	cb := &Callback{
		Reference: "#/components/callbacks/foo",
		Expression: orderedmap.ToOrderedMap(map[string]*PathItem{
			"https://example.com": {
				Get: &Operation{
					OperationId: "shouldBeIgnored",
				},
			},
		}),
	}

	assert.True(t, cb.IsReference())

	node, err := cb.MarshalYAML()
	assert.NoError(t, err)

	// Should render as $ref only, not full callback
	rendered, _ := yaml.Marshal(node)
	assert.Contains(t, string(rendered), "$ref")
	assert.NotContains(t, string(rendered), "shouldBeIgnored")
}

func TestCallback_Render_Reference(t *testing.T) {
	cb := CreateCallbackRef("#/components/callbacks/WebhookCallback")

	rendered, err := cb.Render()
	assert.NoError(t, err)

	assert.Contains(t, string(rendered), "$ref")
	assert.Contains(t, string(rendered), "#/components/callbacks/WebhookCallback")
}

func TestCallback_IsReference_False(t *testing.T) {
	cb := &Callback{
		Expression: orderedmap.ToOrderedMap(map[string]*PathItem{
			"https://example.com": {},
		}),
	}
	assert.False(t, cb.IsReference())
	assert.Equal(t, "", cb.GetReference())
}
