// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	low "github.com/pb33f/libopenapi/datamodel/low/v2"
)

type Response struct {
	Description string
	Schema      *base.SchemaProxy
	Headers     map[string]*Header
	Examples    *Examples
	Extensions  map[string]any
	low         *low.Response
}

func NewResponse(response *low.Response) *Response {
	r := new(Response)
	r.low = response
	r.Extensions = high.ExtractExtensions(response.Extensions)
	if !response.Description.IsEmpty() {
		r.Description = response.Description.Value
	}
	if !response.Schema.IsEmpty() {
		r.Schema = base.NewSchemaProxy(&response.Schema)
	}
	if !response.Headers.IsEmpty() {
		headers := make(map[string]*Header)
		for k := range response.Headers.Value {
			headers[k.Value] = NewHeader(response.Headers.Value[k].Value)
		}
		r.Headers = headers
	}
	if !response.Examples.IsEmpty() {
		r.Examples = NewExamples(response.Examples.Value)
	}
	return r
}

func (r *Response) GoLow() *low.Response {
	return r.low
}
