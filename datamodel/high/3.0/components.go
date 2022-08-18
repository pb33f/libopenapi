// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import low "github.com/pb33f/libopenapi/datamodel/low/3.0"

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

func (c *Components) GoLow() *low.Components {
	return c.low
}
