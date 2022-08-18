// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import low "github.com/pb33f/libopenapi/datamodel/low/3.0"

type Response struct {
	Description string
	Headers     map[string]*Header
	Content     map[string]*MediaType
	Extensions  map[string]any
	Links       map[string]*Link
	low         *low.Response
}

func (r *Response) GoLow() *low.Response {
	return r.low
}
