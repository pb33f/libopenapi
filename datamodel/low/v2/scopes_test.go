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

func TestScopes_Hash(t *testing.T) {

	yml := `burgers: chips
pizza: beans
x-men: needs a reboot or a refresh`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Scopes
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(idxNode.Content[0], idx)

	yml2 := `x-men: needs a reboot or a refresh
pizza: beans
burgers: chips`

	var idxNode2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml2), &idxNode2)
	idx2 := index.NewSpecIndex(&idxNode2)

	var n2 Scopes
	_ = low.BuildModel(idxNode2.Content[0], &n2)
	_ = n2.Build(idxNode2.Content[0], idx2)

	// hash
	assert.Equal(t, n.Hash(), n2.Hash())
	assert.Len(t, n.GetExtensions(), 1)

}
