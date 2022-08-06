package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"gopkg.in/yaml.v3"
)

const (
	HeadersLabel = "headers"
)

type Header struct {
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

func (h *Header) Build(root *yaml.Node) error {
	// extract extensions
	extensionMap, err := ExtractExtensions(root)
	if err != nil {
		return err
	}
	h.Extensions = extensionMap

	// handle examples if set.
	exps, eErr := ExtractMap[*Example](ExamplesLabel, root)
	if eErr != nil {
		return eErr
	}
	if exps != nil {
		h.Examples = exps
	}

	// handle schema
	sch, sErr := ExtractSchema(root)
	if sErr != nil {
		return nil
	}
	if sch != nil {
		h.Schema = *sch
	}

	// handle content, if set.
	con, cErr := ExtractMap[*MediaType](ContentLabel, root)
	if cErr != nil {
		return cErr
	}
	h.Content = con

	return nil
}
