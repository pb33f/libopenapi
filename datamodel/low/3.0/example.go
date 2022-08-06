package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"gopkg.in/yaml.v3"
)

const (
	ExamplesLabel = "examples"
	ExampleLabel  = "example"
)

type Example struct {
	Summary       low.NodeReference[string]
	Description   low.NodeReference[string]
	Value         low.NodeReference[any]
	ExternalValue low.NodeReference[string]
	Extensions    map[low.KeyReference[string]]low.ValueReference[any]
}

func (ex Example) Build(root *yaml.Node) error {
	// extract extensions
	extensionMap, err := ExtractExtensions(root)
	if err != nil {
		return err
	}
	ex.Extensions = extensionMap
	return nil
}
