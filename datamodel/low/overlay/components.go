// Copyright 2022-2025 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package overlay

import (
	"context"
	"fmt"
	"hash/maphash"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
)

// Components represents a low-level Overlay Components Object.
// https://spec.openapis.org/overlay/v1.2.0#components-object
type Components struct {
	Actions    low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[*ReusableAction]]]
	Extensions *orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]]
	KeyNode    *yaml.Node
	RootNode   *yaml.Node
	index      *index.SpecIndex
	context    context.Context
	*low.Reference
	low.NodeMap
}

// GetIndex returns the index.SpecIndex instance attached to the Components object
func (i *Components) GetIndex() *index.SpecIndex {
	return i.index
}

// GetContext returns the context.Context instance used when building the Components object
func (i *Components) GetContext() context.Context {
	return i.context
}

// FindExtension returns a ValueReference containing the extension value, if found.
func (i *Components) FindExtension(ext string) *low.ValueReference[*yaml.Node] {
	return low.FindItemInOrderedMap(ext, i.Extensions)
}

// GetRootNode returns the root yaml node of the Components object
func (i *Components) GetRootNode() *yaml.Node {
	return i.RootNode
}

// GetKeyNode returns the key yaml node of the Components object
func (i *Components) GetKeyNode() *yaml.Node {
	return i.KeyNode
}

// Build extracts reusable actions and extensions.
func (i *Components) Build(ctx context.Context, keyNode, root *yaml.Node, idx *index.SpecIndex) error {
	if err := requireMapping(root); err != nil {
		return err
	}
	i.KeyNode = keyNode
	root = utils.NodeAlias(root)
	i.RootNode = root
	utils.CheckForMergeNodes(root)
	i.Reference = new(low.Reference)
	i.Nodes = low.ExtractNodes(ctx, root)
	i.Extensions = low.ExtractExtensions(root)
	i.index = idx
	i.context = ctx
	low.ExtractExtensionNodes(ctx, i.Extensions, i.Nodes)
	key, value := findField(ActionsLabel, root)
	if value != nil {
		if err := requireMapping(value); err != nil {
			return err
		}
		actions := orderedmap.New[low.KeyReference[string], low.ValueReference[*ReusableAction]]()
		names := make(map[string]struct{}, len(value.Content)/2)
		for n := 0; n < len(value.Content); n += 2 {
			k, v := value.Content[n], value.Content[n+1]
			if k.Kind != yaml.ScalarNode {
				return fmt.Errorf("component action name must be a scalar")
			}
			if _, exists := names[k.Value]; exists {
				return fmt.Errorf("duplicate component action %q", k.Value)
			}
			names[k.Value] = struct{}{}
			action := new(ReusableAction)
			if err := action.Build(ctx, k, v, idx); err != nil {
				return err
			}
			actions.Set(low.KeyReference[string]{Value: k.Value, KeyNode: k}, low.ValueReference[*ReusableAction]{Value: action, ValueNode: v})
		}
		i.Actions = low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[*ReusableAction]]]{Value: actions, KeyNode: key, ValueNode: value}
	}
	return nil
}

// GetExtensions returns all Components extensions and satisfies the low.HasExtensions interface.
func (i *Components) GetExtensions() *orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]] {
	return i.Extensions
}

// Hash will return a consistent Hash of the Components object
func (inf *Components) Hash() uint64 {
	return low.WithHasher(func(h *maphash.Hash) uint64 {
		for pair := orderedmap.SortAlpha(inf.Actions.Value).First(); pair != nil; pair = pair.Next() {
			h.WriteString(pair.Key().Value)
			h.WriteByte(low.HASH_PIPE)
			h.WriteString(low.GenerateHashString(pair.Value().Value))
			h.WriteByte(low.HASH_PIPE)
		}
		for _, ext := range low.HashExtensions(inf.Extensions) {
			h.WriteString(ext)
			h.WriteByte(low.HASH_PIPE)
		}
		return h.Sum64()
	})
}
