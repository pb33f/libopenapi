// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	lowmodel "github.com/pb33f/libopenapi/datamodel/low"
	low "github.com/pb33f/libopenapi/datamodel/low/3.0"
)

type Example struct {
	Summary       string
	Description   string
	Value         any
	ExternalValue string
	Extensions    map[string]any
	low           *low.Example
}

func NewExample(example *low.Example) *Example {
	e := new(Example)
	e.low = example
	e.Summary = example.Summary.Value
	e.Description = example.Description.Value
	e.Value = example.Value
	e.ExternalValue = example.ExternalValue.Value
	e.Extensions = high.ExtractExtensions(example.Extensions)
	return e
}

func (e *Example) GoLow() *low.Example {
	return e.low
}

func ExtractExamples(elements map[lowmodel.KeyReference[string]]lowmodel.ValueReference[*low.Example]) map[string]*Example {
	extracted := make(map[string]*Example)
	for k, v := range elements {
		extracted[k.Value] = NewExample(v.Value)
	}
	return extracted
}
