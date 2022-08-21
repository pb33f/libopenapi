// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	low "github.com/pb33f/libopenapi/datamodel/low/3.0"
)

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

func (s *SecurityScheme) GoLow() *low.SecurityScheme {
	return s.low
}
