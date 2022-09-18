// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	low "github.com/pb33f/libopenapi/datamodel/low/v3"
)

// SecurityScheme represents a high-level OpenAPI 3+ SecurityScheme object that is backed by a low-level one.
//
// Defines a security scheme that can be used by the operations.
//
// Supported schemes are HTTP authentication, an API key (either as a header, a cookie parameter or as a query parameter),
// mutual TLS (use of a client certificate), OAuth2’s common flows (implicit, password, client credentials and
// authorization code) as defined in RFC6749 (https://www.rfc-editor.org/rfc/rfc6749), and OpenID Connect Discovery.
// Please note that as of 2020, the implicit  flow is about to be deprecated by OAuth 2.0 Security Best Current Practice.
// Recommended for most use case is Authorization Code Grant flow with PKCE.
//  - https://spec.openapis.org/oas/v3.1.0#security-scheme-object
type SecurityScheme struct {
	Type             string
	Description      string
	Name             string
	In               string
	Scheme           string
	BearerFormat     string
	Flows            *OAuthFlows
	OpenIdConnectUrl string
	Extensions       map[string]any
	low              *low.SecurityScheme
}

// NewSecurityScheme creates a new high-level SecurityScheme from a low-level one.
func NewSecurityScheme(ss *low.SecurityScheme) *SecurityScheme {
	s := new(SecurityScheme)
	s.low = ss
	s.Type = ss.Type.Value
	s.Description = ss.Description.Value
	s.Name = ss.Name.Value
	s.Scheme = ss.Scheme.Value
	s.In = ss.In.Value
	s.BearerFormat = ss.BearerFormat.Value
	s.OpenIdConnectUrl = ss.OpenIdConnectUrl.Value
	s.Extensions = high.ExtractExtensions(ss.Extensions)
	if !ss.Flows.IsEmpty() {
		s.Flows = NewOAuthFlows(ss.Flows.Value)
	}
	return s
}

// GoLow returns the low-level SecurityScheme that was used to create the high-level one.
func (s *SecurityScheme) GoLow() *low.SecurityScheme {
	return s.low
}
