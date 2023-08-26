// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"crypto/sha256"
	"fmt"
	"github.com/pb33f/libopenapi/utils"
	"sort"
	"strings"

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
//   - https://spec.openapis.org/oas/v3.1.0#callback-object
type Callback struct {
	Expression low.ValueReference[map[low.KeyReference[string]]low.ValueReference[*PathItem]]
	Extensions map[low.KeyReference[string]]low.ValueReference[any]
	*low.Reference
}

// GetExtensions returns all Callback extensions and satisfies the low.HasExtensions interface.
func (cb *Callback) GetExtensions() map[low.KeyReference[string]]low.ValueReference[any] {
	return cb.Extensions
}

// FindExpression will locate a string expression and return a ValueReference containing the located PathItem
func (cb *Callback) FindExpression(exp string) *low.ValueReference[*PathItem] {
	return low.FindItemInMap[*PathItem](exp, cb.Expression.Value)
}

// Build will extract extensions, expressions and PathItem objects for Callback
func (cb *Callback) Build(_, root *yaml.Node, idx *index.SpecIndex) error {
	root = utils.NodeAlias(root)
	utils.CheckForMergeNodes(root)
	cb.Reference = new(low.Reference)
	cb.Extensions = low.ExtractExtensions(root)

	// handle callback
	var currentCB *yaml.Node
	callbacks := make(map[low.KeyReference[string]]low.ValueReference[*PathItem])

	for i, callbackNode := range root.Content {
		if i%2 == 0 {
			currentCB = callbackNode
			continue
		}
		if strings.HasPrefix(currentCB.Value, "x-") {
			continue // ignore extension.
		}
		callback, eErr, _, rv := low.ExtractObjectRaw[*PathItem](currentCB, callbackNode, idx)
		if eErr != nil {
			return eErr
		}
		callbacks[low.KeyReference[string]{
			Value:   currentCB.Value,
			KeyNode: currentCB,
		}] = low.ValueReference[*PathItem]{
			Value:     callback,
			ValueNode: callbackNode,
			Reference: rv,
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

// Hash will return a consistent SHA256 Hash of the Callback object
func (cb *Callback) Hash() [32]byte {
	var f []string
	var keys []string
	keys = make([]string, len(cb.Expression.Value))
	z := 0
	for k := range cb.Expression.Value {
		keys[z] = low.GenerateHashString(cb.Expression.Value[k].Value)
		z++
	}
	sort.Strings(keys)
	f = append(f, keys...)

	keys = make([]string, len(cb.Extensions))
	z = 0
	for k := range cb.Extensions {
		keys[z] = fmt.Sprintf("%s-%x", k.Value, sha256.Sum256([]byte(fmt.Sprint(cb.Extensions[k].Value))))
		z++
	}
	sort.Strings(keys)
	f = append(f, keys...)

	return sha256.Sum256([]byte(strings.Join(f, "|")))
}
