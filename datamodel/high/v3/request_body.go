// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	low "github.com/pb33f/libopenapi/datamodel/low/v3"
)

type RequestBody struct {
	Description string
	Content     map[string]*MediaType
	Required    bool
	Extensions  map[string]any
	low         *low.RequestBody
}

func NewRequestBody(rb *low.RequestBody) *RequestBody {
	r := new(RequestBody)
	r.low = rb
	r.Description = rb.Description.Value
	r.Required = rb.Required.Value
	r.Extensions = high.ExtractExtensions(rb.Extensions)
	r.Content = ExtractContent(rb.Content.Value)
	return r
}

func (r *RequestBody) GoLow() *low.RequestBody {
	return r.low
}
