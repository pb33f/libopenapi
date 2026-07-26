// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"context"

	"github.com/pb33f/libopenapi/datamodel/low"
	lowarazzo "github.com/pb33f/libopenapi/datamodel/low/arazzo"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

// buildSlice converts a slice of low.ValueReference[L] to a slice of H using a conversion function.
func buildSlice[L any, H any](refs []low.ValueReference[L], convert func(L) H) []H {
	if len(refs) == 0 {
		return nil
	}
	out := make([]H, 0, len(refs))
	for _, ref := range refs {
		out = append(out, convert(ref.Value))
	}
	return out
}

func dereferenceAliasNode(node *yaml.Node) *yaml.Node {
	seen := make(map[*yaml.Node]struct{})
	for node != nil && node.Kind == yaml.AliasNode {
		if _, exists := seen[node]; exists || node.Alias == nil {
			return nil
		}
		seen[node] = struct{}{}
		node = node.Alias
	}
	return node
}

func selectorFromNode(node *yaml.Node) *Selector {
	node = dereferenceAliasNode(node)
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	var hasContext, hasSelector, hasType bool
	for i := 0; i+1 < len(node.Content); i += 2 {
		value := dereferenceAliasNode(node.Content[i+1])
		if value == nil {
			return nil
		}
		switch node.Content[i].Value {
		case lowarazzo.ContextLabel:
			if value.Kind != yaml.ScalarNode {
				return nil
			}
			hasContext = true
		case lowarazzo.SelectorLabel:
			if value.Kind != yaml.ScalarNode {
				return nil
			}
			hasSelector = true
		case lowarazzo.TypeLabel:
			hasType = true
		}
	}
	if !hasContext || !hasSelector || !hasType {
		return nil
	}
	selector := new(lowarazzo.Selector)
	_ = low.BuildModel(node, selector)
	if err := selector.Build(context.Background(), nil, node, nil); err != nil {
		return nil
	}
	return NewSelector(selector)
}

func selectorsFromNode(node *yaml.Node) []*Selector {
	return selectorsFromNodePath(node, make(map[*yaml.Node]struct{}))
}

func selectorsFromNodePath(node *yaml.Node, active map[*yaml.Node]struct{}) []*Selector {
	node = dereferenceAliasNode(node)
	if node == nil {
		return nil
	}
	if _, exists := active[node]; exists {
		return nil
	}
	active[node] = struct{}{}
	defer delete(active, node)
	if selector := selectorFromNode(node); selector != nil {
		return []*Selector{selector}
	}
	var selectors []*Selector
	switch node.Kind {
	case yaml.DocumentNode, yaml.SequenceNode:
		for _, child := range node.Content {
			selectors = append(selectors, selectorsFromNodePath(child, active)...)
		}
	case yaml.MappingNode:
		for i := 1; i < len(node.Content); i += 2 {
			selectors = append(selectors, selectorsFromNodePath(node.Content[i], active)...)
		}
	}
	return selectors
}

func buildOutputValueMap(
	refs *orderedmap.Map[low.KeyReference[string], low.ValueReference[*lowarazzo.OutputValue]],
) *orderedmap.Map[string, *OutputValue] {
	if refs == nil {
		return nil
	}
	outputs := orderedmap.New[string, *OutputValue]()
	for pair := refs.First(); pair != nil; pair = pair.Next() {
		outputs.Set(pair.Key().Value, NewOutputValue(pair.Value().Value))
	}
	return outputs
}

// buildValueSlice extracts the Value from each low.ValueReference into a plain slice.
func buildValueSlice[T any](refs []low.ValueReference[T]) []T {
	if len(refs) == 0 {
		return nil
	}
	out := make([]T, 0, len(refs))
	for _, ref := range refs {
		out = append(out, ref.Value)
	}
	return out
}
