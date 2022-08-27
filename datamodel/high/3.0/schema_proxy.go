// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	v3 "github.com/pb33f/libopenapi/datamodel/low/3.0"
)

type SchemaProxy struct {
	schema     *low.NodeReference[*v3.SchemaProxy]
	buildError error
}

func (sp *SchemaProxy) Schema() *Schema {
	s := sp.schema.Value.Schema()
	if s == nil {
		sp.buildError = sp.GetBuildError()
		return nil
	}
	return NewSchema(s)
}

func (sp *SchemaProxy) GetBuildError() error {
	return sp.buildError
}
