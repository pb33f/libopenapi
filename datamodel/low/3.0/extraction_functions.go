package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"strconv"
	"sync"
)

func ExtractSchema(root *yaml.Node) (*low.NodeReference[*Schema], error) {
	_, schLabel, schNode := utils.FindKeyNodeFull(SchemaLabel, root.Content)
	if schNode != nil {
		var schema Schema
		err := BuildModel(schNode, &schema)
		if err != nil {
			return nil, err
		}
		err = schema.Build(schNode, 0)
		if err != nil {
			return nil, err
		}
		return &low.NodeReference[*Schema]{Value: &schema, KeyNode: schLabel, ValueNode: schNode}, nil
	}
	return nil, nil
}

var mapLock sync.Mutex

func ExtractMap[PT low.Buildable[N], N any](label string, root *yaml.Node) (map[low.KeyReference[string]]map[low.KeyReference[string]]low.ValueReference[PT], error) {
	_, labelNode, valueNode := utils.FindKeyNodeFull(label, root.Content)
	if valueNode != nil {
		var currentLabelNode *yaml.Node
		valueMap := make(map[low.KeyReference[string]]low.ValueReference[PT])
		for i, en := range valueNode.Content {
			if i%2 == 0 {
				currentLabelNode = en
				continue
			}
			var n PT = new(N)
			err := BuildModel(valueNode, n)
			if err != nil {
				return nil, err
			}
			berr := n.Build(valueNode)
			if berr != nil {
				return nil, berr
			}
			valueMap[low.KeyReference[string]{
				Value:   currentLabelNode.Value,
				KeyNode: currentLabelNode,
			}] = low.ValueReference[PT]{
				Value:     n,
				ValueNode: en,
			}
		}

		resMap := make(map[low.KeyReference[string]]map[low.KeyReference[string]]low.ValueReference[PT])
		resMap[low.KeyReference[string]{
			Value:   labelNode.Value,
			KeyNode: labelNode,
		}] = valueMap
		return resMap, nil
	}
	return nil, nil
}

func ExtractExtensions(root *yaml.Node) (map[low.KeyReference[string]]low.ValueReference[any], error) {
	extensions := utils.FindExtensionNodes(root.Content)
	extensionMap := make(map[low.KeyReference[string]]low.ValueReference[any])
	for _, ext := range extensions {
		if utils.IsNodeMap(ext.Value) {
			var v interface{}
			err := ext.Value.Decode(&v)
			if err != nil {
				return nil, err
			}
			extensionMap[low.KeyReference[string]{
				Value:   ext.Key.Value,
				KeyNode: ext.Key,
			}] = low.ValueReference[any]{Value: v, ValueNode: ext.Value}
		}
		if utils.IsNodeStringValue(ext.Value) {
			extensionMap[low.KeyReference[string]{
				Value:   ext.Key.Value,
				KeyNode: ext.Key,
			}] = low.ValueReference[any]{Value: ext.Value.Value, ValueNode: ext.Value}
		}
		if utils.IsNodeFloatValue(ext.Value) {
			fv, _ := strconv.ParseFloat(ext.Value.Value, 64)
			extensionMap[low.KeyReference[string]{
				Value:   ext.Key.Value,
				KeyNode: ext.Key,
			}] = low.ValueReference[any]{Value: fv, ValueNode: ext.Value}
		}
		if utils.IsNodeIntValue(ext.Value) {
			iv, _ := strconv.ParseInt(ext.Value.Value, 10, 64)
			extensionMap[low.KeyReference[string]{
				Value:   ext.Key.Value,
				KeyNode: ext.Key,
			}] = low.ValueReference[any]{Value: iv, ValueNode: ext.Value}
		}
		if utils.IsNodeBoolValue(ext.Value) {
			bv, _ := strconv.ParseBool(ext.Value.Value)
			extensionMap[low.KeyReference[string]{
				Value:   ext.Key.Value,
				KeyNode: ext.Key,
			}] = low.ValueReference[any]{Value: bv, ValueNode: ext.Value}
		}
		if utils.IsNodeArray(ext.Value) {
			var v []interface{}
			err := ext.Value.Decode(&v)
			if err != nil {
				return nil, err
			}
			extensionMap[low.KeyReference[string]{
				Value:   ext.Key.Value,
				KeyNode: ext.Key,
			}] = low.ValueReference[any]{Value: v, ValueNode: ext.Value}
		}
	}
	return extensionMap, nil
}
