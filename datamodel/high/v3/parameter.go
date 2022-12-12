// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	low "github.com/pb33f/libopenapi/datamodel/low/v3"
)

// Parameter represents a high-level OpenAPI 3+ Parameter object, that is backed by a low-level one.
//
// A unique parameter is defined by a combination of a name and location.
//  - https://spec.openapis.org/oas/v3.1.0#parameter-object
type Parameter struct {
	Name            string
	In              string
	Description     string
	Required        bool
	Deprecated      bool
	AllowEmptyValue bool
	Style           string
	Explode         *bool
	AllowReserved   bool
	Schema          *base.SchemaProxy
	Example         any
	Examples        map[string]*base.Example
	Content         map[string]*MediaType
	Extensions      map[string]any
	low             *low.Parameter
}

// NewParameter will create a new high-level instance of a Parameter, using a low-level one.
func NewParameter(param *low.Parameter) *Parameter {
    p := new(Parameter)
    p.low = param
    p.Name = param.Name.Value
    p.In = param.In.Value
    p.Description = param.Description.Value
    p.Deprecated = param.Deprecated.Value
    p.AllowEmptyValue = param.AllowEmptyValue.Value
    p.Style = param.Style.Value
    if !param.Explode.IsEmpty() {
        p.Explode = &param.Explode.Value
    }
    p.AllowReserved = param.AllowReserved.Value
    if !param.Schema.IsEmpty() {
        p.Schema = base.NewSchemaProxy(&param.Schema)
    }
    p.Required = param.Required.Value
    p.Example = param.Example.Value
    p.Examples = base.ExtractExamples(param.Examples.Value)
    p.Content = ExtractContent(param.Content.Value)
    p.Extensions = high.ExtractExtensions(param.Extensions)
    return p
}

// GoLow returns the low-level Parameter used to create the high-level one.
func (p *Parameter) GoLow() *low.Parameter {
	return p.low
}
