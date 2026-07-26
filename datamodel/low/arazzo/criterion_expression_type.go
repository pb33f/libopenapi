// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"context"
	"hash/maphash"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

// ExpressionType represents a low-level Arazzo Expression Type Object.
// https://spec.openapis.org/arazzo/v1.1.0#expression-type-object
type ExpressionType struct {
	Type       low.NodeReference[string]
	Version    low.NodeReference[string]
	Extensions *orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]]
	KeyNode    *yaml.Node
	RootNode   *yaml.Node
	index      *index.SpecIndex
	context    context.Context
	*low.Reference
	low.NodeMap
}

// GetIndex returns the index.SpecIndex instance attached to the CriterionExpressionType object.
// For Arazzo low models this is typically nil, because Arazzo parsing does not build a SpecIndex.
// The index parameter is still required to satisfy the shared low.Buildable interface and generic extractors.
func (c *ExpressionType) GetIndex() *index.SpecIndex {
	return c.index
}

// GetContext returns the context.Context instance used when building the CriterionExpressionType object.
func (c *ExpressionType) GetContext() context.Context {
	return c.context
}

// FindExtension returns a ValueReference containing the extension value, if found.
func (c *ExpressionType) FindExtension(ext string) *low.ValueReference[*yaml.Node] {
	return low.FindItemInOrderedMap(ext, c.Extensions)
}

// GetRootNode returns the root yaml node of the CriterionExpressionType object.
func (c *ExpressionType) GetRootNode() *yaml.Node {
	return c.RootNode
}

// GetKeyNode returns the key yaml node of the CriterionExpressionType object.
func (c *ExpressionType) GetKeyNode() *yaml.Node {
	return c.KeyNode
}

// Build will extract all properties of the CriterionExpressionType object.
func (c *ExpressionType) Build(ctx context.Context, keyNode, root *yaml.Node, idx *index.SpecIndex) error {
	root = initBuild(&arazzoBase{
		KeyNode:    &c.KeyNode,
		RootNode:   &c.RootNode,
		Reference:  &c.Reference,
		NodeMap:    &c.NodeMap,
		Extensions: &c.Extensions,
		Index:      &c.index,
		Context:    &c.context,
	}, ctx, keyNode, root, idx)
	return nil
}

// GetExtensions returns all CriterionExpressionType extensions and satisfies the low.HasExtensions interface.
func (c *ExpressionType) GetExtensions() *orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]] {
	return c.Extensions
}

// Hash will return a consistent hash of the CriterionExpressionType object.
func (c *ExpressionType) Hash() uint64 {
	return low.WithHasher(func(h *maphash.Hash) uint64 {
		if !c.Type.IsEmpty() {
			h.WriteString(c.Type.Value)
			h.WriteByte(low.HASH_PIPE)
		}
		if !c.Version.IsEmpty() {
			h.WriteString(c.Version.Value)
			h.WriteByte(low.HASH_PIPE)
		}
		hashExtensionsInto(h, c.Extensions)
		return h.Sum64()
	})
}

// CriterionExpressionType is retained as an alias for source compatibility with Arazzo 1.0 callers.
type CriterionExpressionType = ExpressionType
