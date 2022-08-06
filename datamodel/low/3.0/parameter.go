package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

const (
	SchemaLabel  = "schema"
	ContentLabel = "content"
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
	Examples        map[low.KeyReference[string]]map[low.KeyReference[string]]low.ValueReference[*Example]
	Content         map[low.KeyReference[string]]map[low.KeyReference[string]]low.ValueReference[*MediaType]
	Extensions      map[low.KeyReference[string]]low.ValueReference[any]
}

func (p *Parameter) FindContent(cType string) *low.ValueReference[*MediaType] {
	for _, c := range p.Content {
		for n, o := range c {
			if n.Value == cType {
				return &o
			}
		}
	}
	return nil
}

func (p *Parameter) Build(root *yaml.Node) error {

	// extract extensions
	extensionMap, err := ExtractExtensions(root)
	if err != nil {
		return err
	}
	p.Extensions = extensionMap

	// handle example if set.
	_, expLabel, expNode := utils.FindKeyNodeFull(ExampleLabel, root.Content)
	if expNode != nil {
		p.Example = low.NodeReference[any]{Value: expNode.Value, KeyNode: expLabel, ValueNode: expNode}
	}

	// handle schema
	sch, sErr := ExtractSchema(root)
	if sErr != nil {
		return sErr
	}
	p.Schema = *sch

	// handle examples if set.
	exps, eErr := ExtractMap[*Example](ExamplesLabel, root)
	if eErr != nil {
		return eErr
	}
	if exps != nil {
		p.Examples = exps
	}

	// handle content, if set.
	con, cErr := ExtractMap[*MediaType](ContentLabel, root)
	if cErr != nil {
		return cErr
	}
	p.Content = con
	return nil
}
