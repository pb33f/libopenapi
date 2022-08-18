// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import low "github.com/pb33f/libopenapi/datamodel/low/3.0"

type MediaType struct {
	Schema     *Schema
	Example    any
	Examples   map[string]*Example
	Encoding   map[string]*Encoding
	Extensions map[string]any
	low        *low.MediaType
}

func (m *MediaType) GoLow() *low.MediaType {
	return m.low
}
