// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"gopkg.in/yaml.v3"
)

const (
	ExamplesLabel = "examples"
)

type Examples struct {
	Values map[low.KeyReference[string]]low.ValueReference[any]
}

func (e *Examples) Build(root *yaml.Node, _ *index.SpecIndex) error {
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
