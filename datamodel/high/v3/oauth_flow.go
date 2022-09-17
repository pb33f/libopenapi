// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	low "github.com/pb33f/libopenapi/datamodel/low/v3"
)

// OAuthFlow represents a high-level OpenAPI 3+ OAuthFlow object that is backed by a low-level one.
//  - https://spec.openapis.org/oas/v3.1.0#oauth-flow-object
type OAuthFlow struct {
	AuthorizationUrl string
	TokenUrl         string
	RefreshUrl       string
	Scopes           map[string]string
	Extensions       map[string]any
	low              *low.OAuthFlow
}

// NewOAuthFlow creates a new high-level OAuthFlow instance from a low-level one.
func NewOAuthFlow(flow *low.OAuthFlow) *OAuthFlow {
	o := new(OAuthFlow)
	o.low = flow
	o.TokenUrl = flow.TokenUrl.Value
	o.AuthorizationUrl = flow.AuthorizationUrl.Value
	o.RefreshUrl = flow.RefreshUrl.Value
	scopes := make(map[string]string)
	for k, v := range flow.Scopes.Value {
		scopes[k.Value] = v.Value
	}
	o.Scopes = scopes
	o.Extensions = high.ExtractExtensions(flow.Extensions)
	return o
}

// GoLow returns the low-level OAuthFlow instance used to create the high-level one.
func (o *OAuthFlow) GoLow() *low.OAuthFlow {
	return o.low
}
