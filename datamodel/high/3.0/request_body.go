// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import low "github.com/pb33f/libopenapi/datamodel/low/3.0"

type RequestBody struct {
	Description string
	Content     map[string]*MediaType
	Required    bool
	Extensions  map[string]any
	low         *low.RequestBody
}

func (r *RequestBody) GoLow() *low.RequestBody {
	return r.low
}
