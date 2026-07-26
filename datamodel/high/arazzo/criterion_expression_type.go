// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	low "github.com/pb33f/libopenapi/datamodel/low/arazzo"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

// ExpressionType represents a high-level Arazzo Expression Type Object.
// https://spec.openapis.org/arazzo/v1.1.0#expression-type-object
type ExpressionType struct {
	Type       string                              `json:"type,omitempty" yaml:"type,omitempty"`
	Version    string                              `json:"version,omitempty" yaml:"version,omitempty"`
	Extensions *orderedmap.Map[string, *yaml.Node] `json:"-" yaml:"-"`
	low        *low.ExpressionType
}

// NewExpressionType creates a new high-level ExpressionType instance from a low-level one.
func NewExpressionType(cet *low.ExpressionType) *ExpressionType {
	c := new(ExpressionType)
	c.low = cet
	if !cet.Type.IsEmpty() {
		c.Type = cet.Type.Value
	}
	if !cet.Version.IsEmpty() {
		c.Version = cet.Version.Value
	}
	c.Extensions = high.ExtractExtensions(cet.Extensions)
	return c
}

// GoLow returns the low-level CriterionExpressionType instance used to create the high-level one.
func (c *ExpressionType) GoLow() *low.ExpressionType {
	return c.low
}

// GoLowUntyped returns the low-level CriterionExpressionType instance with no type.
func (c *ExpressionType) GoLowUntyped() any {
	return c.low
}

// Render returns a YAML representation of the CriterionExpressionType object as a byte slice.
func (c *ExpressionType) Render() ([]byte, error) {
	return yaml.Marshal(c)
}

// MarshalYAML creates a ready to render YAML representation of the CriterionExpressionType object.
func (c *ExpressionType) MarshalYAML() (any, error) {
	m := orderedmap.New[string, any]()
	if c.Type != "" {
		m.Set("type", c.Type)
	}
	if c.Version != "" {
		m.Set("version", c.Version)
	}
	marshalExtensions(m, c.Extensions)
	return m, nil
}

// CriterionExpressionType is retained as an alias for source compatibility with Arazzo 1.0 callers.
type CriterionExpressionType = ExpressionType

// NewCriterionExpressionType creates a reusable ExpressionType from the legacy low-level alias.
func NewCriterionExpressionType(cet *low.CriterionExpressionType) *CriterionExpressionType {
	return NewExpressionType(cet)
}
