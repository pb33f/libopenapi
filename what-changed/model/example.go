// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"github.com/pb33f/libopenapi/datamodel/low/base"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/what-changed/core"
)

// ExampleChanges represent changes to an Example object, part of an OpenAPI specification.
type ExampleChanges struct {
	core.PropertyChanges
	ExtensionChanges *ExtensionChanges
}

// TotalChanges returns the total number of changes made to Example
func (e *ExampleChanges) TotalChanges() int {
	l := e.PropertyChanges.TotalChanges()
	if e.ExtensionChanges != nil {
		l += e.ExtensionChanges.PropertyChanges.TotalChanges()
	}
	return l
}

// TotalBreakingChanges returns the total number of breaking changes made to Example
func (e *ExampleChanges) TotalBreakingChanges() int {
	l := e.PropertyChanges.TotalBreakingChanges()
	return l
}

// TotalChanges

func CompareExamples(l, r *base.Example) *ExampleChanges {

	ec := new(ExampleChanges)
	var changes []*core.Change
	var props []*core.PropertyCheck

	// Summary
	props = append(props, &core.PropertyCheck{
		LeftNode:  l.Summary.ValueNode,
		RightNode: r.Summary.ValueNode,
		Label:     v3.SummaryLabel,
		Changes:   &changes,
		Breaking:  false,
		Original:  l,
		New:       r,
	})

	// Description
	props = append(props, &core.PropertyCheck{
		LeftNode:  l.Description.ValueNode,
		RightNode: r.Description.ValueNode,
		Label:     v3.DescriptionLabel,
		Changes:   &changes,
		Breaking:  false,
		Original:  l,
		New:       r,
	})

	// Value
	props = append(props, &core.PropertyCheck{
		LeftNode:  l.Value.ValueNode,
		RightNode: r.Value.ValueNode,
		Label:     v3.ValueLabel,
		Changes:   &changes,
		Breaking:  false,
		Original:  l,
		New:       r,
	})

	// ExternalValue
	props = append(props, &core.PropertyCheck{
		LeftNode:  l.ExternalValue.ValueNode,
		RightNode: r.ExternalValue.ValueNode,
		Label:     v3.ExternalValue,
		Changes:   &changes,
		Breaking:  false,
		Original:  l,
		New:       r,
	})

	// check properties
	core.CheckProperties(props)

	// check extensions
	ec.ExtensionChanges = CheckExtensions(l, r)
	ec.Changes = changes
	if ec.TotalChanges() <= 0 {
		return nil
	}
	return ec
}
