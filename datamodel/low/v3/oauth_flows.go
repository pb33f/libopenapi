// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"crypto/sha256"
	"fmt"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"gopkg.in/yaml.v3"
	"sort"
	"strings"
)

// OAuthFlows represents a low-level OpenAPI 3+ OAuthFlows object.
//  - https://spec.openapis.org/oas/v3.1.0#oauth-flows-object
type OAuthFlows struct {
	Implicit          low.NodeReference[*OAuthFlow]
	Password          low.NodeReference[*OAuthFlow]
	ClientCredentials low.NodeReference[*OAuthFlow]
	AuthorizationCode low.NodeReference[*OAuthFlow]
	Extensions        map[low.KeyReference[string]]low.ValueReference[any]
	*low.Reference
}

// GetExtensions returns all OAuthFlows extensions and satisfies the low.HasExtensions interface.
func (o *OAuthFlows) GetExtensions() map[low.KeyReference[string]]low.ValueReference[any] {
	return o.Extensions
}

// FindExtension will attempt to locate an extension with the supplied name.
func (o *OAuthFlows) FindExtension(ext string) *low.ValueReference[any] {
	return low.FindItemInMap[any](ext, o.Extensions)
}

// Build will extract extensions and all OAuthFlow types from the supplied node.
func (o *OAuthFlows) Build(root *yaml.Node, idx *index.SpecIndex) error {
	o.Reference = new(low.Reference)
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

// Hash will return a consistent SHA256 Hash of the OAuthFlow object
func (o *OAuthFlows) Hash() [32]byte {
	var f []string
	if !o.Implicit.IsEmpty() {
		f = append(f, low.GenerateHashString(o.Implicit.Value))
	}
	if !o.Password.IsEmpty() {
		f = append(f, low.GenerateHashString(o.Password.Value))
	}
	if !o.ClientCredentials.IsEmpty() {
		f = append(f, low.GenerateHashString(o.ClientCredentials.Value))
	}
	if !o.AuthorizationCode.IsEmpty() {
		f = append(f, low.GenerateHashString(o.AuthorizationCode.Value))
	}
	for k := range o.Extensions {
		f = append(f, fmt.Sprintf("%s-%v", k.Value, o.Extensions[k].Value))
	}
	return sha256.Sum256([]byte(strings.Join(f, "|")))
}

// OAuthFlow represents a low-level OpenAPI 3+ OAuthFlow object.
//  - https://spec.openapis.org/oas/v3.1.0#oauth-flow-object
type OAuthFlow struct {
	AuthorizationUrl low.NodeReference[string]
	TokenUrl         low.NodeReference[string]
	RefreshUrl       low.NodeReference[string]
	Scopes           low.NodeReference[map[low.KeyReference[string]]low.ValueReference[string]]
	Extensions       map[low.KeyReference[string]]low.ValueReference[any]
	*low.Reference
}

// GetExtensions returns all OAuthFlow extensions and satisfies the low.HasExtensions interface.
func (o *OAuthFlow) GetExtensions() map[low.KeyReference[string]]low.ValueReference[any] {
	return o.Extensions
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
	o.Reference = new(low.Reference)
	o.Extensions = low.ExtractExtensions(root)
	return nil
}

// Hash will return a consistent SHA256 Hash of the OAuthFlow object
func (o *OAuthFlow) Hash() [32]byte {
	var f []string
	if !o.AuthorizationUrl.IsEmpty() {
		f = append(f, o.AuthorizationUrl.Value)
	}
	if !o.TokenUrl.IsEmpty() {
		f = append(f, o.TokenUrl.Value)
	}
	if !o.RefreshUrl.IsEmpty() {
		f = append(f, o.RefreshUrl.Value)
	}
	keys := make([]string, len(o.Scopes.Value))
	z := 0
	for k := range o.Scopes.Value {
		keys[z] = fmt.Sprintf("%s-%s", k.Value, sha256.Sum256([]byte(fmt.Sprint(o.Scopes.Value[k].Value))))
		z++
	}
	sort.Strings(keys)
	f = append(f, keys...)
	keys = make([]string, len(o.Extensions))
	z = 0
	for k := range o.Extensions {
		keys[z] = fmt.Sprintf("%s-%x", k.Value, sha256.Sum256([]byte(fmt.Sprint(o.Extensions[k].Value))))
		z++
	}
	sort.Strings(keys)
	f = append(f, keys...)
	return sha256.Sum256([]byte(strings.Join(f, "|")))
}
