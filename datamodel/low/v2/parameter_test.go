// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestParameter_Build(t *testing.T) {
	yml := `$ref: break`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Parameter
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestParameter_Build_Items(t *testing.T) {
	yml := `items:
  $ref: break`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Parameter
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestParameter_DefaultSlice(t *testing.T) {
	yml := `default:
  - things
  - junk
  - stuff`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Parameter
	_ = low.BuildModel(&idxNode, &n)

	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	var a []any
	_ = n.Default.Value.Decode(&a)

	assert.Len(t, a, 3)
}

func TestParameter_DefaultMap(t *testing.T) {
	yml := `default:
  things: junk
  stuff: more junk`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Parameter
	_ = low.BuildModel(&idxNode, &n)

	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	var m map[string]any
	_ = n.Default.Value.Decode(&m)

	assert.Len(t, m, 2)
}

func TestParameter_NoDefaultNoError(t *testing.T) {
	yml := `name: pizza-pie`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Parameter
	_ = low.BuildModel(&idxNode, &n)

	err := n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)
}

func TestParameter_Hash_n_Grab(t *testing.T) {
	yml := `name: mcmuffin
in: my-belly
description: tasty!
type: string
format: left
collectionFormat: nice
default: shut that door!
pattern: wow
schema:
  type: int
enum:
  - one
  - 123
x-belly: large
items:
 type: int
maximum: 10
minimum: 1
allowEmptyValue: true
exclusiveMinimum: true
exclusiveMaximum: true
maxLength: 10
minLength: 1
maxItems: 10
minItems: 1
uniqueItems: true
multipleOf: 12
required: true`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Parameter
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	yml2 := `items:
 type: int
format: left
collectionFormat: nice
type: string
maximum: 10
required: true
minimum: 1
name: mcmuffin
in: my-belly
description: tasty!
exclusiveMinimum: true
exclusiveMaximum: true
maxLength: 10
minLength: 1
maxItems: 10
minItems: 1
uniqueItems: true
multipleOf: 12
default: shut that door!
schema: 
  type: int
enum:
  - one
  - 123
x-belly: large
pattern: wow
allowEmptyValue: true
`

	var idxNode2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml2), &idxNode2)
	idx2 := index.NewSpecIndex(&idxNode2)

	var n2 Parameter
	_ = low.BuildModel(idxNode2.Content[0], &n2)
	_ = n2.Build(context.Background(), nil, idxNode2.Content[0], idx2)

	// hash
	assert.Equal(t, n.Hash(), n2.Hash())

	// and grab
	assert.Equal(t, "string", n.GetType().Value)
	assert.Equal(t, "left", n.GetFormat().Value)
	assert.Equal(t, "left", n.GetFormat().Value)
	assert.Equal(t, "nice", n.GetCollectionFormat().Value)

	var def string
	_ = n.GetDefault().Value.Decode(&def)
	assert.Equal(t, "shut that door!", def)
	assert.Equal(t, 10, n.GetMaximum().Value)
	assert.Equal(t, 1, n.GetMinimum().Value)
	assert.True(t, n.GetExclusiveMinimum().Value)
	assert.True(t, n.GetExclusiveMaximum().Value)
	assert.Equal(t, 10, n.GetMaxLength().Value)
	assert.Equal(t, 1, n.GetMinLength().Value)
	assert.Equal(t, 10, n.GetMaxItems().Value)
	assert.Equal(t, 1, n.GetMinItems().Value)
	assert.True(t, n.GetUniqueItems().Value)
	assert.Equal(t, 12, n.GetMultipleOf().Value)
	assert.Equal(t, "wow", n.GetPattern().Value)
	assert.Equal(t, "int", n.GetItems().Value.(*Items).Type.Value)
	assert.Len(t, n.GetEnum().Value, 2)

	var xBelly string
	_ = n.FindExtension("x-belly").Value.Decode(&xBelly)
	assert.Equal(t, "large", xBelly)
	assert.Equal(t, "tasty!", n.GetDescription().Value)
	assert.Equal(t, "mcmuffin", n.GetName().Value)
	assert.Equal(t, "my-belly", n.GetIn().Value)

	v := n.GetSchema().Value.(*base.SchemaProxy).Schema().Type // this is a dynamic value that has multiple choices
	assert.Equal(t, "int", v.Value.A)                          // A is v2
	assert.True(t, n.GetRequired().Value)
	assert.True(t, n.GetAllowEmptyValue().Value)
	assert.Equal(t, 1, orderedmap.Len(n.GetExtensions()))
}
