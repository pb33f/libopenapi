// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"strings"
	"testing"

	"github.com/pkg-base/libopenapi/datamodel/low"
	v3 "github.com/pkg-base/libopenapi/datamodel/low/v3"
	"github.com/pkg-base/libopenapi/index"
	"github.com/pkg-base/libopenapi/orderedmap"
	"github.com/pkg-base/libopenapi/utils"
	"github.com/pkg-base/yaml"
	"github.com/stretchr/testify/assert"
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
