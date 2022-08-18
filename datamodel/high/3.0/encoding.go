// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import low "github.com/pb33f/libopenapi/datamodel/low/3.0"

type Encoding struct {
	ContentType   string
	Headers       map[string]*Header
	Style         string
	Explode       bool
	AllowReserved bool
	low           *low.Encoding
}

func (e *Encoding) GoLow() *low.Encoding {
	return e.low
}
