// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	v3 "github.com/pb33f/libopenapi/datamodel/low/3.0"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestNewOAuthFlows(t *testing.T) {

	yml := `implicit:
  authorizationUrl: https://pb33f.io/oauth
  scopes:
    write:burgers: modify and add new burgers
    read:burgers: read all burgers
authorizationCode:
  authorizationUrl: https://pb33f.io/oauth
  tokenUrl: https://api.pb33f.io/oauth/token
  scopes:
    write:burgers: modify burgers and stuff
    read:burgers: read all the burgers
password:
  authorizationUrl: https://pb33f.io/oauth
  scopes:
    write:burgers: modify and add new burgers
    read:burgers: read all burgers
clientCredentials:
  authorizationUrl: https://pb33f.io/oauth
  scopes:
    write:burgers: modify burgers and stuff
    read:burgers: read all the burgers    `

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n v3.OAuthFlows
	_ = low.BuildModel(&idxNode, &n)
	_ = n.Build(idxNode.Content[0], idx)

	r := NewOAuthFlows(&n)

	assert.Len(t, r.Implicit.Scopes, 2)
	assert.Len(t, r.AuthorizationCode.Scopes, 2)
	assert.Len(t, r.Password.Scopes, 2)
	assert.Len(t, r.ClientCredentials.Scopes, 2)
	assert.Equal(t, 2, r.GoLow().Implicit.Value.AuthorizationUrl.KeyNode.Line)

}
