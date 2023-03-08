// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"strings"
	"testing"
)

func TestNewOAuthFlows(t *testing.T) {

	yml := `implicit:
    authorizationUrl: https://pb33f.io/oauth/implicit
    scopes:
        write:burgers: modify and add new burgers implicitly
        read:burgers: read all burgers
authorizationCode:
    authorizationUrl: https://pb33f.io/oauth/authCode
    tokenUrl: https://api.pb33f.io/oauth/token
    scopes:
        write:burgers: modify burgers and stuff with a code
        read:burgers: read all the burgers
password:
    authorizationUrl: https://pb33f.io/oauth/password
    scopes:
        write:burgers: modify and add new burgers with a password
        read:burgers: read all burgers
clientCredentials:
    authorizationUrl: https://pb33f.io/oauth/clientCreds
    scopes:
        write:burgers: modify burgers and stuff with creds
        read:burgers: read all the burgers`

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

	// now render it back out, and it should be identical!
	rBytes, _ := r.Render()
	assert.Equal(t, yml, strings.TrimSpace(string(rBytes)))

	modified := `implicit:
    authorizationUrl: https://pb33f.io/oauth/implicit
    scopes:
        write:burgers: modify and add new burgers implicitly
        read:burgers: read all burgers
authorizationCode:
    authorizationUrl: https://pb33f.io/oauth/authCode
    tokenUrl: https://api.pb33f.io/oauth/token
    scopes:
        write:burgers: modify burgers and stuff with a code
        read:burgers: read all the burgers
password:
    authorizationUrl: https://pb33f.io/oauth/password
    scopes:
        write:burgers: modify and add new burgers with a password
        read:burgers: read all burgers
clientCredentials:
    authorizationUrl: https://pb33f.io/oauth/clientCreds
    scopes:
        write:burgers: modify burgers and stuff with creds
        read:burgers: read all the burgers
        CHIP:CHOP: microwave a sock`

	// now modify it and render it back out, and it should be identical!
	r.ClientCredentials.Scopes["CHIP:CHOP"] = "microwave a sock"
	rBytes, _ = r.Render()
	assert.Equal(t, modified, strings.TrimSpace(string(rBytes)))

}
