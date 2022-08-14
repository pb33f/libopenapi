package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
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

func (t *Tag) Build(root *yaml.Node, idx *index.SpecIndex) error {
	t.Extensions = low.ExtractExtensions(root)

	// extract externalDocs
	extDocs, dErr := low.ExtractObject[*ExternalDoc](ExternalDocsLabel, root, idx)
	if dErr != nil {
		return dErr
	}
	t.ExternalDocs = extDocs

	return nil
}
