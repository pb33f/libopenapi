// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pb33f/libopenapi/datamodel/high"
	lowmodel "github.com/pb33f/libopenapi/datamodel/low"
	low "github.com/pb33f/libopenapi/datamodel/low/arazzo"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

// Selector represents a high-level Arazzo Selector Object.
// https://spec.openapis.org/arazzo/v1.1.0#selector-object
type Selector struct {
	Context        string                              `json:"context,omitempty" yaml:"context,omitempty"`
	Selector       string                              `json:"selector,omitempty" yaml:"selector,omitempty"`
	Type           string                              `json:"-" yaml:"-"`
	ExpressionType *ExpressionType                     `json:"-" yaml:"-"`
	Extensions     *orderedmap.Map[string, *yaml.Node] `json:"-" yaml:"-"`
	low            *low.Selector
}

// NewSelector creates a high-level Selector from its low-level model.
func NewSelector(selector *low.Selector) *Selector {
	s := &Selector{low: selector}
	if !selector.Context.IsEmpty() {
		s.Context = selector.Context.Value
	}
	if !selector.Selector.IsEmpty() {
		s.Selector = selector.Selector.Value
	}
	if !selector.Type.IsEmpty() && selector.Type.Value != nil {
		node := dereferenceAliasNode(selector.Type.Value)
		if node != nil {
			switch node.Kind {
			case yaml.ScalarNode:
				s.Type = node.Value
			case yaml.MappingNode:
				expressionType := new(low.ExpressionType)
				if err := lowmodel.BuildModel(node, expressionType); err == nil {
					if err = expressionType.Build(context.Background(), nil, node, nil); err == nil {
						s.ExpressionType = NewExpressionType(expressionType)
					}
				}
			}
		}
	}
	s.Extensions = high.ExtractExtensions(selector.Extensions)
	return s
}

// GetEffectiveType returns the scalar type or the type declared by the Expression Type Object.
func (s *Selector) GetEffectiveType() string {
	if s.ExpressionType != nil {
		return s.ExpressionType.Type
	}
	return s.Type
}

// GoLow returns the low-level Selector used to create this model.
func (s *Selector) GoLow() *low.Selector {
	return s.low
}

// GoLowUntyped returns the low-level Selector without a concrete type.
func (s *Selector) GoLowUntyped() any {
	return s.low
}

// Render returns a YAML representation of the Selector.
func (s *Selector) Render() ([]byte, error) {
	return yaml.Marshal(s)
}

// MarshalYAML renders the Selector and rejects competing type variants.
func (s *Selector) MarshalYAML() (any, error) {
	if s.Type != "" && s.ExpressionType != nil {
		return nil, fmt.Errorf("selector cannot contain both scalar and object type variants")
	}
	m := orderedmap.New[string, any]()
	if s.Context != "" {
		m.Set(low.ContextLabel, s.Context)
	}
	if s.Selector != "" {
		m.Set(low.SelectorLabel, s.Selector)
	}
	if s.ExpressionType != nil {
		m.Set(low.TypeLabel, s.ExpressionType)
	} else if s.Type != "" {
		m.Set(low.TypeLabel, s.Type)
	}
	marshalExtensions(m, s.Extensions)
	return m, nil
}

// MarshalJSON renders the same scalar-or-object type union used by YAML.
func (s *Selector) MarshalJSON() ([]byte, error) {
	value, err := s.MarshalYAML()
	if err != nil {
		return nil, err
	}
	return json.Marshal(value)
}
