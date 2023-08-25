// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestSecurityRequirement_Build(t *testing.T) {

	yml := `one:
  - two
  - three
four: 
  - five
  - six`

	var sr SecurityRequirement
	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	yml2 := `four: 
  - six
  - five
one:
  - three
  - two`

	var sr2 SecurityRequirement
	var idxNode2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml2), &idxNode2)

	_ = sr.Build(nil, idxNode.Content[0], nil)
	_ = sr2.Build(nil, idxNode2.Content[0], nil)

	assert.Len(t, sr.Requirements.Value, 2)
	assert.Len(t, sr.GetKeys(), 2)
	assert.Len(t, sr.FindRequirement("one"), 2)
	assert.Equal(t, sr.Hash(), sr2.Hash())
	assert.Nil(t, sr.FindRequirement("i-do-not-exist"))
}
