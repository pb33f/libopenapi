// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"testing"

	"github.com/pkg-base/libopenapi/datamodel/low"
	"github.com/pkg-base/libopenapi/datamodel/low/base"
	"github.com/pkg-base/libopenapi/index"
	"github.com/pkg-base/libopenapi/orderedmap"
	"github.com/pkg-base/yaml"
	"github.com/stretchr/testify/assert"
)

func TestSecurityRequirement_Build(t *testing.T) {
	yml := `something:
  - read:me
  - write:me`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n base.SecurityRequirement
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	assert.NoError(t, err)
	assert.Equal(t, 1, n.Requirements.Value.Len())
	assert.Equal(t, "read:me", n.FindRequirement("something")[0].Value)
	assert.Equal(t, "write:me", n.FindRequirement("something")[1].Value)
	assert.Nil(t, n.FindRequirement("none"))
	assert.NotNil(t, n.GetContext())
	assert.NotNil(t, n.GetIndex())
}

func TestSecurityScheme_Build(t *testing.T) {
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

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.NotNil(t, n.GetRootNode())
	assert.Nil(t, n.GetKeyNode())

	assert.Equal(t, "45cf8d044a079a416a22ef0b1ff6947d0eca31ae39170a2493bae4d845df663b",
		low.GenerateHashString(&n))

	assert.Equal(t, "tea", n.Type.Value)
	assert.Equal(t, "cake", n.Description.Value)
	assert.Equal(t, "biscuit", n.Name.Value)
	assert.Equal(t, "jar", n.In.Value)
	assert.Equal(t, "lovely", n.Scheme.Value)
	assert.Equal(t, "wow", n.BearerFormat.Value)
	assert.Equal(t, "https://pb33f.io/openid", n.OpenIdConnectUrl.Value)

	var xMilk string
	_ = n.FindExtension("x-milk").Value.Decode(&xMilk)
	assert.Equal(t, "please", xMilk)
	assert.Equal(t, "https://pb33f.io", n.Flows.Value.Implicit.Value.TokenUrl.Value)
	assert.Equal(t, 1, orderedmap.Len(n.GetExtensions()))
	assert.NotNil(t, n.GetContext())
	assert.NotNil(t, n.GetIndex())
}

func TestSecurityScheme_Build_Fail(t *testing.T) {
	yml := `flows:
  $ref: #bork`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n SecurityScheme
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.Error(t, err)
}
