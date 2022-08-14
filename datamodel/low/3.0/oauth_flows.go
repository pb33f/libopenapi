// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"gopkg.in/yaml.v3"
)

const (
	ImplicitLabel          = "implicit"
	PasswordLabel          = "password"
	ClientCredentialsLabel = "clientCredentials"
	AuthorizationCodeLabel = "authorizationCode"
)

type OAuthFlows struct {
	Implicit          low.NodeReference[*OAuthFlow]
	Password          low.NodeReference[*OAuthFlow]
	ClientCredentials low.NodeReference[*OAuthFlow]
	AuthorizationCode low.NodeReference[*OAuthFlow]
	Extensions        map[low.KeyReference[string]]low.ValueReference[any]
}

func (o *OAuthFlows) FindExtension(ext string) *low.ValueReference[any] {
	return low.FindItemInMap[any](ext, o.Extensions)
}

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

type OAuthFlow struct {
	AuthorizationUrl low.NodeReference[string]
	TokenUrl         low.NodeReference[string]
	RefreshUrl       low.NodeReference[string]
	Scopes           low.KeyReference[map[low.KeyReference[string]]low.ValueReference[string]]
	Extensions       map[low.KeyReference[string]]low.ValueReference[any]
}

func (o *OAuthFlow) FindScope(scope string) *low.ValueReference[string] {
	return low.FindItemInMap[string](scope, o.Scopes.Value)
}

func (o *OAuthFlow) FindExtension(ext string) *low.ValueReference[any] {
	return low.FindItemInMap[any](ext, o.Extensions)
}

func (o *OAuthFlow) Build(root *yaml.Node, idx *index.SpecIndex) error {
	o.Extensions = low.ExtractExtensions(root)
	return nil
}
