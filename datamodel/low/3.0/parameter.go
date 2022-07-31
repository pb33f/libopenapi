package v3

import (
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"gopkg.in/yaml.v3"
)

type Parameter struct {
	Name            low.NodeReference[string]
	In              low.NodeReference[string]
	Description     low.NodeReference[string]
	Required        low.NodeReference[bool]
	Deprecated      low.NodeReference[bool]
	AllowEmptyValue low.NodeReference[bool]
	Style           low.NodeReference[string]
	Explode         low.NodeReference[bool]
	AllowReserved   low.NodeReference[bool]
	Schema          low.NodeReference[*Schema]
	Example         low.NodeReference[any]
	Examples        map[low.NodeReference[string]]low.NodeReference[*Example]
	Extensions      map[low.NodeReference[string]]low.NodeReference[any]
}

func (p *Parameter) Build(root *yaml.Node, idx *index.SpecIndex) error {

	// extract extensions
	extensionMap, err := datamodel.ExtractExtensions(root)
	if err != nil {
		return err
	}
	p.Extensions = extensionMap

	// deal with schema

	return nil
}
