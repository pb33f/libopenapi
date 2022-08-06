package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"gopkg.in/yaml.v3"
)

type RequestBody struct {
	Description low.NodeReference[string]
	Content     low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*MediaType]]
	Required    low.NodeReference[bool]
	Extensions  map[low.KeyReference[string]]low.ValueReference[any]
}

func (rb *RequestBody) FindContent(cType string) *low.ValueReference[*MediaType] {
	return FindItemInMap[*MediaType](cType, rb.Content.Value)
}

func (rb *RequestBody) Build(root *yaml.Node) error {
	// extract extensions
	extensionMap, err := ExtractExtensions(root)
	if err != nil {
		return err
	}
	if extensionMap != nil {
		rb.Extensions = extensionMap
	}

	// handle content, if set.
	con, cL, cN, cErr := ExtractMapFlat[*MediaType](ContentLabel, root)
	if cErr != nil {
		return cErr
	}
	if con != nil {
		rb.Content = low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*MediaType]]{
			Value:     con,
			KeyNode:   cL,
			ValueNode: cN,
		}
	}
	return nil
}
