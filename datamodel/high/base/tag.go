// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
    "github.com/pb33f/libopenapi/datamodel/high"
    low "github.com/pb33f/libopenapi/datamodel/low/base"
)

// Tag represents a high-level Tag instance that is backed by a low-level one.
//
// Adds metadata to a single tag that is used by the Operation Object. It is not mandatory to have a Tag Object per
// tag defined in the Operation Object instances.
//  - v2: https://swagger.io/specification/v2/#tagObject
//  - v3: https://swagger.io/specification/#tag-object
type Tag struct {
    Name         string
    Description  string
    ExternalDocs *ExternalDoc
    Extensions   map[string]any
    low          *low.Tag
}

// NewTag creates a new high-level Tag instance that is backed by a low-level one.
func NewTag(tag *low.Tag) *Tag {
    t := new(Tag)
    t.low = tag
    if !tag.Name.IsEmpty() {
        t.Name = tag.Name.Value
    }
    if !tag.Description.IsEmpty() {
        t.Description = tag.Description.Value
    }
    if !tag.ExternalDocs.IsEmpty() {
        t.ExternalDocs = NewExternalDoc(tag.ExternalDocs.Value)
    }
    t.Extensions = high.ExtractExtensions(tag.Extensions)
    return t
}

// GoLow returns the low-level Tag instance used to create the high-level one.
func (t *Tag) GoLow() *low.Tag {
    return t.low
}

// Experimental mutation API.
//func (t *Tag) SetName(value string) {
//	t.GoLow().Name.ValueNode.Value = value
//}
//func (t *Tag) SetDescription(value string) {
//	t.GoLow().Description.ValueNode.Value = value
//}

//func (t *Tag) MarshalYAML() (interface{}, error) {
//	m := make(map[string]interface{})
//	for i := range t.Extensions {
//		m[i] = t.Extensions[i]
//	}
//	if t.Name != "" {
//		m[NameLabel] = t.Name
//	}
//	if t.Description != "" {
//		m[DescriptionLabel] = t.Description
//	}
//	if t.ExternalDocs != nil {
//		m[ExternalDocsLabel] = t.ExternalDocs
//	}
//	return m, nil
//}
