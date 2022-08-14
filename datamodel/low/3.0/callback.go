// Copyright 2022 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"gopkg.in/yaml.v3"
)

type Callback struct {
	Expression low.ValueReference[map[low.KeyReference[string]]low.ValueReference[*PathItem]]
	Extensions map[low.KeyReference[string]]low.ValueReference[any]
}

func (cb *Callback) FindExpression(exp string) *low.ValueReference[*PathItem] {
	return low.FindItemInMap[*PathItem](exp, cb.Expression.Value)
}

func (cb *Callback) Build(root *yaml.Node, idx *index.SpecIndex) error {
	cb.Extensions = low.ExtractExtensions(root)

	// handle callback
	var currentCB *yaml.Node
	callbacks := make(map[low.KeyReference[string]]low.ValueReference[*PathItem])

	for i, callbackNode := range root.Content {
		if i%2 == 0 {
			currentCB = callbackNode
			continue
		}
		callback, eErr := low.ExtractObjectRaw[*PathItem](callbackNode, idx)
		if eErr != nil {
			return eErr
		}
		callbacks[low.KeyReference[string]{
			Value:   currentCB.Value,
			KeyNode: currentCB,
		}] = low.ValueReference[*PathItem]{
			Value:     callback,
			ValueNode: callbackNode,
		}
	}
	if len(callbacks) > 0 {
		cb.Expression = low.ValueReference[map[low.KeyReference[string]]low.ValueReference[*PathItem]]{
			Value:     callbacks,
			ValueNode: root,
		}
	}
	return nil
}
