// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestHeader_Build(t *testing.T) {

	yml := `items:
  $ref: break`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n Header
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.Error(t, err)

}

func TestHeader_DefaultAsSlice(t *testing.T) {

	yml := `x-ext: thing
default:
  - why
  - so many
  - variations`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Header
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(idxNode.Content[0], idx)

	assert.NotNil(t, n.Default.Value)
	assert.Len(t, n.Default.Value, 3)
	assert.Len(t, n.GetExtensions(), 1)
}

func TestHeader_DefaultAsObject(t *testing.T) {

	yml := `default:
  lets:
    create:
      a:
       thing: ok?`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Header
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(idxNode.Content[0], idx)

	assert.NotNil(t, n.Default.Value)
}

func TestHeader_NoDefault(t *testing.T) {

	yml := `minimum: 12`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Header
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(idxNode.Content[0], idx)

	assert.Equal(t, 12, n.Minimum.Value)
}

func TestHeader_Hash_n_Grab(t *testing.T) {

	yml := `description: head
type: string
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

	var n Header
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(idxNode.Content[0], idx)

	yml2 := `description: head
items:
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

	var n2 Header
	_ = low.BuildModel(idxNode2.Content[0], &n2)
	_ = n2.Build(idxNode2.Content[0], idx2)

	// hash
	assert.Equal(t, n.Hash(), n2.Hash())

	// and grab
	assert.Equal(t, "string", n.GetType().Value)
	assert.Equal(t, "head", n.GetDescription().Value)
	assert.Equal(t, "left", n.GetFormat().Value)
	assert.Equal(t, "left", n.GetFormat().Value)
	assert.Equal(t, "nice", n.GetCollectionFormat().Value)
	assert.Equal(t, "shut that door!", n.GetDefault().Value)
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
	assert.Equal(t, "large", n.FindExtension("x-belly").Value)

}
