// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestOAuthFlows_Build(t *testing.T) {

	yml := `authorizationUrl: https://pb33f.io/auth
tokenUrl: https://pb33f.io/token
refreshUrl: https://pb33f.io/refresh
scopes:
  fresh:cake: vanilla
  cold:beer: yummy
x-tasty: herbs
`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n OAuthFlow
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "herbs", n.FindExtension("x-tasty").Value)
	assert.Equal(t, "https://pb33f.io/auth", n.AuthorizationUrl.Value)
	assert.Equal(t, "https://pb33f.io/token", n.TokenUrl.Value)
	assert.Equal(t, "https://pb33f.io/refresh", n.RefreshUrl.Value)
	assert.Equal(t, "vanilla", n.FindScope("fresh:cake").Value)
}

func TestOAuthFlow_Build_Implicit(t *testing.T) {

	yml := `implicit:
  authorizationUrl: https://pb33f.io/auth
x-tasty: herbs`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n OAuthFlows
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "herbs", n.FindExtension("x-tasty").Value)
	assert.Equal(t, "https://pb33f.io/auth", n.Implicit.Value.AuthorizationUrl.Value)
}

func TestOAuthFlow_Build_Implicit_Fail(t *testing.T) {

	yml := `implicit:
  $ref: #bork"`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n OAuthFlows
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestOAuthFlow_Build_Password(t *testing.T) {

	yml := `password:
  authorizationUrl: https://pb33f.io/auth`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n OAuthFlows
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "https://pb33f.io/auth", n.Password.Value.AuthorizationUrl.Value)
}

func TestOAuthFlow_Build_Password_Fail(t *testing.T) {

	yml := `password:
  $ref: #bork"`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n OAuthFlows
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestOAuthFlow_Build_ClientCredentials(t *testing.T) {

	yml := `clientCredentials:
  authorizationUrl: https://pb33f.io/auth`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n OAuthFlows
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "https://pb33f.io/auth", n.ClientCredentials.Value.AuthorizationUrl.Value)
}

func TestOAuthFlow_Build_ClientCredentials_Fail(t *testing.T) {

	yml := `clientCredentials:
  $ref: #bork"`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n OAuthFlows
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestOAuthFlow_Build_AuthCode(t *testing.T) {

	yml := `authorizationCode:
  authorizationUrl: https://pb33f.io/auth`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n OAuthFlows
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "https://pb33f.io/auth", n.AuthorizationCode.Value.AuthorizationUrl.Value)
}

func TestOAuthFlow_Build_AuthCode_Fail(t *testing.T) {

	yml := `authorizationCode:
  $ref: #bork"`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n OAuthFlows
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.Error(t, err)
}
