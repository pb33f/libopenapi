// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
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
