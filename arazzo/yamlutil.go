// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"fmt"
	"sort"

	"go.yaml.in/yaml/v4"
)

func toYAMLNode(value any) (*yaml.Node, error) {
	if value == nil {
		return nil, nil
	}
	if node, ok := value.(*yaml.Node); ok {
		return node, nil
	}
	return directYAMLNode(value)
}

func directYAMLNode(value any) (*yaml.Node, error) {
	switch typed := value.(type) {
	case *yaml.Node:
		return typed, nil
	case yaml.Node:
		return &typed, nil
	case map[string]any:
		node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		keys := make([]string, 0, len(typed))
		for k := range typed {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: k}
			valueNode, err := directYAMLNode(typed[k])
			if err != nil {
				return nil, err
			}
			node.Content = append(node.Content, keyNode, valueNode)
		}
		return node, nil
	case map[any]any:
		node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		keys := make([]string, 0, len(typed))
		keyMap := make(map[string]any, len(typed))
		for k, v := range typed {
			ks := fmt.Sprint(k)
			keys = append(keys, ks)
			keyMap[ks] = v
		}
		sort.Strings(keys)
		for _, k := range keys {
			keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: k}
			valueNode, err := directYAMLNode(keyMap[k])
			if err != nil {
				return nil, err
			}
			node.Content = append(node.Content, keyNode, valueNode)
		}
		return node, nil
	case []any:
		node := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for _, item := range typed {
			itemNode, err := directYAMLNode(item)
			if err != nil {
				return nil, err
			}
			node.Content = append(node.Content, itemNode)
		}
		return node, nil
	case []string:
		node := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for _, item := range typed {
			node.Content = append(node.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: item})
		}
		return node, nil
	case string:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: typed}, nil
	case bool:
		if typed {
			return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"}, nil
		}
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "false"}, nil
	case int, int8, int16, int32, int64:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: fmt.Sprint(typed)}, nil
	case uint, uint8, uint16, uint32, uint64:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: fmt.Sprint(typed)}, nil
	case float32, float64:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!float", Value: fmt.Sprint(typed)}, nil
	case nil:
		return nil, nil
	default:
		node := &yaml.Node{}
		if err := node.Encode(value); err != nil {
			return nil, err
		}
		return node, nil
	}
}
