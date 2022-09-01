// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/shared"
)

type Definitions struct {
	Schemas low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*shared.SchemaProxy]]
}

func (d *Definitions) FindSchema(schema string) *low.ValueReference[*shared.SchemaProxy] {
	return low.FindItemInMap[*shared.SchemaProxy](schema, d.Schemas.Value)
}
