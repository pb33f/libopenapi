// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"github.com/pb33f/libopenapi/datamodel/low/base"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
)

// ExampleChanges represent changes to an Example object, part of an OpenAPI specification.
type ExampleChanges struct {
	*PropertyChanges
	ExtensionChanges *ExtensionChanges `json:"extensions,omitempty" yaml:"extensions,omitempty"`
}

// GetAllChanges returns a slice of all changes made between Example objects
func (e *ExampleChanges) GetAllChanges() []*Change {
	if e == nil {
		return nil
	}
	var changes []*Change
	changes = append(changes, e.Changes...)
	if e.ExtensionChanges != nil {
		changes = append(changes, e.ExtensionChanges.GetAllChanges()...)
	}
	return changes
}

// TotalChanges returns the total number of changes made to Example
func (e *ExampleChanges) TotalChanges() int {
	if e == nil {
		return 0
	}
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

// CompareExamples returns a pointer to ExampleChanges that contains all changes made between
// left and right Example instances. If l is nil, the example was added. If r is nil, it was removed.
func CompareExamples(l, r *base.Example) *ExampleChanges {
	ec := new(ExampleChanges)
	var changes []*Change

	if l == nil {
		// Example was added - use RootNode for proper line/column location
		CreateChange(&changes, ObjectAdded, v3.ExampleLabel,
			nil, r.RootNode, BreakingAdded(CompExample, PropValue), nil, r)
		ec.PropertyChanges = NewPropertyChanges(changes)
		return ec
	}
	if r == nil {
		// Example was removed - use RootNode for proper line/column location
		CreateChange(&changes, ObjectRemoved, v3.ExampleLabel,
			l.RootNode, nil, BreakingRemoved(CompExample, PropValue), l, nil)
		ec.PropertyChanges = NewPropertyChanges(changes)
		return ec
	}

	props := make([]*PropertyCheck, 0, 2)

	props = append(props,
		NewPropertyCheck(CompExample, PropSummary,
			l.Summary.ValueNode, r.Summary.ValueNode,
			v3.SummaryLabel, &changes, l, r),
		NewPropertyCheck(CompExample, PropDescription,
			l.Description.ValueNode, r.Description.ValueNode,
			v3.DescriptionLabel, &changes, l, r),
	)

	// Value and DataValue hold any YAML shape, so they are checked with encoding.
	CheckPropertiesWithEncoding([]*PropertyCheck{
		NewPropertyCheck(CompExample, PropValue,
			l.Value.ValueNode, r.Value.ValueNode,
			v3.ValueLabel, &changes, l, r),
		NewPropertyCheck(CompExample, PropDataValue,
			l.DataValue.ValueNode, r.DataValue.ValueNode,
			base.DataValueLabel, &changes, l, r),
	})

	// ExternalValue
	props = append(props, NewPropertyCheck(CompExample, PropExternalValue,
		l.ExternalValue.ValueNode, r.ExternalValue.ValueNode,
		v3.ExternalValue, &changes, l, r))

	// SerializedValue (OpenAPI 3.2+)
	props = append(props, NewPropertyCheck(CompExample, PropSerializedValue,
		l.SerializedValue.ValueNode, r.SerializedValue.ValueNode,
		base.SerializedValueLabel, &changes, l, r))

	// check properties
	CheckProperties(props)

	// check extensions
	ec.ExtensionChanges = CheckExtensions(l, r)
	ec.PropertyChanges = NewPropertyChanges(changes)
	if ec.TotalChanges() <= 0 {
		return nil
	}
	return ec
}
