// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"gopkg.in/yaml.v3"
)

// OAuthFlows represents a low-level OpenAPI 3+ OAuthFlows object.
//  - https://spec.openapis.org/oas/v3.1.0#oauth-flows-object
type OAuthFlows struct {
	Implicit          low.NodeReference[*OAuthFlow]
	Password          low.NodeReference[*OAuthFlow]
	ClientCredentials low.NodeReference[*OAuthFlow]
	AuthorizationCode low.NodeReference[*OAuthFlow]
	Extensions        map[low.KeyReference[string]]low.ValueReference[any]
}

// FindExtension will attempt to locate an extension with the supplied name.
func (o *OAuthFlows) FindExtension(ext string) *low.ValueReference[any] {
	return low.FindItemInMap[any](ext, o.Extensions)
}

// Build will extract extensions and all OAuthFlow types from the supplied node.
func (o *OAuthFlows) Build(root *yaml.Node, idx *index.SpecIndex) error {
	o.Extensions = low.ExtractExtensions(root)

	v, vErr := low.ExtractObject[*OAuthFlow](ImplicitLabel, root, idx)
	if vErr != nil {
		return vErr
	}
	o.Implicit = v

	v, vErr = low.ExtractObject[*OAuthFlow](PasswordLabel, root, idx)
	if vErr != nil {
		return vErr
	}
	o.Password = v

	v, vErr = low.ExtractObject[*OAuthFlow](ClientCredentialsLabel, root, idx)
	if vErr != nil {
		return vErr
	}
	o.ClientCredentials = v

	v, vErr = low.ExtractObject[*OAuthFlow](AuthorizationCodeLabel, root, idx)
	if vErr != nil {
		return vErr
	}
	o.AuthorizationCode = v
	return nil

}

// OAuthFlow represents a low-level OpenAPI 3+ OAuthFlow object.
//  - https://spec.openapis.org/oas/v3.1.0#oauth-flow-object
type OAuthFlow struct {
	AuthorizationUrl low.NodeReference[string]
	TokenUrl         low.NodeReference[string]
	RefreshUrl       low.NodeReference[string]
	Scopes           low.KeyReference[map[low.KeyReference[string]]low.ValueReference[string]]
	Extensions       map[low.KeyReference[string]]low.ValueReference[any]
}

// FindScope attempts to locate a scope using a specified name.
func (o *OAuthFlow) FindScope(scope string) *low.ValueReference[string] {
	return low.FindItemInMap[string](scope, o.Scopes.Value)
}

// FindExtension attempts to locate an extension with a specified key
func (o *OAuthFlow) FindExtension(ext string) *low.ValueReference[any] {
	return low.FindItemInMap[any](ext, o.Extensions)
}

// Build will extract extensions from the node.
func (o *OAuthFlow) Build(root *yaml.Node, idx *index.SpecIndex) error {
	o.Extensions = low.ExtractExtensions(root)
	return nil
}
