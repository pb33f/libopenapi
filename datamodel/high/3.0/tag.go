// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	low "github.com/pb33f/libopenapi/datamodel/low/3.0"
)

type Tag struct {
	Name         string
	Description  string
	ExternalDocs *ExternalDoc
	Extensions   map[string]any
	low          *low.Tag
}

func NewTag(tag *low.Tag) *Tag {
	t := new(Tag)
	t.low = tag
	t.Name = tag.Name.Value
	t.Description = tag.Description.Value
	t.ExternalDocs = NewExternalDoc(tag.ExternalDocs.Value)
	t.Extensions = high.ExtractExtensions(tag.Extensions)
	return t
}

func (t *Tag) GoLow() *low.Tag {
	return t.low
}
