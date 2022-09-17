// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	low "github.com/pb33f/libopenapi/datamodel/low/v3"
)

// OAuthFlows represents a high-level OpenAPI 3+ OAuthFlows object that is backed by a low-level one.
//  - https://spec.openapis.org/oas/v3.1.0#oauth-flows-object
type OAuthFlows struct {
	Implicit          *OAuthFlow
	Password          *OAuthFlow
	ClientCredentials *OAuthFlow
	AuthorizationCode *OAuthFlow
	Extensions        map[string]any
	low               *low.OAuthFlows
}

// NewOAuthFlows creates a new high-level OAuthFlows instance from a low-level one.
func NewOAuthFlows(flows *low.OAuthFlows) *OAuthFlows {
	o := new(OAuthFlows)
	o.low = flows
	if !flows.Implicit.IsEmpty() {
		o.Implicit = NewOAuthFlow(flows.Implicit.Value)
	}
	if !flows.Password.IsEmpty() {
		o.Password = NewOAuthFlow(flows.Password.Value)
	}
	if !flows.ClientCredentials.IsEmpty() {
		o.ClientCredentials = NewOAuthFlow(flows.ClientCredentials.Value)
	}
	if !flows.AuthorizationCode.IsEmpty() {
		o.AuthorizationCode = NewOAuthFlow(flows.AuthorizationCode.Value)
	}
	if !flows.Implicit.IsEmpty() {
		o.Implicit = NewOAuthFlow(flows.Implicit.Value)
	}
	o.Extensions = high.ExtractExtensions(flows.Extensions)
	return o
}

// GoLow returns the low-level OAuthFlows instance used to create the high-level one.
func (o *OAuthFlows) GoLow() *low.OAuthFlows {
	return o.low
}
