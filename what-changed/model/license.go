// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/datamodel/low/v3"
)

// LicenseChanges represent changes to a License object that is a child of Info object. Part of an OpenAPI document
type LicenseChanges struct {
	*PropertyChanges
}

// GetAllChanges returns a slice of all changes made between License objects
func (l *LicenseChanges) GetAllChanges() []*Change {
	return l.Changes
}

// TotalChanges represents the total number of changes made to a License instance.
func (l *LicenseChanges) TotalChanges() int {
	return l.PropertyChanges.TotalChanges()
}

// TotalBreakingChanges always returns 0 for License objects, they are non-binding.
func (l *LicenseChanges) TotalBreakingChanges() int {
	return 0
}

// CompareLicense will check a left (original) and right (new) License object for any changes. If there
// were any, a pointer to a LicenseChanges object is returned, otherwise if nothing changed - the function
// returns nil.
func CompareLicense(l, r *base.License) *LicenseChanges {

	var changes []*Change
	var props []*PropertyCheck

	// check URL
	props = append(props, &PropertyCheck{
		LeftNode:  l.URL.ValueNode,
		RightNode: r.URL.ValueNode,
		Label:     v3.URLLabel,
		Changes:   &changes,
		Breaking:  false,
		Original:  l,
		New:       r,
	})

	// check name
	props = append(props, &PropertyCheck{
		LeftNode:  l.Name.ValueNode,
		RightNode: r.Name.ValueNode,
		Label:     v3.NameLabel,
		Changes:   &changes,
		Breaking:  false,
		Original:  l,
		New:       r,
	})

	// check everything.
	CheckProperties(props)

	lc := new(LicenseChanges)
	lc.PropertyChanges = NewPropertyChanges(changes)
	if lc.TotalChanges() <= 0 {
		return nil
	}
	return lc
}
