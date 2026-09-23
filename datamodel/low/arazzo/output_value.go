// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"context"
	"fmt"
	"hash/maphash"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"go.yaml.in/yaml/v4"
)

// OutputValue represents the scalar-runtime-expression or Selector Object output union.
// https://spec.openapis.org/arazzo/v1.1.0#workflow-object
type OutputValue struct {
	Expression low.NodeReference[string]
	Selector   low.NodeReference[*Selector]
	KeyNode    *yaml.Node
	RootNode   *yaml.Node
	index      *index.SpecIndex
	context    context.Context
}

// GetIndex returns the index attached to the OutputValue.
func (o *OutputValue) GetIndex() *index.SpecIndex {
	return o.index
}

// GetContext returns the context used to build the OutputValue.
func (o *OutputValue) GetContext() context.Context {
	return o.context
}

// GetRootNode returns the scalar or mapping node that defines the output.
func (o *OutputValue) GetRootNode() *yaml.Node {
	return o.RootNode
}

// GetKeyNode returns the output-name key node.
func (o *OutputValue) GetKeyNode() *yaml.Node {
	return o.KeyNode
}

// IsExpression reports whether the output contains a runtime-expression scalar.
func (o *OutputValue) IsExpression() bool {
	return o != nil && !o.Expression.IsEmpty() && o.Selector.IsEmpty()
}

// IsSelector reports whether the output contains a Selector Object.
func (o *OutputValue) IsSelector() bool {
	return o != nil && o.Expression.IsEmpty() && !o.Selector.IsEmpty()
}

// Build parses exactly one permitted output union variant.
func (o *OutputValue) Build(ctx context.Context, keyNode, root *yaml.Node, idx *index.SpecIndex) error {
	o.KeyNode = keyNode
	o.RootNode = root
	o.index = idx
	o.context = ctx
	resolved, err := resolveAliasNode(root)
	if err != nil {
		return err
	}
	switch resolved.Kind {
	case yaml.ScalarNode:
		o.Expression = low.NodeReference[string]{
			Value:     resolved.Value,
			KeyNode:   keyNode,
			ValueNode: root,
		}
	case yaml.MappingNode:
		selector := new(Selector)
		_ = low.BuildModel(resolved, selector)
		if err := selector.Build(ctx, keyNode, resolved, idx); err != nil {
			return err
		}
		o.Selector = low.NodeReference[*Selector]{
			Value:     selector,
			KeyNode:   keyNode,
			ValueNode: root,
		}
	default:
		return fmt.Errorf("output value at line %d, column %d must be a scalar or Selector Object",
			root.Line, root.Column)
	}
	return nil
}

// Hash returns a deterministic hash of the active output variant.
func (o *OutputValue) Hash() uint64 {
	return low.WithHasher(func(h *maphash.Hash) uint64 {
		if o.IsExpression() {
			h.WriteString(o.Expression.Value)
			h.WriteByte(low.HASH_PIPE)
		}
		if o.IsSelector() {
			low.HashUint64(h, o.Selector.Value.Hash())
		}
		return h.Sum64()
	})
}
