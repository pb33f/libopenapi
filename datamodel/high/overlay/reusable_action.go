// Copyright 2022-2025 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package overlay

import (
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/high"
	low "github.com/pb33f/libopenapi/datamodel/low/overlay"
	"github.com/pb33f/libopenapi/orderedmap"
)

// ReusableAction represents a high-level Overlay ReusableAction Object.
// https://spec.openapis.org/overlay/v1.2.0#reusable-action-object
type ReusableAction struct {
	Description string                              `json:"description,omitempty" yaml:"description,omitempty"`
	Fields      *Action                             `json:"fields,omitempty" yaml:"fields,omitempty"`
	Extensions  *orderedmap.Map[string, *yaml.Node] `json:"-" yaml:"-"`
	low         *low.ReusableAction
}

// NewReusableAction creates a new high-level ReusableAction instance from a low-level one.
func NewReusableAction(info *low.ReusableAction) *ReusableAction {
	i := new(ReusableAction)
	i.low = info
	i.Description = info.Description.Value
	if info.Fields.Value != nil {
		i.Fields = NewAction(info.Fields.Value)
	}
	i.Extensions = high.ExtractExtensions(info.Extensions)
	return i
}

// GoLow returns the low-level ReusableAction instance used to create the high-level one.
func (i *ReusableAction) GoLow() *low.ReusableAction {
	return i.low
}

// GoLowUntyped returns the low-level ReusableAction instance with no type.
func (i *ReusableAction) GoLowUntyped() any {
	return i.low
}

// Render returns a YAML representation of the ReusableAction object as a byte slice.
func (i *ReusableAction) Render() ([]byte, error) {
	return yaml.Marshal(i)
}

// MarshalYAML creates a ready to render YAML representation of the ReusableAction object.
func (i *ReusableAction) MarshalYAML() (any, error) {
	m := orderedmap.New[string, any]()
	if i.Description != "" {
		m.Set(low.DescriptionLabel, i.Description)
	}
	if i.Fields != nil {
		m.Set(low.FieldsLabel, i.Fields)
	}
	for pair := i.Extensions.First(); pair != nil; pair = pair.Next() {
		m.Set(pair.Key(), pair.Value())
	}
	return m, nil
}
