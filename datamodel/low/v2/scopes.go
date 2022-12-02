// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"crypto/sha256"
	"fmt"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"sort"
	"strings"
)

// Scopes is a low-level representation of a Swagger / OpenAPI 2 OAuth2 Scopes object.
//
// Scopes lists the available scopes for an OAuth2 security scheme.
//  - https://swagger.io/specification/v2/#scopesObject
type Scopes struct {
	Values     map[low.KeyReference[string]]low.ValueReference[string]
	Extensions map[low.KeyReference[string]]low.ValueReference[any]
}

// GetExtensions returns all Scopes extensions and satisfies the low.HasExtensions interface.
func (s *Scopes) GetExtensions() map[low.KeyReference[string]]low.ValueReference[any] {
	return s.Extensions
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
				if strings.Contains(root.Content[k].Value, "x-") {
					continue
				}
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

// Hash will return a consistent SHA256 Hash of the Scopes object
func (s *Scopes) Hash() [32]byte {
	var f []string
	vals := make(map[string]low.ValueReference[string], len(s.Values))
	keys := make([]string, len(s.Values))
	z := 0
	for k := range s.Values {
		keys[z] = k.Value
		vals[k.Value] = s.Values[k]
		z++
	}
	sort.Strings(keys)
	for k := range keys {
		f = append(f, fmt.Sprintf("%s-%s", keys[k], vals[keys[k]].Value))
	}
	keys = make([]string, len(s.Extensions))
	z = 0
	for k := range s.Extensions {
		keys[z] = fmt.Sprintf("%s-%x", k.Value, sha256.Sum256([]byte(fmt.Sprint(s.Extensions[k].Value))))
		z++
	}
	sort.Strings(keys)
	f = append(f, keys...)
	return sha256.Sum256([]byte(strings.Join(f, "|")))
}
