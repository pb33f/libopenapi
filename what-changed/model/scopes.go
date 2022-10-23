// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/v2"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
)

type ScopesChanges struct {
	PropertyChanges
	ExtensionChanges *ExtensionChanges
}

func (s *ScopesChanges) TotalChanges() int {
	c := s.PropertyChanges.TotalChanges()
	if s.ExtensionChanges != nil {
		c += s.ExtensionChanges.TotalChanges()
	}
	return c
}

func (s *ScopesChanges) TotalBreakingChanges() int {
	return s.PropertyChanges.TotalBreakingChanges()
}

func CompareScopes(l, r *v2.Scopes) *ScopesChanges {
	if low.AreEqual(l, r) {
		return nil
	}
	var changes []*Change
	for v := range l.Values {
		if r != nil && r.FindScope(v.Value) == nil {
			CreateChange(&changes, ObjectRemoved, v3.Scopes,
				l.Values[v].ValueNode, nil, true,
				v.Value, nil)
			continue
		}
		if r != nil && r.FindScope(v.Value) != nil {
			if l.Values[v].Value != r.FindScope(v.Value).Value {
				CreateChange(&changes, Modified, v3.Scopes,
					l.Values[v].ValueNode, r.FindScope(v.Value).ValueNode, true,
					l.Values[v].Value, r.FindScope(v.Value).Value)
			}
		}
	}
	for v := range r.Values {
		if l != nil && l.FindScope(v.Value) == nil {
			CreateChange(&changes, ObjectAdded, v3.Scopes,
				nil, r.Values[v].ValueNode, false,
				nil, v.Value)
		}
	}

	sc := new(ScopesChanges)
	sc.Changes = changes
	sc.ExtensionChanges = CompareExtensions(l.Extensions, r.Extensions)
	return sc
}
