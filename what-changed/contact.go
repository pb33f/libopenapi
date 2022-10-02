// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package what_changed

import (
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	lowv3 "github.com/pb33f/libopenapi/datamodel/low/v3"
)

// ContactChanges Represent changes to a Contact object that is a child of Info, part of an OpenAPI document.
type ContactChanges struct {
	PropertyChanges[*lowbase.Contact]
}

// TotalChanges represents the total number of changes that have occurred to a Contact object
func (c *ContactChanges) TotalChanges() int {
	return len(c.Changes)
}

// CompareContact will check a left (original) and right (new) Contact object for any changes. If there
// were any, a pointer to a ContactChanges object is returned, otherwise if nothing changed - the function
// returns nil.
func CompareContact(l, r *lowbase.Contact) *ContactChanges {

	var changes []*Change[*lowbase.Contact]
	var props []*PropertyCheck[*lowbase.Contact]

	// check URL
	props = append(props, &PropertyCheck[*lowbase.Contact]{
		LeftNode:  l.URL.ValueNode,
		RightNode: r.URL.ValueNode,
		Label:     lowv3.URLLabel,
		Changes:   &changes,
		Breaking:  false,
		Original:  l,
		New:       r,
	})

	// check name
	props = append(props, &PropertyCheck[*lowbase.Contact]{
		LeftNode:  l.Name.ValueNode,
		RightNode: r.Name.ValueNode,
		Label:     lowv3.NameLabel,
		Changes:   &changes,
		Breaking:  false,
		Original:  l,
		New:       r,
	})

	// check email
	props = append(props, &PropertyCheck[*lowbase.Contact]{
		LeftNode:  l.Email.ValueNode,
		RightNode: r.Email.ValueNode,
		Label:     lowv3.EmailLabel,
		Changes:   &changes,
		Breaking:  false,
		Original:  l,
		New:       r,
	})

	// check everything.
	CheckProperties(props)

	dc := new(ContactChanges)
	dc.Changes = changes
	if len(changes) <= 0 {
		return nil
	}
	return dc
}
