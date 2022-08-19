// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	low "github.com/pb33f/libopenapi/datamodel/low/3.0"
)

type Components struct {
	Schemas         map[string]*Schema
	Responses       map[string]*Response
	Parameters      map[string]*Parameter
	Examples        map[string]*Example
	RequestBodies   map[string]*RequestBody
	Headers         map[string]*Header
	SecuritySchemes map[string]*SecurityScheme
	Links           map[string]*Link
	Callbacks       map[string]*Callback
	Extensions      map[string]any
	low             *low.Components
}

func NewComponents(comp *low.Components) *Components {
	c := new(Components)
	c.low = comp
	c.Extensions = high.ExtractExtensions(comp.Extensions)
	callbacks := make(map[string]*Callback)
	links := make(map[string]*Link)

	for k, v := range comp.Callbacks.Value {
		callbacks[k.Value] = NewCallback(v.Value)
	}
	c.Callbacks = callbacks
	for k, v := range comp.Links.Value {
		links[k.Value] = NewLink(v.Value)
	}
	c.Links = links
	return c
}

func (c *Components) GoLow() *low.Components {
	return c.low
}
