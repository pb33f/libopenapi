// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"strings"
	"testing"

	"github.com/pkg-base/libopenapi/datamodel/high/base"
	"github.com/pkg-base/libopenapi/datamodel/low"
	v3 "github.com/pkg-base/libopenapi/datamodel/low/v3"
	"github.com/pkg-base/libopenapi/index"
	"github.com/pkg-base/libopenapi/orderedmap"
	"github.com/pkg-base/libopenapi/utils"
	"github.com/pkg-base/yaml"
	"github.com/stretchr/testify/assert"
)

func TestParameter_MarshalYAML(t *testing.T) {
	ext := orderedmap.New[string, *yaml.Node]()
	ext.Set("x-burgers", utils.CreateStringNode("why not?"))

	explode := true
	param := Parameter{
		Name:          "chicken",
		In:            "nuggets",
		Description:   "beefy",
		Deprecated:    true,
		Style:         "simple",
		Explode:       &explode,
		AllowReserved: true,
		Example:       utils.CreateStringNode("example"),
		Examples: orderedmap.ToOrderedMap(map[string]*base.Example{
			"example": {Value: utils.CreateStringNode("example")},
		}),
		Extensions: ext,
	}

	rend, _ := param.Render()

	desired := `name: chicken
in: nuggets
description: beefy
deprecated: true
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

func TestParameter_MarshalYAMLInline(t *testing.T) {
	ext := orderedmap.New[string, *yaml.Node]()
	ext.Set("x-burgers", utils.CreateStringNode("why not?"))

	explode := true
	param := Parameter{
		Name:          "chicken",
		In:            "nuggets",
		Description:   "beefy",
		Deprecated:    true,
		Style:         "simple",
		Explode:       &explode,
		AllowReserved: true,
		Example:       utils.CreateStringNode("example"),
		Examples: orderedmap.ToOrderedMap(map[string]*base.Example{
			"example": {Value: utils.CreateStringNode("example")},
		}),
		Extensions: ext,
	}

	rend, _ := param.RenderInline()

	desired := `name: chicken
in: nuggets
description: beefy
deprecated: true
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

func TestParameter_IsExploded(t *testing.T) {
	explode := true
	param := Parameter{
		Explode: &explode,
	}

	assert.True(t, param.IsExploded())

	explode = false
	param = Parameter{
		Explode: &explode,
	}

	assert.False(t, param.IsExploded())

	param = Parameter{}

	assert.False(t, param.IsExploded())
}

func TestParameter_IsDefaultFormEncoding(t *testing.T) {
	param := Parameter{}
	assert.True(t, param.IsDefaultFormEncoding())

	param = Parameter{Style: "form"}
	assert.True(t, param.IsDefaultFormEncoding())

	explode := false
	param = Parameter{
		Explode: &explode,
	}
	assert.False(t, param.IsDefaultFormEncoding())

	explode = true
	param = Parameter{
		Explode: &explode,
	}
	assert.True(t, param.IsDefaultFormEncoding())

	param = Parameter{
		Explode: &explode,
		Style:   "simple",
	}
	assert.False(t, param.IsDefaultFormEncoding())
}

func TestParameter_IsDefaultHeaderEncoding(t *testing.T) {
	param := Parameter{}
	assert.True(t, param.IsDefaultHeaderEncoding())

	param = Parameter{Style: "simple"}
	assert.True(t, param.IsDefaultHeaderEncoding())

	explode := false
	param = Parameter{
		Explode: &explode,
		Style:   "simple",
	}
	assert.True(t, param.IsDefaultHeaderEncoding())

	explode = true
	param = Parameter{
		Explode: &explode,
		Style:   "simple",
	}
	assert.False(t, param.IsDefaultHeaderEncoding())

	explode = false
	param = Parameter{
		Explode: &explode,
		Style:   "form",
	}
	assert.False(t, param.IsDefaultHeaderEncoding())
}

func TestParameter_IsDefaultPathEncoding(t *testing.T) {
	param := Parameter{}
	assert.True(t, param.IsDefaultPathEncoding())
}

func TestParameter_Examples(t *testing.T) {
	yml := `examples:
    pbjBurger:
        summary: A horrible, nutty, sticky mess.
        value:
            name: Peanut And Jelly
            numPatties: 3
    cakeBurger:
        summary: A sickly, sweet, atrocity
        value:
            name: Chocolate Cake Burger
            numPatties: 5`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())

	var n v3.Parameter
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	r := NewParameter(&n)

	assert.Equal(t, 2, orderedmap.Len(r.Examples))
}

func TestParameter_Examples_NotFromSchema(t *testing.T) {
	yml := `schema:
  type: string
  examples:
    - example 1
    - example 2
    - example 3`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())

	var n v3.Parameter
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	r := NewParameter(&n)

	assert.Equal(t, 0, orderedmap.Len(r.Examples))
}
