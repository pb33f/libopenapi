// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"crypto/sha256"
	"strings"

	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"

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
	Expression *orderedmap.Map[low.KeyReference[string], low.ValueReference[*PathItem]]
	Extensions *orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]]
	KeyNode    *yaml.Node
	RootNode   *yaml.Node
	*low.Reference
}

// GetExtensions returns all Callback extensions and satisfies the low.HasExtensions interface.
func (cb *Callback) GetExtensions() *orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]] {
	return cb.Extensions
}

// GetRootNode returns the root yaml node of the Callback object
func (cb *Callback) GetRootNode() *yaml.Node {
	return cb.RootNode
}

// GetKeyNode returns the key yaml node of the Callback object
func (cb *Callback) GetKeyNode() *yaml.Node {
	return cb.KeyNode
}

// FindExpression will locate a string expression and return a ValueReference containing the located PathItem
func (cb *Callback) FindExpression(exp string) *low.ValueReference[*PathItem] {
	return low.FindItemInOrderedMap(exp, cb.Expression)
}

// Build will extract extensions, expressions and PathItem objects for Callback
func (cb *Callback) Build(ctx context.Context, keyNode, root *yaml.Node, idx *index.SpecIndex) error {
	cb.KeyNode = keyNode
	root = utils.NodeAlias(root)
	cb.RootNode = root
	utils.CheckForMergeNodes(root)
	cb.Reference = new(low.Reference)
	cb.Extensions = low.ExtractExtensions(root)

	expressions, err := extractPathItemsMap(ctx, root, idx)
	if err != nil {
		return err
	}
	cb.Expression = expressions

	return nil
}

// Hash will return a consistent SHA256 Hash of the Callback object
func (cb *Callback) Hash() [32]byte {
	var f []string
	for pair := orderedmap.First(orderedmap.SortAlpha(cb.Expression)); pair != nil; pair = pair.Next() {
		f = append(f, low.GenerateHashString(pair.Value().Value))
	}

	f = append(f, low.HashExtensions(cb.Extensions)...)
	return sha256.Sum256([]byte(strings.Join(f, "|")))
}
