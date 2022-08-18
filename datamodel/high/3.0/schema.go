// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import low "github.com/pb33f/libopenapi/datamodel/low/3.0"

type Schema struct {
	Title                string
	MultipleOf           int
	Maximum              int
	ExclusiveMaximum     int
	Minimum              int
	ExclusiveMinimum     int
	MaxLength            int
	MinLength            int
	Pattern              string
	Format               string
	MaxItems             int
	MinItems             int
	UniqueItems          int
	MaxProperties        int
	MinProperties        int
	Required             []string
	Enum                 []string
	Type                 string
	AllOf                []*Schema
	OneOf                []*Schema
	AnyOf                []*Schema
	Not                  []*Schema
	Items                []*Schema
	Properties           map[string]*Schema
	AdditionalProperties any
	Description          string
	Default              any
	Nullable             bool
	Discriminator        *Discriminator
	ReadOnly             bool
	WriteOnly            bool
	XML                  *XML
	ExternalDocs         *ExternalDoc
	Example              any
	Deprecated           bool
	Extensions           map[string]any
	low                  *low.Schema
}

func (s *Schema) GoLow() *low.Schema {
	return s.low
}
