// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestDiscriminator_FindMappingValue(t *testing.T) {
	yml := `propertyName: freshCakes
mapping:
  something: nothing`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)

	var n Discriminator
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)
	assert.Equal(t, "nothing", n.FindMappingValue("something").Value)
	assert.Nil(t, n.FindMappingValue("freshCakes"))
}

func TestDiscriminator_Hash(t *testing.T) {
	left := `propertyName: freshCakes
mapping:
  something: nothing`

	right := `mapping:
  something: nothing
propertyName: freshCakes`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc Discriminator
	var rDoc Discriminator
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	lDoc.RootNode = &lNode
	lDoc.KeyNode = &rNode

	assert.Equal(t, lDoc.Hash(), rDoc.Hash())
	assert.NotNil(t, lDoc.GetRootNode())
	assert.NotNil(t, lDoc.GetKeyNode())
}

func TestDiscriminator_DefaultMapping_OpenAPI32(t *testing.T) {
	yml := `propertyName: petType
mapping:
  dog: '#/components/schemas/Dog'
  cat: '#/components/schemas/Cat'
defaultMapping: '#/components/schemas/UnknownPet'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)

	var n Discriminator
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)
	assert.Equal(t, "petType", n.PropertyName.Value)
	assert.Equal(t, "#/components/schemas/UnknownPet", n.DefaultMapping.Value)
	assert.Equal(t, "#/components/schemas/Dog", n.FindMappingValue("dog").Value)
	assert.Equal(t, "#/components/schemas/Cat", n.FindMappingValue("cat").Value)
}

func TestDiscriminator_DefaultMapping_Hash(t *testing.T) {
	left := `propertyName: petType
mapping:
  dog: '#/components/schemas/Dog'
defaultMapping: '#/components/schemas/UnknownPet'`

	right := `propertyName: petType
defaultMapping: '#/components/schemas/UnknownPet'
mapping:
  dog: '#/components/schemas/Dog'`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	var lDoc Discriminator
	var rDoc Discriminator
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)

	// Same content, different order should produce same hash
	assert.Equal(t, lDoc.Hash(), rDoc.Hash())
}

func TestDiscriminator_DefaultMapping_HashDifferent(t *testing.T) {
	left := `propertyName: petType
defaultMapping: '#/components/schemas/UnknownPet'`

	right := `propertyName: petType
defaultMapping: '#/components/schemas/OtherPet'`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	var lDoc Discriminator
	var rDoc Discriminator
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)

	// Different defaultMapping should produce different hash
	assert.NotEqual(t, lDoc.Hash(), rDoc.Hash())
}
