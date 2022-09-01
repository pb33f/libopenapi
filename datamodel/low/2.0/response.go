// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
)

type Response struct {
	Description low.NodeReference[string]
	Schema      low.NodeReference[*base.SchemaProxy]
	Headers     low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Header]]
	Examples    low.NodeReference[map[low.KeyReference[string]]low.ValueReference[any]]
}
