// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	low "github.com/pb33f/libopenapi/datamodel/low/3.0"
)

type MediaType struct {
	Schema     *Schema
	Example    any
	Examples   map[string]*Example
	Encoding   map[string]*Encoding
	Extensions map[string]any
	low        *low.MediaType
}

func NewMediaType(mediaType *low.MediaType) *MediaType {
	m := new(MediaType)
	m.low = mediaType
	if !mediaType.Schema.IsEmpty() {
		m.Schema = NewSchema(mediaType.Schema.Value)
	}
	m.Example = mediaType.Example
	m.Examples = ExtractExamples(mediaType.Examples.Value)
	m.Extensions = high.ExtractExtensions(mediaType.Extensions)

	// TODO continue this.

	return m
}

func (m *MediaType) GoLow() *low.MediaType {
	return m.low
}
