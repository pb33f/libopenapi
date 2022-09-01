// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/shared"
)

type Swagger struct {
	Swagger  low.NodeReference[string]
	Info     low.NodeReference[*shared.Info]
	Host     low.NodeReference[string]
	BasePath low.NodeReference[string]
	Schemes  low.KeyReference[[]low.ValueReference[string]]
	Consumes low.KeyReference[[]low.ValueReference[string]]
	Produces low.KeyReference[[]low.ValueReference[string]]
}
