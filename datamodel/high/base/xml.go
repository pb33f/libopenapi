// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
    "github.com/pb33f/libopenapi/datamodel/high"
    low "github.com/pb33f/libopenapi/datamodel/low/base"
)

// XML represents a high-level representation of an XML object defined by all versions of OpenAPI and backed by
// low-level XML object.
//  v2 - https://swagger.io/specification/v2/#xmlObject
//  v3 - https://swagger.io/specification/#xml-object
type XML struct {
    Name       string
    Namespace  string
    Prefix     string
    Attribute  bool
    Wrapped    bool
    Extensions map[string]any
    low        *low.XML
}

// NewXML creates a new high-level XML instance from a low-level one.
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

// GoLow returns the low level XML reference used to create the high level one.
func (x *XML) GoLow() *low.XML {
    return x.low
}
