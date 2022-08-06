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
	Headers       map[low.KeyReference[string]]map[low.KeyReference[string]]low.ValueReference[*Header]
	Style         low.NodeReference[string]
	Explode       low.NodeReference[bool]
	AllowReserved low.NodeReference[bool]
}

func (en *Encoding) FindHeader(hType string) *low.ValueReference[*Header] {
	for _, c := range en.Headers {
		for n, o := range c {
			if n.Value == hType {
				return &o
			}
		}
	}
	return nil
}

func (en *Encoding) Build(root *yaml.Node) error {

	headers, err := ExtractMap[*Header](HeadersLabel, root)
	if err != nil {
		return err
	}
	if headers != nil {
		en.Headers = headers
	}

	return nil
}
