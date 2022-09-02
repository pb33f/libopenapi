// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"fmt"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

type Responses struct {
	Codes      map[low.KeyReference[string]]low.ValueReference[*Response]
	Default    low.NodeReference[*Response]
	Extensions map[low.KeyReference[string]]low.ValueReference[any]
}

func (r *Responses) Build(root *yaml.Node, idx *index.SpecIndex) error {
	r.Extensions = low.ExtractExtensions(root)

	if utils.IsNodeMap(root) {
		codes, err := low.ExtractMapFlatNoLookup[*Response](root, idx)
		if err != nil {
			return err
		}
		if codes != nil {
			r.Codes = codes
		}

		def, derr := low.ExtractObject[*Response](DefaultLabel, root, idx)
		if derr != nil {
			return derr
		}
		if def.Value != nil {
			r.Default = def
		}
	} else {
		return fmt.Errorf("responses build failed: vn node is not a map! line %d, col %d", root.Line, root.Column)
	}
	return nil
}

func (r *Responses) FindResponseByCode(code string) *low.ValueReference[*Response] {
	return low.FindItemInMap[*Response](code, r.Codes)
}
