// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestSecurityRequirement_Build(t *testing.T) {
	t.Parallel()
	yml := `something:
  - read:me
  - write:me`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n base.SecurityRequirement
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(nil, idxNode.Content[0], idx)

	assert.NoError(t, err)
	assert.Len(t, n.Requirements.Value, 1)
	assert.Equal(t, "read:me", n.FindRequirement("something")[0].Value)
	assert.Equal(t, "write:me", n.FindRequirement("something")[1].Value)
	assert.Nil(t, n.FindRequirement("none"))
}

func TestSecurityScheme_Build(t *testing.T) {
	t.Parallel()
	yml := `type: tea
description: cake
name: biscuit
in: jar
scheme: lovely
bearerFormat: wow
flows:
 implicit:
  tokenUrl: https://pb33f.io
openIdConnectUrl: https://pb33f.io/openid  
x-milk: please`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n SecurityScheme
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(nil, idxNode.Content[0], idx)
	assert.NoError(t, err)

	assert.Equal(t, "0b5ee36519fdfc6383c7befd92294d77b5799cd115911ff8c3e194f345a8c103",
		low.GenerateHashString(&n))

	assert.Equal(t, "tea", n.Type.Value)
	assert.Equal(t, "cake", n.Description.Value)
	assert.Equal(t, "biscuit", n.Name.Value)
	assert.Equal(t, "jar", n.In.Value)
	assert.Equal(t, "lovely", n.Scheme.Value)
	assert.Equal(t, "wow", n.BearerFormat.Value)
	assert.Equal(t, "https://pb33f.io/openid", n.OpenIdConnectUrl.Value)
	assert.Equal(t, "please", n.FindExtension("x-milk").Value)
	assert.Equal(t, "https://pb33f.io", n.Flows.Value.Implicit.Value.TokenUrl.Value)
	assert.Len(t, n.GetExtensions(), 1)

}

func TestSecurityScheme_Build_Fail(t *testing.T) {
	t.Parallel()
	yml := `flows:
  $ref: #bork`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n SecurityScheme
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(nil, idxNode.Content[0], idx)
	assert.Error(t, err)
}
