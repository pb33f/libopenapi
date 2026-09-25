package json

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

// YAMLNodeToJSON converts yaml/json stored in a yaml.Node to json ordered matching the original yaml/json
func YAMLNodeToJSON(node *yaml.Node, indentation string) ([]byte, error) {
	c := converter{aliasesInFlight: make(map[*yaml.Node]struct{})}
	v, err := c.handleYAMLNode(node)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(v, "", indentation)
}

// converter tracks alias targets currently being expanded, so a self-referencing anchor
// (e.g. `a: &x [1, *x]`) is reported as an error instead of recursing forever.
type converter struct {
	aliasesInFlight map[*yaml.Node]struct{}
}

func (c converter) handleYAMLNode(node *yaml.Node) (any, error) {
	if node == nil {
		return nil, errors.New("nil yaml node")
	}
	switch node.Kind {
	case yaml.DocumentNode:
		if len(node.Content) == 0 {
			return nil, errors.New("empty yaml document")
		}
		return c.handleYAMLNode(node.Content[0])
	case yaml.SequenceNode:
		return c.handleSequenceNode(node)
	case yaml.MappingNode:
		return c.handleMappingNode(node)
	case yaml.ScalarNode:
		return handleScalarNode(node)
	case yaml.AliasNode:
		return c.handleAliasNode(node)
	default:
		return nil, fmt.Errorf("unknown node kind: %v", node.Kind)
	}
}

func (c converter) handleAliasNode(node *yaml.Node) (any, error) {
	if _, inFlight := c.aliasesInFlight[node.Alias]; inFlight {
		return nil, fmt.Errorf("recursive alias '%s' at line %d, column %d", node.Value, node.Line, node.Column)
	}
	c.aliasesInFlight[node.Alias] = struct{}{}
	defer delete(c.aliasesInFlight, node.Alias)
	return c.handleYAMLNode(node.Alias)
}

func (c converter) handleMappingNode(node *yaml.Node) (any, error) {
	v := orderedmap.New[string, any]()
	for i, n := range node.Content {
		if i%2 == 0 {
			continue
		}
		keyNode := node.Content[i-1]
		kv, err := c.handleYAMLNode(keyNode)
		if err != nil {
			return nil, err
		}

		key, isString := kv.(string)
		if !isString {
			keyData, err := json.Marshal(kv)
			if err != nil {
				return nil, err
			}
			key = string(keyData)
		}

		vv, err := c.handleYAMLNode(n)
		if err != nil {
			return nil, err
		}

		v.Set(key, vv)
	}

	return v, nil
}

func (c converter) handleSequenceNode(node *yaml.Node) (any, error) {
	v := make([]any, len(node.Content))
	for i, n := range node.Content {
		vv, err := c.handleYAMLNode(n)
		if err != nil {
			return nil, err
		}

		v[i] = vv
	}

	return v, nil
}

func handleScalarNode(node *yaml.Node) (any, error) {
	var v any

	if err := node.Decode(&v); err != nil {
		return nil, err
	}

	return v, nil
}
