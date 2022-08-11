package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

const (
	ExamplesLabel = "examples"
	ExampleLabel  = "example"
	ValueLabel    = "value"
)

type Example struct {
	Summary       low.NodeReference[string]
	Description   low.NodeReference[string]
	Value         low.NodeReference[any]
	ExternalValue low.NodeReference[string]
	Extensions    map[low.KeyReference[string]]low.ValueReference[any]
}

func (ex *Example) Build(root *yaml.Node, idx *index.SpecIndex) error {
	ex.Extensions = ExtractExtensions(root)

	// extract value
	_, ln, vn := utils.FindKeyNodeFull(ValueLabel, root.Content)
	if vn != nil {
		var n map[string]interface{}
		err := vn.Decode(&n)
		if err != nil {
			return err
		}
		ex.Value = low.NodeReference[any]{
			Value:     n,
			KeyNode:   ln,
			ValueNode: vn,
		}
	}

	return nil
}
