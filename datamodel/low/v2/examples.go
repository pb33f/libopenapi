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

// Examples represents a low-level Swagger / OpenAPI 2 Example object.
// Allows sharing examples for operation responses
//   - https://swagger.io/specification/v2/#exampleObject
type Examples struct {
	Values map[low.KeyReference[string]]low.ValueReference[any]
}

// FindExample attempts to locate an example value, using a key label.
func (e *Examples) FindExample(name string) *low.ValueReference[any] {
	return low.FindItemInMap[any](name, e.Values)
}

// Build will extract all examples and will attempt to unmarshal content into a map or slice based on type.
func (e *Examples) Build(_, root *yaml.Node, _ *index.SpecIndex) error {
	root = utils.NodeAlias(root)
	utils.CheckForMergeNodes(root)
	var keyNode, currNode *yaml.Node
	var err error
	e.Values = make(map[low.KeyReference[string]]low.ValueReference[any])
	for i := range root.Content {
		if i%2 == 0 {
			keyNode = root.Content[i]
			continue
		}
		currNode = root.Content[i]
		var n map[string]interface{}
		err = currNode.Decode(&n)
		if err != nil {
			var k []interface{}
			err = currNode.Decode(&k)
			if err != nil {
				// lets just default to interface
				var j interface{}
				_ = currNode.Decode(&j)
				e.Values[low.KeyReference[string]{
					Value:   keyNode.Value,
					KeyNode: keyNode,
				}] = low.ValueReference[any]{
					Value:     j,
					ValueNode: currNode,
				}
				continue
			}
			e.Values[low.KeyReference[string]{
				Value:   keyNode.Value,
				KeyNode: keyNode,
			}] = low.ValueReference[any]{
				Value:     k,
				ValueNode: currNode,
			}
			continue
		}
		e.Values[low.KeyReference[string]{
			Value:   keyNode.Value,
			KeyNode: keyNode,
		}] = low.ValueReference[any]{
			Value:     n,
			ValueNode: currNode,
		}

	}
	return nil
}

// Hash will return a consistent SHA256 Hash of the Examples object
func (e *Examples) Hash() [32]byte {
	var f []string
	keys := make([]string, len(e.Values))
	z := 0
	for k := range e.Values {
		keys[z] = k.Value
		z++
	}
	sort.Strings(keys)
	for k := range keys {
		f = append(f, fmt.Sprintf("%v", e.FindExample(keys[k]).Value))
	}
	return sha256.Sum256([]byte(strings.Join(f, "|")))
}
