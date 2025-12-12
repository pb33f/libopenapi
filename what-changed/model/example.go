// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"fmt"
	"sort"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/utils"
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
// left and right Example instances.
func CompareExamples(l, r *base.Example) *ExampleChanges {
	ec := new(ExampleChanges)
	var changes []*Change
	var props []*PropertyCheck

	// Summary
	props = append(props, &PropertyCheck{
		LeftNode:  l.Summary.ValueNode,
		RightNode: r.Summary.ValueNode,
		Label:     v3.SummaryLabel,
		Changes:   &changes,
		Breaking:  BreakingModified(CompExample, PropSummary),
		Original:  l,
		New:       r,
	})

	// Description
	props = append(props, &PropertyCheck{
		LeftNode:  l.Description.ValueNode,
		RightNode: r.Description.ValueNode,
		Label:     v3.DescriptionLabel,
		Changes:   &changes,
		Breaking:  BreakingModified(CompExample, PropDescription),
		Original:  l,
		New:       r,
	})

	// Value
	if utils.IsNodeMap(l.Value.ValueNode) && utils.IsNodeMap(r.Value.ValueNode) {
		lKeys := make([]string, len(l.Value.ValueNode.Content)/2)
		rKeys := make([]string, len(r.Value.ValueNode.Content)/2)
		z := 0
		for k := range l.Value.ValueNode.Content {
			if k%2 == 0 {
				// if there is no value (value is another map or something else), render the node into yaml and hash it.
				// https://github.com/pb33f/libopenapi/issues/61
				val := l.Value.ValueNode.Content[k+1].Value
				if val == "" {
					val = low.HashYAMLNodeSlice(l.Value.ValueNode.Content[k+1].Content)
				}
				lKeys[z] = fmt.Sprintf("%v-%v-%v",
					l.Value.ValueNode.Content[k].Value,
					l.Value.ValueNode.Content[k+1].Tag,
					fmt.Sprintf("%x", val))
				z++
			} else {
				continue
			}
		}
		z = 0
		for k := range r.Value.ValueNode.Content {
			if k%2 == 0 {
				// if there is no value (value is another map or something else), render the node into yaml and hash it.
				// https://github.com/pb33f/libopenapi/issues/61
				val := r.Value.ValueNode.Content[k+1].Value
				if val == "" {
					val = low.HashYAMLNodeSlice(r.Value.ValueNode.Content[k+1].Content)
				}
				rKeys[z] = fmt.Sprintf("%v-%v-%v",
					r.Value.ValueNode.Content[k].Value,
					r.Value.ValueNode.Content[k+1].Tag,
					fmt.Sprintf("%x", val))
				z++
			} else {
				continue
			}
		}
		sort.Strings(lKeys)
		sort.Strings(rKeys)
		for k := range lKeys {
			if k < len(rKeys) && lKeys[k] != rKeys[k] {
				CreateChangeWithEncoding(&changes, Modified, v3.ValueLabel,
					l.Value.GetValueNode(), r.Value.GetValueNode(), BreakingModified(CompExample, PropValue), l.Value.GetValue(), r.Value.GetValue())
				continue
			}
			if k >= len(rKeys) {
				CreateChangeWithEncoding(&changes, PropertyRemoved, v3.ValueLabel,
					l.Value.ValueNode, r.Value.ValueNode, BreakingRemoved(CompExample, PropValue), l.Value.Value, r.Value.Value)
			}
		}
		for k := range rKeys {
			if k >= len(lKeys) {
				CreateChangeWithEncoding(&changes, PropertyAdded, v3.ValueLabel,
					l.Value.ValueNode, r.Value.ValueNode, BreakingAdded(CompExample, PropValue), l.Value.Value, r.Value.Value)
			}
		}
	} else {
		props = append(props, &PropertyCheck{
			LeftNode:  l.Value.ValueNode,
			RightNode: r.Value.ValueNode,
			Label:     v3.ValueLabel,
			Changes:   &changes,
			Breaking:  BreakingModified(CompExample, PropValue),
			Original:  l,
			New:       r,
		})
	}
	// ExternalValue
	props = append(props, &PropertyCheck{
		LeftNode:  l.ExternalValue.ValueNode,
		RightNode: r.ExternalValue.ValueNode,
		Label:     v3.ExternalValue,
		Changes:   &changes,
		Breaking:  BreakingModified(CompExample, PropExternalValue),
		Original:  l,
		New:       r,
	})

	// DataValue (OpenAPI 3.2+)
	props = append(props, &PropertyCheck{
		LeftNode:  l.DataValue.ValueNode,
		RightNode: r.DataValue.ValueNode,
		Label:     base.DataValueLabel,
		Changes:   &changes,
		Breaking:  BreakingModified(CompExample, PropDataValue),
		Original:  l,
		New:       r,
	})

	// SerializedValue (OpenAPI 3.2+)
	props = append(props, &PropertyCheck{
		LeftNode:  l.SerializedValue.ValueNode,
		RightNode: r.SerializedValue.ValueNode,
		Label:     base.SerializedValueLabel,
		Changes:   &changes,
		Breaking:  BreakingModified(CompExample, PropSerializedValue),
		Original:  l,
		New:       r,
	})

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
