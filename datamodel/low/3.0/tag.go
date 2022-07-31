package v3

import (
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

const (
	Tags         = "tags"
	ExternalDocs = "externalDocs"
)

type Tag struct {
	Name         low.NodeReference[string]
	Description  low.NodeReference[string]
	ExternalDocs low.NodeReference[*ExternalDoc]
	Extensions   map[low.NodeReference[string]]low.NodeReference[any]
}

func (t *Tag) Build(root *yaml.Node) error {
	_, ln, exDocs := utils.FindKeyNodeFull(ExternalDocs, root.Content)

	// extract extensions
	extensionMap, err := datamodel.ExtractExtensions(root)
	if err != nil {
		return err
	}
	t.Extensions = extensionMap

	// extract external docs
	var externalDoc ExternalDoc
	err = datamodel.BuildModel(exDocs, &externalDoc)
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
