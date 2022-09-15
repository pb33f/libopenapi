// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
)

type Document struct {
	Version           low.ValueReference[string]
	Info              low.NodeReference[*base.Info]
	JsonSchemaDialect low.NodeReference[string]                                                     // 3.1
	Webhooks          low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*PathItem]] // 3.1
	Servers           low.NodeReference[[]low.ValueReference[*Server]]
	Paths             low.NodeReference[*Paths]
	Components        low.NodeReference[*Components]
	Security          low.NodeReference[*SecurityRequirement]
	Tags              low.NodeReference[[]low.ValueReference[*base.Tag]]
	ExternalDocs      low.NodeReference[*base.ExternalDoc]
	Extensions        map[low.KeyReference[string]]low.ValueReference[any]
	Index             *index.SpecIndex
}

//
//func (d *Document) AddTag() *base.Tag {
//	t := base.NewTag()
//	//d.Tags.KeyNode
//	t.Name.Value = "nice new tag"
//
//	dat, _ := yaml.Marshal(t)
//	var inject yaml.Node
//	_ = yaml.Unmarshal(dat, &inject)
//
//	d.Tags.ValueNode.Content = append(d.Tags.ValueNode.Content, inject.Content[0])
//
//	return t
//}
