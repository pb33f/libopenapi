// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	low "github.com/pb33f/libopenapi/datamodel/low/3.0"
)

type Parameter struct {
	Name            string
	In              string
	Description     string
	Required        bool
	Deprecated      bool
	AllowEmptyValue bool
	Style           string
	Explode         bool
	AllowReserved   bool
	Schema          *Schema
	Example         any
	Examples        map[string]*Example
	Content         map[string]*MediaType
	Extensions      map[string]any
	low             *low.Parameter
}

func NewParameter(param *low.Parameter) *Parameter {
	p := new(Parameter)
	p.low = param
	p.Name = param.Name.Value
	p.In = param.In.Value
	p.Description = param.Description.Value
	p.Deprecated = param.Deprecated.Value
	p.AllowEmptyValue = param.AllowEmptyValue.Value
	p.Style = param.Style.Value
	p.Explode = param.Explode.Value
	p.AllowReserved = param.AllowReserved.Value
	p.Schema = NewSchema(param.Schema.Value)
	p.Example = param.Example.Value
	p.Examples = ExtractExamples(param.Examples.Value)

	return p
}

func (p *Parameter) GoLow() *low.Parameter {
	return p.low
}
