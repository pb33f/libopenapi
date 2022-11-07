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

func TestSecurityScheme_Build_Borked(t *testing.T) {

	yml := `scopes:
  $ref: break`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n SecurityScheme
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.Error(t, err)

}

func TestSecurityScheme_Build_Scopes(t *testing.T) {

	yml := `scopes:
  some:thing: here
  something: there`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	var n SecurityScheme
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Len(t, n.Scopes.Value.Values, 2)

}

func TestSecurityScheme_Hash(t *testing.T) {

	yml := `type: secure
description: a very secure thing
name: securityPerson
in: my heart
flow: watery
authorizationUrl: https://pb33f.io
tokenUrl: https://pb33f.io/token
scopes:
  fish:monkey
x-beer: not for a while`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n SecurityScheme
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(idxNode.Content[0], idx)

	yml2 := `in: my heart
scopes:
  fish:monkey
name: securityPerson
type: secure
flow: watery
description: a very secure thing
tokenUrl: https://pb33f.io/token
x-beer: not for a while
authorizationUrl: https://pb33f.io
`

	var idxNode2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml2), &idxNode2)
	idx2 := index.NewSpecIndex(&idxNode2)

	var n2 SecurityScheme
	_ = low.BuildModel(idxNode2.Content[0], &n2)
	_ = n2.Build(idxNode2.Content[0], idx2)

	// hash
	assert.Equal(t, n.Hash(), n2.Hash())

}
