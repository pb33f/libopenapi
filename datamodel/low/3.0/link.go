package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

type Link struct {
	OperationRef low.NodeReference[string]
	OperationId  low.NodeReference[string]
	Parameters   low.NodeReference[map[low.KeyReference[string]]low.ValueReference[string]]
	RequestBody  low.NodeReference[string]
	Description  low.NodeReference[string]
	Server       low.NodeReference[*Server]
	Extensions   map[low.KeyReference[string]]low.ValueReference[any]
}

func (l *Link) Build(root *yaml.Node) error {
	extensionMap, err := ExtractExtensions(root)
	if err != nil {
		return err
	}
	l.Extensions = extensionMap

	// extract parameters
	_, pl, pv := utils.FindKeyNodeFull(ParametersLabel, root.Content)
	if pv != nil {
		params := make(map[low.KeyReference[string]]low.ValueReference[string])
		var currentParam *yaml.Node
		for i, param := range pv.Content {
			if i%2 == 0 {
				currentParam = param
				continue
			}
			params[low.KeyReference[string]{
				Value:   currentParam.Value,
				KeyNode: currentParam,
			}] = low.ValueReference[string]{
				Value:     param.Value,
				ValueNode: param,
			}
		}
		l.Parameters = low.NodeReference[map[low.KeyReference[string]]low.ValueReference[string]]{
			Value:     params,
			KeyNode:   pl,
			ValueNode: pv,
		}
	}
	return nil
}
