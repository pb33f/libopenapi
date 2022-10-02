// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package what_changed

import (
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	lowv3 "github.com/pb33f/libopenapi/datamodel/low/v3"
)

// ExternalDocChanges represents changes made to any ExternalDoc object from an OpenAPI document.
type ExternalDocChanges struct {
	PropertyChanges[*lowbase.ExternalDoc]
	ExtensionChanges *ExtensionChanges
}

// TotalChanges returns a count of everything that changed
func (e *ExternalDocChanges) TotalChanges() int {
	c := len(e.Changes)
	if e.ExtensionChanges != nil {
		c += len(e.ExtensionChanges.Changes)
	}
	return c
}

// CompareExternalDocs will compare a left (original) and a right (new) slice of ValueReference
// nodes for any changes between them. If there are changes, then a pointer to ExternalDocChanges
// is returned, otherwise if nothing changed - then nil is returned.
func CompareExternalDocs(l, r *lowbase.ExternalDoc) *ExternalDocChanges {
	var changes []*Change[*lowbase.ExternalDoc]
	var props []*PropertyCheck[*lowbase.ExternalDoc]

	// URL
	props = append(props, &PropertyCheck[*lowbase.ExternalDoc]{
		LeftNode:  l.URL.ValueNode,
		RightNode: r.URL.ValueNode,
		Label:     lowv3.URLLabel,
		Changes:   &changes,
		Breaking:  false,
		Original:  l,
		New:       r,
	})

	// description.
	props = append(props, &PropertyCheck[*lowbase.ExternalDoc]{
		LeftNode:  l.Description.ValueNode,
		RightNode: r.Description.ValueNode,
		Label:     lowv3.DescriptionLabel,
		Changes:   &changes,
		Breaking:  false,
		Original:  l,
		New:       r,
	})

	// check everything.
	CheckProperties(props)

	dc := new(ExternalDocChanges)
	dc.Changes = changes

	// check extensions
	dc.ExtensionChanges = CheckExtensions(l, r)
	if len(dc.Changes) <= 0 && dc.ExtensionChanges == nil {
		return nil
	}
	return dc
}
