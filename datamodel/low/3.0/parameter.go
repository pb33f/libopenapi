package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

const (
	SchemaLabel = "schema"
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
	extensionMap, err := ExtractExtensions(root)
	if err != nil {
		return err
	}
	p.Extensions = extensionMap

	// handle schema
	_, schLabel, schNode := utils.FindKeyNodeFull(SchemaLabel, root.Content)
	if schNode != nil {
		// deal with schema flat props
		var schema Schema
		err = BuildModel(schNode, &schema)
		if err != nil {
			return err
		}

		// now comes the part where things may get hairy, schemas are recursive.
		// which means we could be here forever if our resolver has some unknown bug in it.
		// in order to prevent this from happening, we will add a counter that tracks the depth
		// and will hard stop once we reach 50 levels. That's too deep for any data structure IMHO.
		err = schema.Build(schNode, idx, 0)
		if err != nil {
			return err
		}

		p.Schema = low.NodeReference[*Schema]{Value: &schema, KeyNode: schLabel, ValueNode: schNode}
	}

	return nil
}
