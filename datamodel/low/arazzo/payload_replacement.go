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

// PayloadReplacement represents a low-level Arazzo Payload Replacement Object.
// https://spec.openapis.org/arazzo/v1.1.0#payload-replacement-object
type PayloadReplacement struct {
	Target             low.NodeReference[string]
	TargetSelectorType low.NodeReference[*yaml.Node]
	Value              low.NodeReference[*yaml.Node]
	Extensions         *orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]]
	KeyNode            *yaml.Node
	RootNode           *yaml.Node
	index              *index.SpecIndex
	context            context.Context
	*low.Reference
	low.NodeMap
}

// GetIndex returns the index.SpecIndex instance attached to the PayloadReplacement object.
// For Arazzo low models this is typically nil, because Arazzo parsing does not build a SpecIndex.
// The index parameter is still required to satisfy the shared low.Buildable interface and generic extractors.
func (p *PayloadReplacement) GetIndex() *index.SpecIndex {
	return p.index
}

// GetContext returns the context.Context instance used when building the PayloadReplacement object.
func (p *PayloadReplacement) GetContext() context.Context {
	return p.context
}

// FindExtension returns a ValueReference containing the extension value, if found.
func (p *PayloadReplacement) FindExtension(ext string) *low.ValueReference[*yaml.Node] {
	return low.FindItemInOrderedMap(ext, p.Extensions)
}

// GetRootNode returns the root yaml node of the PayloadReplacement object.
func (p *PayloadReplacement) GetRootNode() *yaml.Node {
	return p.RootNode
}

// GetKeyNode returns the key yaml node of the PayloadReplacement object.
func (p *PayloadReplacement) GetKeyNode() *yaml.Node {
	return p.KeyNode
}

// Build will extract all properties of the PayloadReplacement object.
func (p *PayloadReplacement) Build(ctx context.Context, keyNode, root *yaml.Node, idx *index.SpecIndex) error {
	root = initBuild(&arazzoBase{
		KeyNode:    &p.KeyNode,
		RootNode:   &p.RootNode,
		Reference:  &p.Reference,
		NodeMap:    &p.NodeMap,
		Extensions: &p.Extensions,
		Index:      &p.index,
		Context:    &p.context,
	}, ctx, keyNode, root, idx)

	var err error
	p.Target, err = extractScalarString(TargetLabel, root)
	if err != nil {
		return err
	}
	p.Value = extractRawNode(ValueLabel, root)
	p.TargetSelectorType = extractRawNode(TargetSelectorTypeLabel, root)
	if p.TargetSelectorType.IsEmpty() {
		return nil
	}
	resolvedType, err := resolveAliasNode(p.TargetSelectorType.Value)
	if err != nil {
		return fmt.Errorf("targetSelectorType: %w", err)
	}
	if resolvedType.Kind != yaml.ScalarNode && resolvedType.Kind != yaml.MappingNode {
		return fmt.Errorf("targetSelectorType at line %d, column %d must be a scalar or mapping",
			p.TargetSelectorType.Value.Line, p.TargetSelectorType.Value.Column)
	}
	return nil
}

// GetExtensions returns all PayloadReplacement extensions and satisfies the low.HasExtensions interface.
func (p *PayloadReplacement) GetExtensions() *orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]] {
	return p.Extensions
}

// Hash will return a consistent hash of the PayloadReplacement object.
func (p *PayloadReplacement) Hash() uint64 {
	return low.WithHasher(func(h *maphash.Hash) uint64 {
		if !p.Target.IsEmpty() {
			h.WriteString(p.Target.Value)
			h.WriteByte(low.HASH_PIPE)
		}
		if !p.TargetSelectorType.IsEmpty() {
			hashYAMLNode(h, p.TargetSelectorType.Value)
		}
		if !p.Value.IsEmpty() {
			hashYAMLNode(h, p.Value.Value)
		}
		hashExtensionsInto(h, p.Extensions)
		return h.Sum64()
	})
}
