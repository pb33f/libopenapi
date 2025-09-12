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

func TestComponents_MarshalYAML(t *testing.T) {
	comp := &Components{
		Responses: orderedmap.ToOrderedMap(map[string]*Response{
			"200": {
				Description: "OK",
			},
		}),
		Parameters: orderedmap.ToOrderedMap(map[string]*Parameter{
			"id": {
				Name: "id",
				In:   "path",
			},
		}),
		RequestBodies: orderedmap.ToOrderedMap(map[string]*RequestBody{
			"body": {
				Content: orderedmap.ToOrderedMap(map[string]*MediaType{
					"application/json": {
						Example: utils.CreateStringNode("why?"),
					},
				}),
			},
		}),
		PathItems: orderedmap.ToOrderedMap(map[string]*PathItem{
			"/ding/dong/{bing}/{bong}/go": {
				Get: &Operation{
					Description: "get",
				},
			},
		}),
	}

	dat, _ := comp.Render()

	var idxNode yaml.Node
	_ = yaml.Unmarshal(dat, &idxNode)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())

	var n v3.Components
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(context.Background(), idxNode.Content[0], idx)

	r := NewComponents(&n)

	desired := `responses:
    "200":
        description: OK
parameters:
    id:
        name: id
        in: path
requestBodies:
    body:
        content:
            application/json:
                example: why?
pathItems:
    /ding/dong/{bing}/{bong}/go:
        get:
            description: get`

	dat, _ = r.Render()
	assert.Equal(t, desired, strings.TrimSpace(string(dat)))
	assert.NotNil(t, r.GoLowUntyped())
}
