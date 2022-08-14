package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
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

func (h *Header) Build(root *yaml.Node, idx *index.SpecIndex) error {
	h.Extensions = low.ExtractExtensions(root)

	// handle examples if set.
	exps, eErr := low.ExtractMap[*Example](ExamplesLabel, root, idx)
	if eErr != nil {
		return eErr
	}
	if exps != nil {
		h.Examples = exps
	}

	// handle schema
	sch, sErr := ExtractSchema(root, idx)
	if sErr != nil {
		return nil
	}
	if sch != nil {
		h.Schema = *sch
	}

	// handle content, if set.
	con, cErr := low.ExtractMap[*MediaType](ContentLabel, root, idx)
	if cErr != nil {
		return cErr
	}
	h.Content = con

	return nil
}
