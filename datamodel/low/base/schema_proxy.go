// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type SchemaProxy struct {
	kn         *yaml.Node
	vn         *yaml.Node
	idx        *index.SpecIndex
	rendered   *Schema
	buildError error
}

func (sp *SchemaProxy) Build(root *yaml.Node, idx *index.SpecIndex) error {
	sp.vn = root
	sp.idx = idx
	return nil
}

func (sp *SchemaProxy) Schema() *Schema {
	if sp.rendered != nil {
		return sp.rendered
	}
	schema := new(Schema)
	_ = low.BuildModel(sp.vn, schema)
	err := schema.Build(sp.vn, sp.idx)
	if err != nil {
		low.Log.Error("unable to build schema",
			zap.Int("line", sp.vn.Line),
			zap.Int("column", sp.vn.Column),
			zap.String("error", err.Error()))
		sp.buildError = err
		return nil
	}
	sp.rendered = schema
	return schema
}

func (sp *SchemaProxy) GetBuildError() error {
	return sp.buildError
}
