// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	low "github.com/pb33f/libopenapi/datamodel/low/v2"
)

// Scopes is a high-level representation of a Swagger / OpenAPI 2 OAuth2 Scopes object, that is backed by a low-level one.
//
// Scopes lists the available scopes for an OAuth2 security scheme.
//   - https://swagger.io/specification/v2/#scopesObject
type Scopes struct {
	Values map[string]string
	low    *low.Scopes
}

// NewScopes creates a new high-level instance of Scopes from a low-level one.
func NewScopes(scopes *low.Scopes) *Scopes {
	s := new(Scopes)
	s.low = scopes
	scopeValues := make(map[string]string)
	for k := range scopes.Values {
		scopeValues[k.Value] = scopes.Values[k].Value
	}
	s.Values = scopeValues
	return s
}

// GoLow returns the low-level instance of Scopes used to create the high-level one.
func (s *Scopes) GoLow() *low.Scopes {
	return s.low
}
