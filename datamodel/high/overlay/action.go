// Copyright 2022-2025 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package overlay

import (
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/high"
	low "github.com/pb33f/libopenapi/datamodel/low/overlay"
	"github.com/pb33f/libopenapi/orderedmap"
)

// Action represents an Overlay action or reusable action reference.
// Ref identifies a reusable action; the other fields provide local overrides.
// https://spec.openapis.org/overlay/v1.2.0#action-object
type Action struct {
	Ref         string                              `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Target      string                              `json:"target,omitempty" yaml:"target,omitempty"`
	Description string                              `json:"description,omitempty" yaml:"description,omitempty"`
	Update      *yaml.Node                          `json:"update,omitempty" yaml:"update,omitempty"`
	Remove      bool                                `json:"remove,omitempty" yaml:"remove,omitempty"`
	Copy        string                              `json:"copy,omitempty" yaml:"copy,omitempty"`
	Extensions  *orderedmap.Map[string, *yaml.Node] `json:"-" yaml:"-"`
	low         *low.Action
	removeSet   bool
}

// NewAction creates a new high-level Action instance from a low-level one.
func NewAction(action *low.Action) *Action {
	a := new(Action)
	a.low = action
	a.Ref = action.Ref.Value
	a.removeSet = action.Remove.KeyNode != nil
	if !action.Target.IsEmpty() {
		a.Target = action.Target.Value
	}
	if !action.Description.IsEmpty() {
		a.Description = action.Description.Value
	}
	if !action.Update.IsEmpty() {
		a.Update = action.Update.Value
	}
	if !action.Remove.IsEmpty() {
		a.Remove = action.Remove.Value
	}
	if !action.Copy.IsEmpty() {
		a.Copy = action.Copy.Value
	}
	a.Extensions = high.ExtractExtensions(action.Extensions)
	return a
}

// GoLow returns the low-level Action instance used to create the high-level one.
func (a *Action) GoLow() *low.Action {
	return a.low
}

// GoLowUntyped returns the low-level Action instance with no type.
func (a *Action) GoLowUntyped() any {
	return a.low
}

// Render returns a YAML representation of the Action object as a byte slice.
func (a *Action) Render() ([]byte, error) {
	return yaml.Marshal(a)
}

// MarshalYAML creates a ready to render YAML representation of the Action object.
func (a *Action) MarshalYAML() (any, error) {
	m := orderedmap.New[string, any]()
	if a.Ref != "" || (a.low != nil && a.low.Ref.KeyNode != nil) {
		m.Set(low.RefLabel, a.Ref)
	}
	if a.Target != "" || (a.low != nil && a.low.Target.KeyNode != nil) {
		m.Set(low.TargetLabel, a.Target)
	}
	if a.Description != "" || (a.low != nil && a.low.Description.KeyNode != nil) {
		m.Set(low.DescriptionLabel, a.Description)
	}
	if a.Copy != "" || (a.low != nil && a.low.Copy.KeyNode != nil) {
		m.Set(low.CopyLabel, a.Copy)
	}
	if a.Update != nil {
		m.Set(low.UpdateLabel, a.Update)
	}
	if a.Remove || a.removeSet {
		m.Set(low.RemoveLabel, a.Remove)
	}
	for pair := a.Extensions.First(); pair != nil; pair = pair.Next() {
		m.Set(pair.Key(), pair.Value())
	}
	return m, nil
}

// SetRemove sets remove and preserves an explicit false override when rendering or resolving a reusable action.
func (a *Action) SetRemove(remove bool) { a.Remove = remove; a.removeSet = true }

// HasRemove reports whether remove is explicitly set, including a false override.
func (a *Action) HasRemove() bool { return a.Remove || a.removeSet }
