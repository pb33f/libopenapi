// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	lowmodel "github.com/pb33f/libopenapi/datamodel/low"
	low "github.com/pb33f/libopenapi/datamodel/low/3.0"
)

type Header struct {
	Description     string
	Required        bool
	Deprecated      bool
	AllowEmptyValue bool
	Style           string
	Explode         bool
	AllowReserved   bool
	Schema          *SchemaProxy
	Example         any
	Examples        map[string]*Example
	Content         map[string]*MediaType
	Extensions      map[string]any
	low             *low.Header
}

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
		h.Schema = &SchemaProxy{schema: &lowmodel.NodeReference[*low.SchemaProxy]{
			Value:     header.Schema.Value,
			KeyNode:   header.Schema.KeyNode,
			ValueNode: header.Schema.ValueNode,
		}}
	}
	h.Content = ExtractContent(header.Content.Value)
	h.Example = header.Example.Value
	h.Examples = ExtractExamples(header.Examples.Value)
	return h
}

func (h *Header) GoLow() *low.Header {
	return h.low
}

func ExtractHeaders(elements map[lowmodel.KeyReference[string]]lowmodel.ValueReference[*low.Header]) map[string]*Header {
	extracted := make(map[string]*Header)
	for k, v := range elements {
		extracted[k.Value] = NewHeader(v.Value)
	}
	return extracted
}
