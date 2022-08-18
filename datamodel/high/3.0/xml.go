// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import low "github.com/pb33f/libopenapi/datamodel/low/3.0"

type XML struct {
	Name       string
	Namespace  string
	Prefix     string
	Attribute  bool
	Wrapped    bool
	Extensions map[string]any
	low        *low.XML
}

func (x *XML) GoLow() *low.XML {
	return x.low
}
