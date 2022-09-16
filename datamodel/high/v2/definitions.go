// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	lowmodel "github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	low "github.com/pb33f/libopenapi/datamodel/low/v2"
)

type Definitions struct {
	Definitions map[string]*highbase.SchemaProxy
	low         *low.Definitions
}

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

func (d *Definitions) GoLow() *low.Definitions {
	return d.low
}
