// Copyright 2022-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package overlay

import (
	"context"
	"fmt"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
)

func requireMapping(node *yaml.Node) error {
	node = utils.NodeAlias(node)
	if node == nil || node.Kind != yaml.MappingNode || len(node.Content)%2 != 0 {
		return fmt.Errorf("overlay object must be a mapping")
	}
	return nil
}

func findField(label string, root *yaml.Node) (*yaml.Node, *yaml.Node) {
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == label {
			return root.Content[i], utils.NodeAlias(root.Content[i+1])
		}
	}
	return nil, nil
}

func extractString(label string, root *yaml.Node) (low.NodeReference[string], error) {
	key, value := findField(label, root)
	if value == nil {
		return low.NodeReference[string]{}, nil
	}
	if value.Kind != yaml.ScalarNode || value.Tag != "!!str" {
		return low.NodeReference[string]{}, fmt.Errorf("%s must be a string", label)
	}
	return low.NodeReference[string]{Value: value.Value, KeyNode: key, ValueNode: value}, nil
}

// Overlay references are resolved by the overlay engine, never by the OpenAPI index.
func extractObject[T low.Buildable[N], N any](ctx context.Context, label string, root *yaml.Node, idx *index.SpecIndex) (low.NodeReference[T], error) {
	var result low.NodeReference[T]
	key, value := findField(label, root)
	if value == nil {
		return result, nil
	}
	var model T = new(N)
	if err := model.Build(ctx, key, value, idx); err != nil {
		return result, err
	}
	return low.NodeReference[T]{Value: model, KeyNode: key, ValueNode: value}, nil
}
