package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"gopkg.in/yaml.v3"
)

type ExternalDoc struct {
	Description low.NodeReference[string]
	URL         low.NodeReference[string]
	Extensions  map[low.KeyReference[string]]low.ValueReference[any]
}

func (ex *ExternalDoc) Build(root *yaml.Node, idx *index.SpecIndex) error {
	ex.Extensions = ExtractExtensions(root)
	return nil
}
