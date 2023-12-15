// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	v2 "github.com/pb33f/libopenapi/datamodel/low/v2"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/orderedmap"
)

// ScopesChanges represents changes between two Swagger Scopes Objects
type ScopesChanges struct {
	*PropertyChanges
	ExtensionChanges *ExtensionChanges `json:"extensions,omitempty" yaml:"extensions,omitempty"`
}

// GetAllChanges returns a slice of all changes made between Scopes objects
func (s *ScopesChanges) GetAllChanges() []*Change {
	var changes []*Change
	changes = append(changes, s.Changes...)
	if s.ExtensionChanges != nil {
		changes = append(changes, s.ExtensionChanges.GetAllChanges()...)
	}
	return changes
}

// TotalChanges returns the total changes found between two Swagger Scopes objects.
func (s *ScopesChanges) TotalChanges() int {
	c := s.PropertyChanges.TotalChanges()
	if s.ExtensionChanges != nil {
		c += s.ExtensionChanges.TotalChanges()
	}
	return c
}

// TotalBreakingChanges returns the total number of breaking changes between two Swagger Scopes objects.
func (s *ScopesChanges) TotalBreakingChanges() int {
	return s.PropertyChanges.TotalBreakingChanges()
}

// CompareScopes compares a left and right Swagger Scopes objects for changes. If anything is found, returns
// a pointer to ScopesChanges, or returns nil if nothing is found.
func CompareScopes(l, r *v2.Scopes) *ScopesChanges {
	if low.AreEqual(l, r) {
		return nil
	}
	var changes []*Change
	for pair := orderedmap.First(l.Values); pair != nil; pair = pair.Next() {
		if r != nil && r.FindScope(pair.Key().Value) == nil {
			CreateChange(&changes, ObjectRemoved, v3.Scopes,
				pair.Value().ValueNode, nil, true,
				pair.Key().Value, nil)
			continue
		}
		if r != nil && r.FindScope(pair.Key().Value) != nil {
			if pair.Value().Value != r.FindScope(pair.Key().Value).Value {
				CreateChange(&changes, Modified, v3.Scopes,
					pair.Value().ValueNode, r.FindScope(pair.Key().Value).ValueNode, true,
					pair.Value().Value, r.FindScope(pair.Key().Value).Value)
			}
		}
	}
	for pair := orderedmap.First(r.Values); pair != nil; pair = pair.Next() {
		if l != nil && l.FindScope(pair.Key().Value) == nil {
			CreateChange(&changes, ObjectAdded, v3.Scopes,
				nil, pair.Value().ValueNode, false,
				nil, pair.Key().Value)
		}
	}

	sc := new(ScopesChanges)
	sc.PropertyChanges = NewPropertyChanges(changes)
	sc.ExtensionChanges = CompareExtensions(l.Extensions, r.Extensions)
	return sc
}
