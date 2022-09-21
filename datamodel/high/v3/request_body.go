// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	low "github.com/pb33f/libopenapi/datamodel/low/v3"
)

// RequestBody represents a high-level OpenAPI 3+ RequestBody object, backed by a low-level one.
//  - https://spec.openapis.org/oas/v3.1.0#request-body-object
type RequestBody struct {
	Description string
	Content     map[string]*MediaType
	Required    bool
	Extensions  map[string]any
	low         *low.RequestBody
}

// NewRequestBody will create a new high-level RequestBody instance, from a low-level one.
func NewRequestBody(rb *low.RequestBody) *RequestBody {
	r := new(RequestBody)
	r.low = rb
	r.Description = rb.Description.Value
	r.Required = rb.Required.Value
	r.Extensions = high.ExtractExtensions(rb.Extensions)
	r.Content = ExtractContent(rb.Content.Value)
	return r
}

// GoLow returns the low-level RequestBody instance used to create the high-level one.
func (r *RequestBody) GoLow() *low.RequestBody {
	return r.low
}
