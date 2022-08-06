package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"gopkg.in/yaml.v3"
)

const (
	EncodingLabel = "encoding"
)

type Encoding struct {
	ContentType   low.NodeReference[string]
	Headers       low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Header]]
	Style         low.NodeReference[string]
	Explode       low.NodeReference[bool]
	AllowReserved low.NodeReference[bool]
}

func (en *Encoding) FindHeader(hType string) *low.ValueReference[*Header] {
	return FindItemInMap[*Header](hType, en.Headers.Value)
}

func (en *Encoding) Build(root *yaml.Node) error {

	headers, hL, hN, err := ExtractMapFlat[*Header](HeadersLabel, root)
	if err != nil {
		return err
	}
	if headers != nil {
		en.Headers = low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Header]]{
			Value:     headers,
			KeyNode:   hL,
			ValueNode: hN,
		}
	}

	return nil
}
