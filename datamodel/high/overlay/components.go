// Copyright 2022-2025 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package overlay

import (
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/high"
	low "github.com/pb33f/libopenapi/datamodel/low/overlay"
	"github.com/pb33f/libopenapi/orderedmap"
)

// Components represents a high-level Overlay Components Object.
// https://spec.openapis.org/overlay/v1.2.0#components-object
type Components struct {
	Actions    *orderedmap.Map[string, *ReusableAction] `json:"actions,omitempty" yaml:"actions,omitempty"`
	Extensions *orderedmap.Map[string, *yaml.Node]      `json:"-" yaml:"-"`
	low        *low.Components
}

// NewComponents creates a new high-level Components instance from a low-level one.
func NewComponents(info *low.Components) *Components {
	i := new(Components)
	i.low = info
	if info.Actions.Value != nil {
		i.Actions = orderedmap.New[string, *ReusableAction]()
		for pair := info.Actions.Value.First(); pair != nil; pair = pair.Next() {
			i.Actions.Set(pair.Key().Value, NewReusableAction(pair.Value().Value))
		}
	}
	i.Extensions = high.ExtractExtensions(info.Extensions)
	return i
}

// GoLow returns the low-level Components instance used to create the high-level one.
func (i *Components) GoLow() *low.Components {
	return i.low
}

// GoLowUntyped returns the low-level Components instance with no type.
func (i *Components) GoLowUntyped() any {
	return i.low
}

// Render returns a YAML representation of the Components object as a byte slice.
func (i *Components) Render() ([]byte, error) {
	return yaml.Marshal(i)
}

// MarshalYAML creates a ready to render YAML representation of the Components object.
func (i *Components) MarshalYAML() (any, error) {
	m := orderedmap.New[string, any]()
	if i.Actions != nil {
		m.Set(low.ActionsLabel, i.Actions)
	}
	for pair := i.Extensions.First(); pair != nil; pair = pair.Next() {
		m.Set(pair.Key(), pair.Value())
	}
	return m, nil
}
