package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/utils"
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

	_, ln, exDocs := utils.FindKeyNodeFull(ExternalDocsLabel, root.Content)
	// extract external docs
	var externalDoc ExternalDoc
	err = BuildModel(exDocs, &externalDoc)
	if err != nil {
		return err
	}
	t.ExternalDocs = low.NodeReference[*ExternalDoc]{
		Value:     &externalDoc,
		KeyNode:   ln,
		ValueNode: exDocs,
	}
	return nil
}
