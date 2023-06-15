// Copyright 2022-2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	lowmodel "github.com/pb33f/libopenapi/datamodel/low"
	low "github.com/pb33f/libopenapi/datamodel/low/v3"
	"gopkg.in/yaml.v3"
)

// Encoding represents an OpenAPI 3+ Encoding object
//   - https://spec.openapis.org/oas/v3.1.0#encoding-object
type Encoding struct {
	ContentType   string             `json:"contentType,omitempty" yaml:"contentType,omitempty"`
	Headers       map[string]*Header `json:"headers,omitempty" yaml:"headers,omitempty"`
	Style         string             `json:"style,omitempty" yaml:"style,omitempty"`
	Explode       *bool              `json:"explode,omitempty" yaml:"explode,omitempty"`
	AllowReserved bool               `json:"allowReserved,omitempty" yaml:"allowReserved,omitempty"`
	low           *low.Encoding
}

// NewEncoding creates a new instance of Encoding from a low-level one.
func NewEncoding(encoding *low.Encoding) *Encoding {
	e := new(Encoding)
	e.low = encoding
	e.ContentType = encoding.ContentType.Value
	e.Style = encoding.Style.Value
	e.Explode = &encoding.Explode.Value
	e.AllowReserved = encoding.AllowReserved.Value
	e.Headers = ExtractHeaders(encoding.Headers.Value)
	return e
}

// GoLow returns the low-level Encoding instance used to create the high-level one.
func (e *Encoding) GoLow() *low.Encoding {
	return e.low
}

// GoLowUntyped will return the low-level Encoding instance that was used to create the high-level one, with no type
func (e *Encoding) GoLowUntyped() any {
	return e.low
}

// Render will return a YAML representation of the Encoding object as a byte slice.
func (e *Encoding) Render() ([]byte, error) {
	return yaml.Marshal(e)
}

// MarshalYAML will create a ready to render YAML representation of the Encoding object.
func (e *Encoding) MarshalYAML() (interface{}, error) {
	nb := high.NewNodeBuilder(e, e.low)
	return nb.Render(), nil
}

// ExtractEncoding converts hard to navigate low-level plumbing Encoding definitions, into a high-level simple map
func ExtractEncoding(elements map[lowmodel.KeyReference[string]]lowmodel.ValueReference[*low.Encoding]) map[string]*Encoding {
	extracted := make(map[string]*Encoding)
	for k, v := range elements {
		extracted[k.Value] = NewEncoding(v.Value)
	}
	return extracted
}
