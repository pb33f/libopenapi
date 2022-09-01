// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
)

type ParameterDefinitions struct {
	Parameters low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Parameter]]
}

type Definitions struct {
	Schemas low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*base.SchemaProxy]]
}

func (d *Definitions) FindSchema(schema string) *low.ValueReference[*base.SchemaProxy] {
	return low.FindItemInMap[*base.SchemaProxy](schema, d.Schemas.Value)
}

func (pd *ParameterDefinitions) FindSchema(schema string) *low.ValueReference[*Parameter] {
	return low.FindItemInMap[*Parameter](schema, pd.Parameters.Value)
}
