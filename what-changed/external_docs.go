// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package what_changed

import (
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	lowv3 "github.com/pb33f/libopenapi/datamodel/low/v3"
)

type ExternalDocChanges struct {
	PropertyChanges[*lowbase.ExternalDoc]
	ExtensionChanges *ExtensionChanges
}

func (e *ExternalDocChanges) TotalChanges() int {
	c := len(e.Changes)
	if e.ExtensionChanges != nil {
		c += len(e.ExtensionChanges.Changes)
	}
	return c
}

func CompareExternalDocs(l, r *lowbase.ExternalDoc) *ExternalDocChanges {
	var changes []*Change[*lowbase.ExternalDoc]
	var props []*PropertyCheck[*lowbase.ExternalDoc]

	// check URL
	props = append(props, &PropertyCheck[*lowbase.ExternalDoc]{
		LeftNode:  l.URL.ValueNode,
		RightNode: r.URL.ValueNode,
		Label:     lowv3.URLLabel,
		Changes:   &changes,
		Breaking:  false,
		Original:  l,
		New:       r,
	})

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
