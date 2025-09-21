// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestNewOAuthFlows_WithDevice(t *testing.T) {
	// Test for line 42: Device flow support in OpenAPI 3.2+
	yml := `implicit:
    authorizationUrl: https://pb33f.io/oauth/implicit
    scopes:
        write:burgers: modify and add new burgers implicitly
        read:burgers: read all burgers
device:
    tokenUrl: https://pb33f.io/oauth/device/token
    scopes:
        write:burgers: modify burgers using device flow
        read:burgers: read all burgers with device`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n v3.OAuthFlows
	_ = low.BuildModel(&idxNode, &n)
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	r := NewOAuthFlows(&n)

	// Test that device flow was parsed
	assert.NotNil(t, r.Device)
	assert.Equal(t, "https://pb33f.io/oauth/device/token", r.Device.TokenUrl)
	assert.NotNil(t, r.Device.Scopes)
	assert.Equal(t, 2, r.Device.Scopes.Len())
}

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
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	r := NewOAuthFlows(&n)

	assert.Equal(t, 2, r.Implicit.Scopes.Len())
	assert.Equal(t, 2, r.AuthorizationCode.Scopes.Len())
	assert.Equal(t, 2, r.Password.Scopes.Len())
	assert.Equal(t, 2, r.ClientCredentials.Scopes.Len())
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
	r.ClientCredentials.Scopes.Set("CHIP:CHOP", "microwave a sock")
	rBytes, _ = r.Render()
	assert.Equal(t, modified, strings.TrimSpace(string(rBytes)))
}
