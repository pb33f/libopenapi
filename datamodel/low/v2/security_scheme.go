// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"crypto/sha256"
	"fmt"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"gopkg.in/yaml.v3"
	"sort"
	"strings"
)

// SecurityScheme is a low-level representation of a Swagger / OpenAPI 2 SecurityScheme object.
//
// SecurityScheme allows the definition of a security scheme that can be used by the operations. Supported schemes are
// basic authentication, an API key (either as a header or as a query parameter) and OAuth2's common flows
// (implicit, password, application and access code)
//  - https://swagger.io/specification/v2/#securityDefinitionsObject
type SecurityScheme struct {
	Type             low.NodeReference[string]
	Description      low.NodeReference[string]
	Name             low.NodeReference[string]
	In               low.NodeReference[string]
	Flow             low.NodeReference[string]
	AuthorizationUrl low.NodeReference[string]
	TokenUrl         low.NodeReference[string]
	Scopes           low.NodeReference[*Scopes]
	Extensions       map[low.KeyReference[string]]low.ValueReference[any]
}

// GetExtensions returns all SecurityScheme extensions and satisfies the low.HasExtensions interface.
func (ss *SecurityScheme) GetExtensions() map[low.KeyReference[string]]low.ValueReference[any] {
	return ss.Extensions
}

// Build will extract extensions and scopes from the node.
func (ss *SecurityScheme) Build(root *yaml.Node, idx *index.SpecIndex) error {
	ss.Extensions = low.ExtractExtensions(root)

	scopes, sErr := low.ExtractObject[*Scopes](ScopesLabel, root, idx)
	if sErr != nil {
		return sErr
	}
	ss.Scopes = scopes
	return nil
}

// Hash will return a consistent SHA256 Hash of the SecurityScheme object
func (ss *SecurityScheme) Hash() [32]byte {
	var f []string
	if !ss.Type.IsEmpty() {
		f = append(f, ss.Type.Value)
	}
	if !ss.Description.IsEmpty() {
		f = append(f, ss.Description.Value)
	}
	if !ss.Name.IsEmpty() {
		f = append(f, ss.Name.Value)
	}
	if !ss.In.IsEmpty() {
		f = append(f, ss.In.Value)
	}
	if !ss.Flow.IsEmpty() {
		f = append(f, ss.Flow.Value)
	}
	if !ss.AuthorizationUrl.IsEmpty() {
		f = append(f, ss.AuthorizationUrl.Value)
	}
	if !ss.TokenUrl.IsEmpty() {
		f = append(f, ss.TokenUrl.Value)
	}
	if !ss.Scopes.IsEmpty() {
		f = append(f, low.GenerateHashString(ss.Scopes.Value))
	}
	keys := make([]string, len(ss.Extensions))
	z := 0
	for k := range ss.Extensions {
		keys[z] = fmt.Sprintf("%s-%x", k.Value, sha256.Sum256([]byte(fmt.Sprint(ss.Extensions[k].Value))))
		z++
	}
	sort.Strings(keys)
	f = append(f, keys...)
	return sha256.Sum256([]byte(strings.Join(f, "|")))
}
