// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	lowmodel "github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	low "github.com/pb33f/libopenapi/datamodel/low/v2"
)

// Definitions is a high-level represents of a Swagger / OpenAPI 2 Definitions object, backed by a low-level one.
//
// An object to hold data types that can be consumed and produced by operations. These data types can be primitives,
// arrays or models.
//   - https://swagger.io/specification/v2/#definitionsObject
type Definitions struct {
	Definitions map[string]*highbase.SchemaProxy
	low         *low.Definitions
}

// NewDefinitions will create a new high-level instance of a Definition from a low-level one.
func NewDefinitions(definitions *low.Definitions) *Definitions {
	rd := new(Definitions)
	rd.low = definitions
	defs := make(map[string]*highbase.SchemaProxy)
	for k := range definitions.Schemas {
		defs[k.Value] = highbase.NewSchemaProxy(&lowmodel.NodeReference[*lowbase.SchemaProxy]{
			Value: definitions.Schemas[k].Value,
		})
	}
	rd.Definitions = defs
	return rd
}

// GoLow returns the low-level Definitions object used to create the high-level one.
func (d *Definitions) GoLow() *low.Definitions {
	return d.low
}
