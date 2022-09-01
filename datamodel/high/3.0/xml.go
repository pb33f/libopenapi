// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	low "github.com/pb33f/libopenapi/datamodel/low/base"
)

type XML struct {
	Name       string
	Namespace  string
	Prefix     string
	Attribute  bool
	Wrapped    bool
	Extensions map[string]any
	low        *low.XML
}

func NewXML(xml *low.XML) *XML {
	x := new(XML)
	x.low = xml
	x.Name = xml.Name.Value
	x.Namespace = xml.Namespace.Value
	x.Prefix = xml.Namespace.Value
	x.Attribute = xml.Attribute.Value
	x.Wrapped = xml.Wrapped.Value
	x.Extensions = high.ExtractExtensions(xml.Extensions)
	return x
}

func (x *XML) GoLow() *low.XML {
	return x.low
}
