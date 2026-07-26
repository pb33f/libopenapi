// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"strings"

	"github.com/pb33f/libopenapi/datamodel/high"
	low "github.com/pb33f/libopenapi/datamodel/low/arazzo"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

// Parameter represents a high-level Arazzo Parameter Object.
// A parameter can be a full parameter definition or a Reusable Object with a $components reference.
// https://spec.openapis.org/arazzo/v1.1.0#parameter-object
type Parameter struct {
	Name       string                              `json:"name,omitempty" yaml:"name,omitempty"`
	In         string                              `json:"in,omitempty" yaml:"in,omitempty"`
	Value      *yaml.Node                          `json:"value,omitempty" yaml:"value,omitempty"`
	Reference  string                              `json:"reference,omitempty" yaml:"reference,omitempty"`
	Extensions *orderedmap.Map[string, *yaml.Node] `json:"-" yaml:"-"`
	low        *low.Parameter
}

// IsReusable returns true if this parameter is a Reusable Object (has a reference field).
func (p *Parameter) IsReusable() bool {
	return p.Reference != ""
}

// NewParameter creates a new high-level Parameter instance from a low-level one.
func NewParameter(param *low.Parameter) *Parameter {
	p := new(Parameter)
	p.low = param
	if !param.Name.IsEmpty() {
		p.Name = param.Name.Value
	}
	if !param.In.IsEmpty() {
		p.In = param.In.Value
	}
	if !param.Value.IsEmpty() {
		p.Value = param.Value.Value
	}
	if !param.ComponentRef.IsEmpty() {
		p.Reference = param.ComponentRef.Value
	}
	p.Extensions = high.ExtractExtensions(param.Extensions)
	return p
}

// IsRuntimeExpression reports whether the parameter value is a runtime-expression scalar.
func (p *Parameter) IsRuntimeExpression() bool {
	if p == nil || p.Value == nil || p.Value.Kind != yaml.ScalarNode {
		return false
	}
	return strings.HasPrefix(p.Value.Value, "$") || strings.Contains(p.Value.Value, "{$")
}

// IsSelector reports whether the parameter value is a Selector Object.
func (p *Parameter) IsSelector() bool {
	return p != nil && selectorFromNode(p.Value) != nil
}

// GetSelector returns the typed Selector Object when active.
func (p *Parameter) GetSelector() (*Selector, bool) {
	if p == nil {
		return nil, false
	}
	selector := selectorFromNode(p.Value)
	return selector, selector != nil
}

// GetValueNode returns the original parameter value node.
func (p *Parameter) GetValueNode() *yaml.Node {
	if p == nil {
		return nil
	}
	return p.Value
}

// GoLow returns the low-level Parameter instance used to create the high-level one.
func (p *Parameter) GoLow() *low.Parameter {
	return p.low
}

// GoLowUntyped returns the low-level Parameter instance with no type.
func (p *Parameter) GoLowUntyped() any {
	return p.low
}

// Render returns a YAML representation of the Parameter object as a byte slice.
func (p *Parameter) Render() ([]byte, error) {
	return yaml.Marshal(p)
}

// MarshalYAML creates a ready to render YAML representation of the Parameter object.
func (p *Parameter) MarshalYAML() (any, error) {
	m := orderedmap.New[string, any]()
	if p.Reference != "" {
		m.Set(low.ReferenceLabel, p.Reference)
		if p.Value != nil {
			m.Set(low.ValueLabel, p.Value)
		}
		return m, nil
	}
	if p.Name != "" {
		m.Set(low.NameLabel, p.Name)
	}
	if p.In != "" {
		m.Set(low.InLabel, p.In)
	}
	if p.Value != nil {
		m.Set(low.ValueLabel, p.Value)
	}
	marshalExtensions(m, p.Extensions)
	return m, nil
}
