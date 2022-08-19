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

func NewEncoding(encoding *low.Encoding) *Encoding {
	e := new(Encoding)
	e.low = encoding
	e.ContentType = encoding.ContentType.Value
	e.Style = encoding.Style.Value
	e.Explode = encoding.Explode.Value
	e.AllowReserved = encoding.AllowReserved.Value
	//e.Headers
	return e
}

func (e *Encoding) GoLow() *low.Encoding {
	return e.low
}
