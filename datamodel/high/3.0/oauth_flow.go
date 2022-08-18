// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import low "github.com/pb33f/libopenapi/datamodel/low/3.0"

type OAuthFlow struct {
	AuthorizationUrl string
	TokenUrl         string
	RefreshUrl       string
	Scopes           map[string]string
	Extensions       map[string]any
	low              *low.OAuthFlow
}

func (o *OAuthFlow) GoLow() *low.OAuthFlow {
	return o.low
}
