// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

// Scopes is a low-level representation of a Swagger / OpenAPI 2 OAuth2 Scopes object.
//
// Scopes lists the available scopes for an OAuth2 security scheme.
//  - https://swagger.io/specification/v2/#scopesObject
type Scopes struct {
	Values     map[low.KeyReference[string]]low.ValueReference[string]
	Extensions map[low.KeyReference[string]]low.ValueReference[any]
}

// FindScope will attempt to locate a scope string using a key.
func (s *Scopes) FindScope(scope string) *low.ValueReference[string] {
	return low.FindItemInMap[string](scope, s.Values)
}

// Build will extract scope values and extensions from node.
func (s *Scopes) Build(root *yaml.Node, idx *index.SpecIndex) error {
	s.Extensions = low.ExtractExtensions(root)
	valueMap := make(map[low.KeyReference[string]]low.ValueReference[string])
	if utils.IsNodeMap(root) {
		for k := range root.Content {
			if k%2 == 0 {
				valueMap[low.KeyReference[string]{
					Value:   root.Content[k].Value,
					KeyNode: root.Content[k],
				}] = low.ValueReference[string]{
					Value:     root.Content[k+1].Value,
					ValueNode: root.Content[k+1],
				}
			}
		}
		s.Values = valueMap
	}
	return nil
}
