// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package what_changed

import (
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	lowv3 "github.com/pb33f/libopenapi/datamodel/low/v3"
)

type ContactChanges struct {
	PropertyChanges[*lowbase.Contact]
}

func (c *ContactChanges) TotalChanges() int {
	return len(c.Changes)
}

func CompareContact(l, r *lowbase.Contact) *ContactChanges {

	var changes []*Change[*lowbase.Contact]
	changeType := 0

	// check if the url was added
	if l != nil && r != nil && l.URL.Value == "" && r.URL.Value != "" {
		changeType = PropertyAdded
		CreateChange[*lowbase.Contact](&changes, changeType, lowv3.URLLabel,
			nil, r.Name.ValueNode, false, l, r)
	}

	// check if the name was added
	if l != nil && r != nil && l.Name.Value == "" && r.Name.Value != "" {
		changeType = PropertyAdded
		CreateChange[*lowbase.Contact](&changes, changeType, lowv3.NameLabel,
			nil, r.Name.ValueNode, false, l, r)
	}

	// if both urls are set, but are different.
	if l != nil && r != nil && l.URL.Value != r.URL.Value {
		changeType = Modified
		ctx := CreateContext(l.URL.ValueNode, r.URL.ValueNode)
		if ctx.HasChanged() {
			changeType = ModifiedAndMoved
		}
		CreateChange(&changes, changeType, lowv3.URLLabel,
			l.URL.ValueNode, r.Name.ValueNode, false, l, r)
	}

	// if both names are set, but are different.
	if l != nil && r != nil && l.Name.Value != r.Name.Value {
		changeType = Modified
		ctx := CreateContext(l.Name.ValueNode, r.Name.ValueNode)
		if ctx.HasChanged() {
			changeType = ModifiedAndMoved
		}
		CreateChange[*lowbase.Contact](&changes, changeType, lowv3.NameLabel,
			l.Name.ValueNode, r.Name.ValueNode, false, l, r)
	}

	// if both email addresses are set, but are different.
	if l != nil && r != nil && l.Email.Value != r.Email.Value {
		changeType = Modified
		ctx := CreateContext(l.Email.ValueNode, r.Email.ValueNode)
		if ctx.HasChanged() {
			changeType = ModifiedAndMoved
		}
		CreateChange[*lowbase.Contact](&changes, changeType, lowv3.EmailLabel,
			l.Email.ValueNode, r.Email.ValueNode, false, l, r)
	}

	if changeType == 0 {
		// no change, return nothing.
		return nil
	}
	dc := new(ContactChanges)
	dc.Changes = changes
	return dc

}
