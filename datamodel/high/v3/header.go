// Copyright 2022-2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	lowmodel "github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	low "github.com/pb33f/libopenapi/datamodel/low/v3"
	"gopkg.in/yaml.v3"
)

// Header represents a high-level OpenAPI 3+ Header object that is backed by a low-level one.
//  - https://spec.openapis.org/oas/v3.1.0#header-object
type Header struct {
	Description     string                       `json:"description,omitempty" yaml:"description,omitempty"`
	Required        bool                         `json:"required,omitempty" yaml:"required,omitempty"`
	Deprecated      bool                         `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	AllowEmptyValue bool                         `json:"allowEmptyValue,omitempty" yaml:"allowEmptyValue,omitempty"`
	Style           string                       `json:"style,omitempty" yaml:"style,omitempty"`
	Explode         bool                         `json:"explode,omitempty" yaml:"explode,omitempty"`
	AllowReserved   bool                         `json:"allowReserved,omitempty" yaml:"allowReserved,omitempty"`
	Schema          *highbase.SchemaProxy        `json:"schema,omitempty" yaml:"schema,omitempty"`
	Example         any                          `json:"example,omitempty" yaml:"example,omitempty"`
	Examples        map[string]*highbase.Example `json:"examples,omitempty" yaml:"examples,omitempty"`
	Content         map[string]*MediaType        `json:"content,omitempty" yaml:"content,omitempty"`
	Extensions      map[string]any               `json:"-" yaml:"-"`
	low             *low.Header
}

// NewHeader creates a new high-level Header instance from a low-level one.
func NewHeader(header *low.Header) *Header {
	h := new(Header)
	h.low = header
	h.Description = header.Description.Value
	h.Required = header.Required.Value
	h.Deprecated = header.Deprecated.Value
	h.AllowEmptyValue = header.AllowEmptyValue.Value
	h.Style = header.Style.Value
	h.Explode = header.Explode.Value
	h.AllowReserved = header.AllowReserved.Value
	if !header.Schema.IsEmpty() {
		h.Schema = highbase.NewSchemaProxy(&lowmodel.NodeReference[*base.SchemaProxy]{
			Value:     header.Schema.Value,
			KeyNode:   header.Schema.KeyNode,
			ValueNode: header.Schema.ValueNode,
		})
	}
	h.Content = ExtractContent(header.Content.Value)
	h.Example = header.Example.Value
	h.Examples = highbase.ExtractExamples(header.Examples.Value)
	return h
}

// GoLow returns the low-level Header instance used to create the high-level one.
func (h *Header) GoLow() *low.Header {
	return h.low
}

// GoLowUntyped will return the low-level Header instance that was used to create the high-level one, with no type
func (h *Header) GoLowUntyped() any {
	return h.low
}

// ExtractHeaders will extract a hard to navigate low-level Header map, into simple high-level one.
func ExtractHeaders(elements map[lowmodel.KeyReference[string]]lowmodel.ValueReference[*low.Header]) map[string]*Header {
	extracted := make(map[string]*Header)
	for k, v := range elements {
		extracted[k.Value] = NewHeader(v.Value)
	}
	return extracted
}

// Render will return a YAML representation of the Header object as a byte slice.
func (h *Header) Render() ([]byte, error) {
	return yaml.Marshal(h)
}

// MarshalYAML will create a ready to render YAML representation of the Header object.
func (h *Header) MarshalYAML() (interface{}, error) {
	nb := high.NewNodeBuilder(h, h.low)
	return nb.Render(), nil
}
