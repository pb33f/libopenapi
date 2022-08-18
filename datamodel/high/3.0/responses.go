// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import low "github.com/pb33f/libopenapi/datamodel/low/3.0"

type Responses struct {
	Codes   map[string]*Response
	Default *Response
	low     *low.Responses
}

func (r *Responses) GoLow() *low.Responses {
	return r.low
}
