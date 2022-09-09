// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	v3 "github.com/pb33f/libopenapi/datamodel/low/base"
)

type SchemaProxy struct {
	schema     *low.NodeReference[*v3.SchemaProxy]
	buildError error
}

func NewSchemaProxy(schema *low.NodeReference[*v3.SchemaProxy]) *SchemaProxy {
	return &SchemaProxy{schema: schema}
}

func (sp *SchemaProxy) Schema() *Schema {
	s := sp.schema.Value.Schema()
	if s == nil {
		sp.buildError = sp.schema.Value.GetBuildError()
		return nil
	}
	return NewSchema(s)
}

func (sp *SchemaProxy) GetBuildError() error {
	return sp.buildError
}
