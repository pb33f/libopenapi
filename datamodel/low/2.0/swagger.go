// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
)

type Swagger struct {
	Swagger  low.NodeReference[string]
	Info     low.NodeReference[*base.Info]
	Host     low.NodeReference[string]
	BasePath low.NodeReference[string]
	Schemes  low.NodeReference[[]low.ValueReference[string]]
	Consumes low.NodeReference[[]low.ValueReference[string]]
	Produces low.NodeReference[[]low.ValueReference[string]]

	// TODO: paths

	Definitions         low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*base.SchemaProxy]]
	Parameters          low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Parameter]]
	Responses           low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Response]]
	SecurityDefinitions low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*SecurityScheme]]
	Security            low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*SecurityRequirement]]
	Tags                low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*base.Tag]]
	ExternalDocs        low.NodeReference[*base.ExternalDoc]
	Extensions          map[low.KeyReference[string]]low.ValueReference[any]
}
