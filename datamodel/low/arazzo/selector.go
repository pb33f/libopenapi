// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"context"
	"fmt"
	"hash/maphash"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

// Selector represents a low-level Arazzo Selector Object.
// https://spec.openapis.org/arazzo/v1.1.0#selector-object
type Selector struct {
	Context    low.NodeReference[string]
	Selector   low.NodeReference[string]
	Type       low.NodeReference[*yaml.Node]
	Extensions *orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]]
	KeyNode    *yaml.Node
	RootNode   *yaml.Node
	index      *index.SpecIndex
	context    context.Context
	*low.Reference
	low.NodeMap
}

// GetIndex returns the index attached to the Selector.
func (s *Selector) GetIndex() *index.SpecIndex {
	return s.index
}

// GetContext returns the context used to build the Selector.
func (s *Selector) GetContext() context.Context {
	return s.context
}

// FindExtension returns a Selector extension when present.
func (s *Selector) FindExtension(ext string) *low.ValueReference[*yaml.Node] {
	return low.FindItemInOrderedMap(ext, s.Extensions)
}

// GetRootNode returns the Selector mapping node.
func (s *Selector) GetRootNode() *yaml.Node {
	return s.RootNode
}

// GetKeyNode returns the key node associated with the Selector.
func (s *Selector) GetKeyNode() *yaml.Node {
	return s.KeyNode
}

// Build extracts the Selector fields while preserving its scalar-or-object type union.
func (s *Selector) Build(ctx context.Context, keyNode, root *yaml.Node, idx *index.SpecIndex) error {
	root = initBuild(&arazzoBase{
		KeyNode:    &s.KeyNode,
		RootNode:   &s.RootNode,
		Reference:  &s.Reference,
		NodeMap:    &s.NodeMap,
		Extensions: &s.Extensions,
		Index:      &s.index,
		Context:    &s.context,
	}, ctx, keyNode, root, idx)

	var err error
	s.Context, err = extractScalarString(ContextLabel, root)
	if err != nil {
		return err
	}
	s.Selector, err = extractScalarString(SelectorLabel, root)
	if err != nil {
		return err
	}
	s.Type = extractRawNode(TypeLabel, root)
	if s.Type.IsEmpty() {
		return nil
	}
	resolvedType, err := resolveAliasNode(s.Type.Value)
	if err != nil {
		return fmt.Errorf("selector type: %w", err)
	}
	if resolvedType.Kind != yaml.ScalarNode && resolvedType.Kind != yaml.MappingNode {
		return fmt.Errorf("selector type at line %d, column %d must be a scalar or mapping",
			s.Type.Value.Line, s.Type.Value.Column)
	}
	return nil
}

// GetExtensions returns all Selector extensions.
func (s *Selector) GetExtensions() *orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]] {
	return s.Extensions
}

// Hash returns a deterministic hash of every Selector field.
func (s *Selector) Hash() uint64 {
	return low.WithHasher(func(h *maphash.Hash) uint64 {
		if !s.Context.IsEmpty() {
			h.WriteString(s.Context.Value)
			h.WriteByte(low.HASH_PIPE)
		}
		if !s.Selector.IsEmpty() {
			h.WriteString(s.Selector.Value)
			h.WriteByte(low.HASH_PIPE)
		}
		if !s.Type.IsEmpty() {
			hashYAMLNode(h, s.Type.Value)
		}
		hashExtensionsInto(h, s.Extensions)
		return h.Sum64()
	})
}
