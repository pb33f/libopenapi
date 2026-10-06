// Copyright 2022-2025 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package overlay

import (
	"context"
	"hash/maphash"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
)

// ReusableAction represents a low-level Overlay ReusableAction Object.
// https://spec.openapis.org/overlay/v1.2.0#reusable-action-object
type ReusableAction struct {
	Description low.NodeReference[string]
	Fields      low.NodeReference[*Action]
	Extensions  *orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]]
	KeyNode     *yaml.Node
	RootNode    *yaml.Node
	index       *index.SpecIndex
	context     context.Context
	*low.Reference
	low.NodeMap
}

// GetIndex returns the index.SpecIndex instance attached to the ReusableAction object
func (i *ReusableAction) GetIndex() *index.SpecIndex {
	return i.index
}

// GetContext returns the context.Context instance used when building the ReusableAction object
func (i *ReusableAction) GetContext() context.Context {
	return i.context
}

// FindExtension returns a ValueReference containing the extension value, if found.
func (i *ReusableAction) FindExtension(ext string) *low.ValueReference[*yaml.Node] {
	return low.FindItemInOrderedMap(ext, i.Extensions)
}

// GetRootNode returns the root yaml node of the ReusableAction object
func (i *ReusableAction) GetRootNode() *yaml.Node {
	return i.RootNode
}

// GetKeyNode returns the key yaml node of the ReusableAction object
func (i *ReusableAction) GetKeyNode() *yaml.Node {
	return i.KeyNode
}

// Build extracts action fields and extensions.
func (i *ReusableAction) Build(ctx context.Context, keyNode, root *yaml.Node, idx *index.SpecIndex) error {
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
	var err error
	i.Description, err = extractString(DescriptionLabel, root)
	if err != nil {
		return err
	}
	i.Fields, err = extractObject[*Action](ctx, FieldsLabel, root, idx)
	return err
}

// GetExtensions returns all ReusableAction extensions and satisfies the low.HasExtensions interface.
func (i *ReusableAction) GetExtensions() *orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]] {
	return i.Extensions
}

// Hash will return a consistent Hash of the ReusableAction object
func (inf *ReusableAction) Hash() uint64 {
	return low.WithHasher(func(h *maphash.Hash) uint64 {
		if !inf.Description.IsEmpty() {
			h.WriteString(inf.Description.Value)
			h.WriteByte(low.HASH_PIPE)
		}
		if !inf.Fields.IsEmpty() {
			h.WriteString(low.GenerateHashString(inf.Fields.Value))
			h.WriteByte(low.HASH_PIPE)
		}
		for _, ext := range low.HashExtensions(inf.Extensions) {
			h.WriteString(ext)
			h.WriteByte(low.HASH_PIPE)
		}
		return h.Sum64()
	})
}
