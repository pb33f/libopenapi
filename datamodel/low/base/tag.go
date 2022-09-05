// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"gopkg.in/yaml.v3"
)

const (
	TagsLabel         = "tags"
	ExternalDocsLabel = "externalDocs"
	NameLabel         = "name"
	DescriptionLabel  = "description"
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

//func (t *Tag) MarshalYAML() (interface{}, error) {
//	m := make(map[string]interface{})
//	for i := range t.Extensions {
//		m[i.Value] = t.Extensions[i].Value
//	}
//	if t.Name.Value != "" {
//		m[NameLabel] = t.Name.Value
//	}
//	if t.Description.Value != "" {
//		m[DescriptionLabel] = t.Description.Value
//	}
//	if t.ExternalDocs.Value != nil {
//		m[ExternalDocsLabel] = t.ExternalDocs.Value
//	}
//	return m, nil
//}
//
//func NewTag() *Tag {
//	return new(Tag)
//}
