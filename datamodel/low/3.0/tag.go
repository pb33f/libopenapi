// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

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

func (t *Tag) FindExtension(ext string) *low.ValueReference[any] {
	return low.FindItemInMap[any](ext, t.Extensions)
}

func (t *Tag) Build(root *yaml.Node, idx *index.SpecIndex) error {
	t.Extensions = low.ExtractExtensions(root)

	// extract externalDocs
	extDocs, err := low.ExtractObject[*ExternalDoc](ExternalDocsLabel, root, idx)
	t.ExternalDocs = extDocs
	return err
}
