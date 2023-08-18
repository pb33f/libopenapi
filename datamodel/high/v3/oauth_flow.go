// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	low "github.com/pb33f/libopenapi/datamodel/low/v3"
	"gopkg.in/yaml.v3"
)

// OAuthFlow represents a high-level OpenAPI 3+ OAuthFlow object that is backed by a low-level one.
//   - https://spec.openapis.org/oas/v3.1.0#oauth-flow-object
type OAuthFlow struct {
	AuthorizationUrl string            `json:"authorizationUrl,omitempty" yaml:"authorizationUrl,omitempty"`
	TokenUrl         string            `json:"tokenUrl,omitempty" yaml:"tokenUrl,omitempty"`
	RefreshUrl       string            `json:"refreshUrl,omitempty" yaml:"refreshUrl,omitempty"`
	Scopes           map[string]string `json:"scopes,omitempty" yaml:"scopes,omitempty"`
	Extensions       map[string]any    `json:"-" yaml:"-"`
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

// GoLowUntyped will return the low-level Discriminator instance that was used to create the high-level one, with no type
func (o *OAuthFlow) GoLowUntyped() any {
	return o.low
}

// Render will return a YAML representation of the OAuthFlow object as a byte slice.
func (o *OAuthFlow) Render() ([]byte, error) {
	return yaml.Marshal(o)
}

// MarshalYAML will create a ready to render YAML representation of the OAuthFlow object.
func (o *OAuthFlow) MarshalYAML() (interface{}, error) {
	nb := high.NewNodeBuilder(o, o.low)
	return nb.Render(), nil
}
