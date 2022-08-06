package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"gopkg.in/yaml.v3"
)

type ExternalDoc struct {
	Description low.NodeReference[string]
	URL         low.NodeReference[string]
	Extensions  map[low.KeyReference[string]]low.ValueReference[any]
}

func (ex *ExternalDoc) Build(root *yaml.Node) error {
	// extract extensions
	extensionMap, err := ExtractExtensions(root)
	if err != nil {
		return err
	}
	ex.Extensions = extensionMap
	return nil
}
