// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestItems_Build(t *testing.T) {
	yml := `items:
  $ref: break`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Items
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestItems_DefaultAsSlice(t *testing.T) {
	yml := `x-thing: thing
default:
  - pizza
  - cake`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Items
	_ = low.BuildModel(&idxNode, &n)
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	var def []string
	_ = n.Default.Value.Decode(&def)

	assert.Len(t, def, 2)
	assert.Equal(t, 1, orderedmap.Len(n.GetExtensions()))
}

func TestItems_DefaultAsMap(t *testing.T) {
	yml := `default:
  hot: pizza
  tasty: beer`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Items
	_ = low.BuildModel(&idxNode, &n)
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	var def map[string]string
	_ = n.Default.GetValue().Decode(&def)

	assert.Len(t, def, 2)
}

func TestItems_Hash_n_Grab(t *testing.T) {
	yml := `type: string
format: left
collectionFormat: nice
default: shut that door!
pattern: wow
enum:
  - one
  - 123
x-belly: large
items:
 type: int
maximum: 10
minimum: 1
exclusiveMinimum: true
exclusiveMaximum: true
maxLength: 10
minLength: 1
maxItems: 10
minItems: 1
uniqueItems: true
multipleOf: 12`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Items
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	yml2 := `items:
 type: int
format: left
collectionFormat: nice
type: string
maximum: 10
minimum: 1
exclusiveMinimum: true
exclusiveMaximum: true
maxLength: 10
minLength: 1
maxItems: 10
minItems: 1
uniqueItems: true
multipleOf: 12
default: shut that door!
enum:
  - one
  - 123
x-belly: large
pattern: wow
`

	var idxNode2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml2), &idxNode2)
	idx2 := index.NewSpecIndex(&idxNode2)

	var n2 Items
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
	_ = n.FindExtension("x-belly").GetValue().Decode(&xBelly)
	assert.Equal(t, "large", xBelly)
}

func TestItems_GetDescription(t *testing.T) {
	i := Items{}
	assert.Nil(t, i.GetDescription())
}
