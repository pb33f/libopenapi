// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/v3"
)

type RequestBodyChanges struct {
	PropertyChanges
	ContentChanges   map[string]*MediaTypeChanges
	ExtensionChanges *ExtensionChanges
}

func (rb *RequestBodyChanges) TotalChanges() int {
	c := rb.PropertyChanges.TotalChanges()
	for k := range rb.ContentChanges {
		c += rb.ContentChanges[k].TotalChanges()
	}
	if rb.ExtensionChanges != nil {
		c += rb.ExtensionChanges.TotalChanges()
	}
	return c
}

func (rb *RequestBodyChanges) TotalBreakingChanges() int {
	c := rb.PropertyChanges.TotalBreakingChanges()
	for k := range rb.ContentChanges {
		c += rb.ContentChanges[k].TotalBreakingChanges()
	}
	return c
}

func CompareRequestBodies(l, r *v3.RequestBody) *RequestBodyChanges {
	if low.AreEqual(l, r) {
		return nil
	}

	var changes []*Change
	var props []*PropertyCheck

	// description
	props = append(props, &PropertyCheck{
		LeftNode:  l.Description.ValueNode,
		RightNode: r.Description.ValueNode,
		Label:     v3.DescriptionLabel,
		Changes:   &changes,
		Breaking:  false,
		Original:  l,
		New:       r,
	})

	// required
	props = append(props, &PropertyCheck{
		LeftNode:  l.Required.ValueNode,
		RightNode: r.Required.ValueNode,
		Label:     v3.RequiredLabel,
		Changes:   &changes,
		Breaking:  true,
		Original:  l,
		New:       r,
	})

	CheckProperties(props)

	rbc := new(RequestBodyChanges)
	rbc.ContentChanges = CheckMapForChanges(l.Content.Value, r.Content.Value,
		&changes, v3.ContentLabel, CompareMediaTypes)
	rbc.ExtensionChanges = CompareExtensions(l.Extensions, r.Extensions)
	rbc.Changes = changes

	return rbc
}
