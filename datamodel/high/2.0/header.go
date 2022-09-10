// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	low "github.com/pb33f/libopenapi/datamodel/low/2.0"
)

type Header struct {
	Type             string
	Format           string
	Description      string
	Items            *Items
	CollectionFormat string
	Default          any
	Maximum          int
	ExclusiveMaximum bool
	Minimum          int
	ExclusiveMinimum bool
	MaxLength        int
	MinLength        int
	Pattern          string
	MaxItems         int
	MinItems         int
	UniqueItems      bool
	Enum             []string
	MultipleOf       int
	Extensions       map[string]any
	low              *low.Header
}

func NewHeader(header *low.Header) *Header {
	h := new(Header)
	h.low = header
	h.Extensions = high.ExtractExtensions(header.Extensions)
	if !header.Type.IsEmpty() {
		h.Type = header.Type.Value
	}
	if !header.Format.IsEmpty() {
		h.Format = header.Type.Value
	}
	if !header.Description.IsEmpty() {
		h.Description = header.Description.Value
	}
	if !header.Items.IsEmpty() {
		h.Items = NewItems(header.Items.Value)
	}
	if !header.CollectionFormat.IsEmpty() {
		h.CollectionFormat = header.CollectionFormat.Value
	}
	if !header.Default.IsEmpty() {
		h.Default = header.Default.Value
	}
	if !header.Maximum.IsEmpty() {
		h.Maximum = header.Maximum.Value
	}
	if !header.ExclusiveMaximum.IsEmpty() {
		h.ExclusiveMaximum = header.ExclusiveMaximum.Value
	}
	if !header.Minimum.IsEmpty() {
		h.Minimum = header.Minimum.Value
	}
	if !header.ExclusiveMinimum.Value {
		h.ExclusiveMinimum = header.ExclusiveMinimum.Value
	}
	if !header.MaxLength.IsEmpty() {
		h.MaxLength = header.MaxLength.Value
	}
	if !header.MinLength.IsEmpty() {
		h.MinLength = header.MinLength.Value
	}
	if !header.Pattern.IsEmpty() {
		h.Pattern = header.Pattern.Value
	}
	if !header.MinItems.IsEmpty() {
		h.MinItems = header.MinItems.Value
	}
	if !header.MaxItems.IsEmpty() {
		h.MaxItems = header.MaxItems.Value
	}
	if !header.UniqueItems.IsEmpty() {
		h.UniqueItems = header.UniqueItems.IsEmpty()
	}
	if !header.Enum.IsEmpty() {
		h.Enum = header.Enum.Value
	}
	if !header.MultipleOf.IsEmpty() {
		h.MultipleOf = header.MultipleOf.Value
	}
	return h
}

func (h *Header) GoLow() *low.Header {
	return h.low
}
