package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"gopkg.in/yaml.v3"
)

type Callback struct {
	Expression low.ValueReference[map[low.KeyReference[string]]low.ValueReference[*PathItem]]
	Extensions map[low.KeyReference[string]]low.ValueReference[any]
}

func (cb *Callback) Build(root *yaml.Node) error {
	extensionMap, err := ExtractExtensions(root)
	if err != nil {
		return err
	}
	cb.Extensions = extensionMap

	// handle callback
	var currentCB *yaml.Node
	callbacks := make(map[low.KeyReference[string]]low.ValueReference[*PathItem])
	for i, callbackNode := range root.Content {
		if i%2 == 0 {
			currentCB = callbackNode
			continue
		}
		callback, eErr := ExtractObjectRaw[*PathItem](callbackNode)
		if eErr != nil {
			return eErr
		}
		callbacks[low.KeyReference[string]{
			Value:   currentCB.Value,
			KeyNode: currentCB,
		}] = low.ValueReference[*PathItem]{
			Value:     callback,
			ValueNode: callbackNode,
		}
	}
	if len(callbacks) > 0 {
		cb.Expression = low.ValueReference[map[low.KeyReference[string]]low.ValueReference[*PathItem]]{
			Value:     callbacks,
			ValueNode: root,
		}
	}
	return nil
}
