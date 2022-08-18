// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import low "github.com/pb33f/libopenapi/datamodel/low/3.0"

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

func (s *SecurityScheme) GoLow() *low.SecurityScheme {
	return s.low
}
