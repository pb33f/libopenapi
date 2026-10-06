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

// Overlay represents a low-level OpenAPI Overlay document.
// https://spec.openapis.org/overlay/v1.2.0
type Overlay struct {
	Overlay    low.NodeReference[string]
	Self       low.NodeReference[string]
	Components low.NodeReference[*Components]
	Info       low.NodeReference[*Info]
	Extends    low.NodeReference[string]
	Actions    low.NodeReference[[]low.ValueReference[*Action]]
	Extensions *orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]]
	KeyNode    *yaml.Node
	RootNode   *yaml.Node
	index      *index.SpecIndex
	context    context.Context
	*low.Reference
	low.NodeMap
}

// GetIndex returns the index.SpecIndex instance attached to the Overlay object
func (o *Overlay) GetIndex() *index.SpecIndex {
	return o.index
}

// GetContext returns the context.Context instance used when building the Overlay object
func (o *Overlay) GetContext() context.Context {
	return o.context
}

// FindExtension returns a ValueReference containing the extension value, if found.
func (o *Overlay) FindExtension(ext string) *low.ValueReference[*yaml.Node] {
	return low.FindItemInOrderedMap(ext, o.Extensions)
}

// GetRootNode returns the root yaml node of the Overlay object
func (o *Overlay) GetRootNode() *yaml.Node {
	return o.RootNode
}

// GetKeyNode returns the key yaml node of the Overlay object
func (o *Overlay) GetKeyNode() *yaml.Node {
	return o.KeyNode
}

// Build will extract all properties of the Overlay document.
func (o *Overlay) Build(ctx context.Context, keyNode, root *yaml.Node, idx *index.SpecIndex) error {
	if err := requireMapping(root); err != nil {
		return err
	}
	o.KeyNode = keyNode
	root = utils.NodeAlias(root)
	o.RootNode = root
	utils.CheckForMergeNodes(root)
	o.Reference = new(low.Reference)
	o.Nodes = low.ExtractNodes(ctx, root)
	o.Extensions = low.ExtractExtensions(root)
	o.index = idx
	o.context = ctx
	low.ExtractExtensionNodes(ctx, o.Extensions, o.Nodes)

	var err error
	o.Overlay, err = extractString(OverlayLabel, root)
	if err != nil {
		return err
	}
	o.Extends, err = extractString(ExtendsLabel, root)
	if err != nil {
		return err
	}
	o.Self, err = extractString(SelfLabel, root)
	if err != nil {
		return err
	}
	o.Components, err = extractObject[*Components](ctx, ComponentsLabel, root, idx)
	if err != nil {
		return err
	}

	// Extract info object
	info, err := extractObject[*Info](ctx, InfoLabel, root, idx)
	if err != nil {
		return err
	}
	o.Info = info

	// Extract actions array
	o.Actions, err = o.extractActions(ctx, root, idx)
	return err
}

func (o *Overlay) extractActions(ctx context.Context, root *yaml.Node, idx *index.SpecIndex) (low.NodeReference[[]low.ValueReference[*Action]], error) {
	var result low.NodeReference[[]low.ValueReference[*Action]]

	key, value := findField(ActionsLabel, root)
	if value == nil {
		return result, nil
	}
	result.KeyNode, result.ValueNode = key, value
	if value.Kind != yaml.SequenceNode {
		return result, fmt.Errorf("actions must be an array")
	}
	actions := make([]low.ValueReference[*Action], 0, len(value.Content))
	for _, actionNode := range value.Content {
		action := new(Action)
		if err := action.Build(ctx, nil, actionNode, idx); err != nil {
			return result, err
		}
		actions = append(actions, low.ValueReference[*Action]{Value: action, ValueNode: actionNode})
	}
	result.Value = actions
	return result, nil
}

// GetExtensions returns all Overlay extensions and satisfies the low.HasExtensions interface.
func (o *Overlay) GetExtensions() *orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]] {
	return o.Extensions
}

// Hash will return a consistent Hash of the Overlay object
func (o *Overlay) Hash() uint64 {
	return low.WithHasher(func(h *maphash.Hash) uint64 {
		if !o.Overlay.IsEmpty() {
			h.WriteString(o.Overlay.Value)
			h.WriteByte(low.HASH_PIPE)
		}
		if !o.Self.IsEmpty() {
			h.WriteString(SelfLabel)
			h.WriteString(o.Self.Value)
			h.WriteByte(low.HASH_PIPE)
		}
		if !o.Components.IsEmpty() {
			h.WriteString(ComponentsLabel)
			h.WriteString(low.GenerateHashString(o.Components.Value))
			h.WriteByte(low.HASH_PIPE)
		}
		if !o.Info.IsEmpty() {
			h.WriteString(low.GenerateHashString(o.Info.Value))
			h.WriteByte(low.HASH_PIPE)
		}
		if !o.Extends.IsEmpty() {
			h.WriteString(o.Extends.Value)
			h.WriteByte(low.HASH_PIPE)
		}
		if !o.Actions.IsEmpty() {
			for _, action := range o.Actions.Value {
				h.WriteString(low.GenerateHashString(action.Value))
				h.WriteByte(low.HASH_PIPE)
			}
		}
		for _, ext := range low.HashExtensions(o.Extensions) {
			h.WriteString(ext)
			h.WriteByte(low.HASH_PIPE)
		}
		return h.Sum64()
	})
}
