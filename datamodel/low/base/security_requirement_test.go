// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
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

	_ = sr.Build(context.Background(), nil, idxNode.Content[0], nil)
	_ = sr2.Build(context.Background(), nil, idxNode2.Content[0], nil)

	assert.Equal(t, 2, orderedmap.Len(sr.Requirements.Value))
	assert.Len(t, sr.GetKeys(), 2)
	assert.Len(t, sr.FindRequirement("one"), 2)
	assert.Equal(t, sr.Hash(), sr2.Hash())
	assert.Nil(t, sr.FindRequirement("i-do-not-exist"))
}

func TestSecurityRequirement_TestEmptyReq(t *testing.T) {

	yml := `one:
  - two
  - {}`

	var sr SecurityRequirement
	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	_ = sr.Build(context.Background(), nil, idxNode.Content[0], nil)

	assert.Equal(t, 1, orderedmap.Len(sr.Requirements.Value))
	assert.Len(t, sr.GetKeys(), 1)
	assert.True(t, sr.ContainsEmptyRequirement)

}

func TestSecurityRequirement_TestEmptyContent(t *testing.T) {
	var sr SecurityRequirement
	_ = sr.Build(context.Background(), nil, &yaml.Node{}, nil)
	assert.True(t, sr.ContainsEmptyRequirement)

}
