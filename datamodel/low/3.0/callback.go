package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

type Callback struct {
	Expression low.ValueReference[map[low.KeyReference[string]]low.ValueReference[*PathItem]]
	Extensions map[low.KeyReference[string]]low.ValueReference[any]
}

func (cb *Callback) FindExpression(exp string) *low.ValueReference[*PathItem] {
	return FindItemInMap[*PathItem](exp, cb.Expression.Value)
}

func (cb *Callback) Build(root *yaml.Node, idx *index.SpecIndex) error {
	cb.Extensions = ExtractExtensions(root)

	// handle callback
	var currentCB *yaml.Node
	callbacks := make(map[low.KeyReference[string]]low.ValueReference[*PathItem])

	if ok, _, _ := utils.IsNodeRefValue(root); ok {
		r := LocateRefNode(root, idx)
		if r != nil {
			root = r
		} else {
			return nil
		}
	}

	for i, callbackNode := range root.Content {
		if i%2 == 0 {
			currentCB = callbackNode
			continue
		}
		callback, eErr := ExtractObjectRaw[*PathItem](callbackNode, idx)
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
