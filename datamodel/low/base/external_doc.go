// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"gopkg.in/yaml.v3"
)

type ExternalDoc struct {
	Description low.NodeReference[string]
	URL         low.NodeReference[string]
	Extensions  map[low.KeyReference[string]]low.ValueReference[any]
}

func (ex *ExternalDoc) FindExtension(ext string) *low.ValueReference[any] {
	return low.FindItemInMap[any](ext, ex.Extensions)
}

func (ex *ExternalDoc) Build(root *yaml.Node, idx *index.SpecIndex) error {
	ex.Extensions = low.ExtractExtensions(root)
	return nil
}
