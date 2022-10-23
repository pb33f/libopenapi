// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/what-changed/core"
)

// ExternalDocChanges represents changes made to any ExternalDoc object from an OpenAPI document.
type ExternalDocChanges struct {
	core.PropertyChanges
	ExtensionChanges *ExtensionChanges
}

// TotalChanges returns a count of everything that changed
func (e *ExternalDocChanges) TotalChanges() int {
	c := e.PropertyChanges.TotalChanges()
	if e.ExtensionChanges != nil {
		c += e.ExtensionChanges.TotalChanges()
	}
	return c
}

// TotalBreakingChanges always returns 0 for ExternalDoc objects, they are non-binding.
func (e *ExternalDocChanges) TotalBreakingChanges() int {
	return 0
}

// CompareExternalDocs will compare a left (original) and a right (new) slice of ValueReference
// nodes for any changes between them. If there are changes, then a pointer to ExternalDocChanges
// is returned, otherwise if nothing changed - then nil is returned.
func CompareExternalDocs(l, r *base.ExternalDoc) *ExternalDocChanges {
	var changes []*core.Change
	var props []*core.PropertyCheck

	// URL
	props = append(props, &core.PropertyCheck{
		LeftNode:  l.URL.ValueNode,
		RightNode: r.URL.ValueNode,
		Label:     v3.URLLabel,
		Changes:   &changes,
		Breaking:  false,
		Original:  l,
		New:       r,
	})

	// description.
	props = append(props, &core.PropertyCheck{
		LeftNode:  l.Description.ValueNode,
		RightNode: r.Description.ValueNode,
		Label:     v3.DescriptionLabel,
		Changes:   &changes,
		Breaking:  false,
		Original:  l,
		New:       r,
	})

	// check everything.
	core.CheckProperties(props)

	dc := new(ExternalDocChanges)
	dc.Changes = changes

	// check extensions
	dc.ExtensionChanges = CheckExtensions(l, r)
	if dc.TotalChanges() <= 0 {
		return nil
	}
	return dc
}
