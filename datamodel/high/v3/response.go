// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	low "github.com/pb33f/libopenapi/datamodel/low/v3"
)

type Response struct {
	Description string
	Headers     map[string]*Header
	Content     map[string]*MediaType
	Extensions  map[string]any
	Links       map[string]*Link
	low         *low.Response
}

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
		lnks := make(map[string]*Link)
		for k, v := range response.Links.Value {
			lnks[k.Value] = NewLink(v.Value)
		}
		r.Links = lnks
	}
	return r
}

func (r *Response) GoLow() *low.Response {
	return r.low
}
