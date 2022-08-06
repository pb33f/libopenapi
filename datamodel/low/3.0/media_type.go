package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

type MediaType struct {
	Schema     low.NodeReference[*Schema]
	Example    low.NodeReference[any]
	Examples   map[low.KeyReference[string]]map[low.KeyReference[string]]low.ValueReference[*Example]
	Encoding   map[low.KeyReference[string]]map[low.KeyReference[string]]low.ValueReference[*Encoding]
	Extensions map[low.KeyReference[string]]low.ValueReference[any]
}

func (mt *MediaType) Build(root *yaml.Node) error {

	// extract extensions
	extensionMap, err := ExtractExtensions(root)
	if err != nil {
		return err
	}
	mt.Extensions = extensionMap

	// handle example if set.
	_, expLabel, expNode := utils.FindKeyNodeFull(ExampleLabel, root.Content)
	if expNode != nil {
		mt.Example = low.NodeReference[any]{Value: expNode.Value, KeyNode: expLabel, ValueNode: expNode}
	}

	// handle schema
	sch, sErr := ExtractSchema(root)
	if sErr != nil {
		return nil
	}
	if sch != nil {
		mt.Schema = *sch
	}

	// handle examples if set.
	exps, eErr := ExtractMap[*Example](ExamplesLabel, root)
	if eErr != nil {
		return eErr
	}
	if exps != nil {
		mt.Examples = exps
	}

	// handle encoding
	encs, encErr := ExtractMap[*Encoding](EncodingLabel, root)
	if encErr != nil {
		return err
	}
	if encs != nil {
		mt.Encoding = encs
	}
	return nil
}
