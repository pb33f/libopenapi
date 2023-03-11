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

// SecurityScheme represents a low-level OpenAPI 3+ SecurityScheme object.
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
	Type             low.NodeReference[string]
	Description      low.NodeReference[string]
	Name             low.NodeReference[string]
	In               low.NodeReference[string]
	Scheme           low.NodeReference[string]
	BearerFormat     low.NodeReference[string]
	Flows            low.NodeReference[*OAuthFlows]
	OpenIdConnectUrl low.NodeReference[string]
	Extensions       map[low.KeyReference[string]]low.ValueReference[any]
	*low.Reference
}

// FindExtension attempts to locate an extension using the supplied key.
func (ss *SecurityScheme) FindExtension(ext string) *low.ValueReference[any] {
	return low.FindItemInMap[any](ext, ss.Extensions)
}

// GetExtensions returns all SecurityScheme extensions and satisfies the low.HasExtensions interface.
func (ss *SecurityScheme) GetExtensions() map[low.KeyReference[string]]low.ValueReference[any] {
	return ss.Extensions
}

// Build will extract OAuthFlows and extensions from the node.
func (ss *SecurityScheme) Build(root *yaml.Node, idx *index.SpecIndex) error {
	ss.Reference = new(low.Reference)
	ss.Extensions = low.ExtractExtensions(root)

	oa, oaErr := low.ExtractObject[*OAuthFlows](OAuthFlowsLabel, root, idx)
	if oaErr != nil {
		return oaErr
	}
	if oa.Value != nil {
		ss.Flows = oa
	}
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
	if !ss.Scheme.IsEmpty() {
		f = append(f, ss.Scheme.Value)
	}
	if !ss.BearerFormat.IsEmpty() {
		f = append(f, ss.BearerFormat.Value)
	}
	if !ss.Flows.IsEmpty() {
		f = append(f, low.GenerateHashString(ss.Flows.Value))
	}
	if !ss.OpenIdConnectUrl.IsEmpty() {
		f = append(f, ss.OpenIdConnectUrl.Value)
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
