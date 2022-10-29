// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	low "github.com/pb33f/libopenapi/datamodel/low/v3"
)

// Response represents a high-level OpenAPI 3+ Response object that is backed by a low-level one.
//
// Describes a single response from an API Operation, including design-time, static links to
// operations based on the response.
//  - https://spec.openapis.org/oas/v3.1.0#response-object
type Response struct {
	Description string
	Headers     map[string]*Header
	Content     map[string]*MediaType
	Extensions  map[string]any
	Links       map[string]*Link
	low         *low.Response
}

// NewResponse creates a new high-level Response object that is backed by a low-level one.
func NewResponse(response *low.Response) *Response {
	r := new(Response)
	r.low = response
	r.Description = response.Description.Value
	if !response.Headers.IsEmpty() {
		r.Headers = ExtractHeaders(response.Headers.Value)
	}
	r.Extensions = high.ExtractExtensions(response.Extensions)
	if !response.Content.IsEmpty() {
		r.Content = ExtractContent(response.Content.Value)
	}
	if !response.Links.IsEmpty() {
		responseLinks := make(map[string]*Link)
		for k, v := range response.Links.Value {
			responseLinks[k.Value] = NewLink(v.Value)
		}
		r.Links = responseLinks
	}
	return r
}

// GoLow returns the low-level Response object that was used to create the high-level one.
func (r *Response) GoLow() *low.Response {
	return r.low
}
