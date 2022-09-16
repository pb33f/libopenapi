// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	low "github.com/pb33f/libopenapi/datamodel/low/v2"
)

type Scopes struct {
	Values map[string]string
	low    *low.Scopes
}

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

func (s *Scopes) GoLow() *low.Scopes {
	return s.low
}
