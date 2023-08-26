// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestOAuthFlows_Build(t *testing.T) {
	t.Parallel()

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
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(nil, idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "herbs", n.FindExtension("x-tasty").Value)
	assert.Equal(t, "https://pb33f.io/auth", n.AuthorizationUrl.Value)
	assert.Equal(t, "https://pb33f.io/token", n.TokenUrl.Value)
	assert.Equal(t, "https://pb33f.io/refresh", n.RefreshUrl.Value)
	assert.Equal(t, "vanilla", n.FindScope("fresh:cake").Value)
	assert.Len(t, n.GetExtensions(), 1)
}

func TestOAuthFlow_Build_Implicit(t *testing.T) {
	t.Parallel()

	yml := `implicit:
  authorizationUrl: https://pb33f.io/auth
x-tasty: herbs`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n OAuthFlows
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(nil, idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "herbs", n.FindExtension("x-tasty").Value)
	assert.Equal(t, "https://pb33f.io/auth", n.Implicit.Value.AuthorizationUrl.Value)
	assert.Len(t, n.GetExtensions(), 1)
}

func TestOAuthFlow_Build_Implicit_Fail(t *testing.T) {
	t.Parallel()

	yml := `implicit:
  $ref: #bork"`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n OAuthFlows
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(nil, idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestOAuthFlow_Build_Password(t *testing.T) {
	t.Parallel()

	yml := `password:
  authorizationUrl: https://pb33f.io/auth`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n OAuthFlows
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(nil, idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "https://pb33f.io/auth", n.Password.Value.AuthorizationUrl.Value)
}

func TestOAuthFlow_Build_Password_Fail(t *testing.T) {
	t.Parallel()

	yml := `password:
  $ref: #bork"`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n OAuthFlows
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(nil, idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestOAuthFlow_Build_ClientCredentials(t *testing.T) {
	t.Parallel()

	yml := `clientCredentials:
  authorizationUrl: https://pb33f.io/auth`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n OAuthFlows
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(nil, idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "https://pb33f.io/auth", n.ClientCredentials.Value.AuthorizationUrl.Value)
}

func TestOAuthFlow_Build_ClientCredentials_Fail(t *testing.T) {
	t.Parallel()

	yml := `clientCredentials:
  $ref: #bork"`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n OAuthFlows
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(nil, idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestOAuthFlow_Build_AuthCode(t *testing.T) {
	t.Parallel()

	yml := `authorizationCode:
  authorizationUrl: https://pb33f.io/auth`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n OAuthFlows
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(nil, idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, "https://pb33f.io/auth", n.AuthorizationCode.Value.AuthorizationUrl.Value)
}

func TestOAuthFlow_Build_AuthCode_Fail(t *testing.T) {
	t.Parallel()

	yml := `authorizationCode:
  $ref: #bork"`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n OAuthFlows
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(nil, idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestOAuthFlow_Hash(t *testing.T) {
	t.Parallel()

	yml := `authorizationUrl: https://pb33f.io/auth
tokenUrl: https://pb33f.io/token
refreshUrl: https://pb33f.io/refresh
scopes:
  smoke: weed
x-sleepy: tired`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n OAuthFlow
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(nil, idxNode.Content[0], idx)

	yml2 := `refreshUrl: https://pb33f.io/refresh
tokenUrl: https://pb33f.io/token
authorizationUrl: https://pb33f.io/auth
x-sleepy: tired
scopes:
  smoke: weed`

	var idxNode2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml2), &idxNode2)
	idx2 := index.NewSpecIndex(&idxNode2)

	var n2 OAuthFlow
	_ = low.BuildModel(idxNode2.Content[0], &n2)
	_ = n2.Build(nil, idxNode2.Content[0], idx2)

	// hash
	assert.Equal(t, n.Hash(), n2.Hash())

}

func TestOAuthFlows_Hash(t *testing.T) {
	t.Parallel()

	yml := `implicit:
  authorizationUrl: https://pb33f.io/auth
password:
  authorizationUrl: https://pb33f.io/auth
clientCredentials:
  authorizationUrl: https://pb33f.io/auth
authorizationCode:
  authorizationUrl: https://pb33f.io/auth
x-code: cody
`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n OAuthFlows
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(nil, idxNode.Content[0], idx)

	yml2 := `authorizationCode:
  authorizationUrl: https://pb33f.io/auth
clientCredentials:
  authorizationUrl: https://pb33f.io/auth
x-code: cody
implicit:
  authorizationUrl: https://pb33f.io/auth
password:
  authorizationUrl: https://pb33f.io/auth
`

	var idxNode2 yaml.Node
	_ = yaml.Unmarshal([]byte(yml2), &idxNode2)
	idx2 := index.NewSpecIndex(&idxNode2)

	var n2 OAuthFlows
	_ = low.BuildModel(idxNode2.Content[0], &n2)
	_ = n2.Build(nil, idxNode2.Content[0], idx2)

	// hash
	assert.Equal(t, n.Hash(), n2.Hash())
}
