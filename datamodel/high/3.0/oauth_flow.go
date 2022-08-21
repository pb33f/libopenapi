// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	low "github.com/pb33f/libopenapi/datamodel/low/3.0"
)

type OAuthFlow struct {
	AuthorizationUrl string
	TokenUrl         string
	RefreshUrl       string
	Scopes           map[string]string
	Extensions       map[string]any
	low              *low.OAuthFlow
}

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

func (o *OAuthFlow) GoLow() *low.OAuthFlow {
	return o.low
}
