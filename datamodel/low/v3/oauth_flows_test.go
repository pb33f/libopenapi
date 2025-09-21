// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
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
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)

	assert.NotNil(t, n.GetRootNode())

	var xTasty string
	_ = n.FindExtension("x-tasty").Value.Decode(&xTasty)
	assert.Equal(t, "herbs", xTasty)
	assert.Equal(t, "https://pb33f.io/auth", n.AuthorizationUrl.Value)
	assert.Equal(t, "https://pb33f.io/token", n.TokenUrl.Value)
	assert.Equal(t, "https://pb33f.io/refresh", n.RefreshUrl.Value)
	assert.Equal(t, "vanilla", n.FindScope("fresh:cake").Value)
	assert.Equal(t, 1, orderedmap.Len(n.GetExtensions()))
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

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)

	assert.NotNil(t, n.GetRootNode())
	assert.Nil(t, n.GetKeyNode())

	var xTasty string
	_ = n.FindExtension("x-tasty").GetValue().Decode(&xTasty)
	assert.Equal(t, "herbs", xTasty)
	assert.Equal(t, "https://pb33f.io/auth", n.Implicit.Value.AuthorizationUrl.Value)
	assert.Equal(t, 1, orderedmap.Len(n.GetExtensions()))
	assert.NotNil(t, n.GetContext())
	assert.NotNil(t, n.GetIndex())
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

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
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

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
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

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
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

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
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

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
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

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
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

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.Error(t, err)
}

func TestOAuthFlow_Hash(t *testing.T) {
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
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

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
	_ = n2.Build(context.Background(), nil, idxNode2.Content[0], idx2)

	// hash
	assert.Equal(t, n.Hash(), n2.Hash())
	assert.NotNil(t, n2.GetContext())
	assert.NotNil(t, n2.GetIndex())
}

func TestOAuthFlows_DeviceFlow(t *testing.T) {
	yml := `device:
  tokenUrl: https://oauth2.example.com/device/token
  scopes:
    read: read access
    write: write access`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n OAuthFlows
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	assert.NotNil(t, n.Device.Value)
	assert.Equal(t, "https://oauth2.example.com/device/token", n.Device.Value.TokenUrl.Value)
	assert.Equal(t, 2, n.Device.Value.Scopes.Value.Len())
	assert.Equal(t, "read access", n.Device.Value.FindScope("read").Value)
	assert.Equal(t, "write access", n.Device.Value.FindScope("write").Value)

	// test hash includes device flow
	hash1 := n.Hash()
	if !n.Device.IsEmpty() {
		originalDevice := n.Device.Value
		n.Device = low.NodeReference[*OAuthFlow]{} // clear the reference
		hash2 := n.Hash()
		assert.NotEqual(t, hash1, hash2)
		n.Device.Value = originalDevice // restore
	}
}

func TestOAuthFlows_Hash(t *testing.T) {
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
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

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
	_ = n2.Build(context.Background(), nil, idxNode2.Content[0], idx2)

	// hash
	assert.Equal(t, n.Hash(), n2.Hash())
}

func TestOAuthFlow_Build_Device_Fail(t *testing.T) {
	yml := `device:
  $ref: #bork"`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n OAuthFlows
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.Error(t, err)
}
