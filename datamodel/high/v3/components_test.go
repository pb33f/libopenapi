// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"github.com/pb33f/libopenapi/datamodel/low"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"strings"
	"testing"
)

func TestComponents_MarshalYAML(t *testing.T) {

	comp := &Components{
		Responses: map[string]*Response{
			"200": {
				Description: "OK",
			},
		},
		Parameters: map[string]*Parameter{
			"id": {
				Name: "id",
				In:   "path",
			},
		},
		RequestBodies: map[string]*RequestBody{
			"body": {
				Content: map[string]*MediaType{
					"application/json": {
						Example: "why?",
					},
				},
			},
		},
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
                example: why?`

	dat, _ = r.Render()
	assert.Equal(t, desired, strings.TrimSpace(string(dat)))
}
