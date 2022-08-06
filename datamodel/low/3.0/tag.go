package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"gopkg.in/yaml.v3"
)

const (
	TagsLabel         = "tags"
	ExternalDocsLabel = "externalDocs"
)

type Tag struct {
	Name         low.NodeReference[string]
	Description  low.NodeReference[string]
	ExternalDocs low.NodeReference[*ExternalDoc]
	Extensions   map[low.KeyReference[string]]low.ValueReference[any]
}

func (t *Tag) Build(root *yaml.Node) error {
	// extract extensions
	extensionMap, err := ExtractExtensions(root)
	if err != nil {
		return err
	}
	t.Extensions = extensionMap

	// extract externalDocs
	extDocs, dErr := ExtractObject[ExternalDoc](ExternalDocsLabel, root)
	if dErr != nil {
		return dErr
	}
	t.ExternalDocs = extDocs

	return nil
}
