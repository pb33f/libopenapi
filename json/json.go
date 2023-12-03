package json

import (
	"encoding/json"
	"fmt"

	"github.com/pb33f/libopenapi/orderedmap"
	"gopkg.in/yaml.v3"
)

// YAMLNodeToJSON converts yaml/json stored in a yaml.Node to json ordered matching the original yaml/json
//
// NOTE: The limitation is this won't work with YAML that is not compatible with JSON, ie yaml with anchors or complex map keys
func YAMLNodeToJSON(node *yaml.Node, indentation string) ([]byte, error) {
	v, err := handleYAMLNode(node)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(v, "", indentation)
}

func handleYAMLNode(node *yaml.Node) (any, error) {
	switch node.Kind {
	case yaml.DocumentNode:
		return handleYAMLNode(node.Content[0])
	case yaml.SequenceNode:
		return handleSequenceNode(node)
	case yaml.MappingNode:
		return handleMappingNode(node)
	case yaml.ScalarNode:
		return handleScalarNode(node)
	case yaml.AliasNode:
		panic("currently unsupported")
	default:
		return nil, fmt.Errorf("unknown node kind: %v", node.Kind)
	}
}

func handleMappingNode(node *yaml.Node) (any, error) {
	m := orderedmap.New[string, yaml.Node]()

	if err := node.Decode(m); err != nil {
		return nil, err
	}

	v := orderedmap.New[string, any]()
	for pair := orderedmap.First(m); pair != nil; pair = pair.Next() {
		n := pair.Value()
		vv, err := handleYAMLNode(&n)
		if err != nil {
			return nil, err
		}

		v.Set(pair.Key(), vv)
	}

	return v, nil
}

func handleSequenceNode(node *yaml.Node) (any, error) {
	var s []yaml.Node

	if err := node.Decode(&s); err != nil {
		return nil, err
	}

	v := make([]any, len(s))
	for i, n := range s {
		vv, err := handleYAMLNode(&n)
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
