// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"gopkg.in/yaml.v3"
)

// Callback represents a low-level Callback object for OpenAPI 3+.
//
// A map of possible out-of band callbacks related to the parent operation. Each value in the map is a
// PathItem Object that describes a set of requests that may be initiated by the API provider and the expected
// responses. The key value used to identify the path item object is an expression, evaluated at runtime,
// that identifies a URL to use for the callback operation.
//  - https://spec.openapis.org/oas/v3.1.0#callback-object
type Callback struct {
	Expression low.ValueReference[map[low.KeyReference[string]]low.ValueReference[*PathItem]]
	Extensions map[low.KeyReference[string]]low.ValueReference[any]
}

// FindExpression will locate a string expression and return a ValueReference containing the located PathItem
func (cb *Callback) FindExpression(exp string) *low.ValueReference[*PathItem] {
	return low.FindItemInMap[*PathItem](exp, cb.Expression.Value)
}

// Build will extract extensions, expressions and PathItem objects for Callback
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
