// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	low "github.com/pb33f/libopenapi/datamodel/low/3.0"
)

type Document struct {
	Version      string
	Info         *Info
	Servers      []*Server
	Paths        *Paths
	Components   *Components
	Security     *SecurityRequirement
	Tags         []*Tag
	ExternalDocs *ExternalDoc
	Extensions   map[string]any
	low          *low.Document
}

func NewDocument(document *low.Document) *Document {
	d := new(Document)
	d.low = document
	d.Info = NewInfo(document.Info.Value)
	d.Version = document.Version.Value
	var servers []*Server
	for _, ser := range document.Servers.Value {
		servers = append(servers, NewServer(ser.Value))
	}
	d.Servers = servers
	var tags []*Tag
	for _, tag := range document.Tags.Value {
		tags = append(tags, NewTag(tag.Value))
	}
	d.Tags = tags
	if !document.ExternalDocs.IsEmpty() {
		d.ExternalDocs = NewExternalDoc(document.ExternalDocs.Value)
	}
	d.Paths = NewPaths(document.Paths.Value)
	d.Extensions = high.ExtractExtensions(document.Extensions)
	d.Components = NewComponents(document.Components.Value)
	return d
}

func (d *Document) GoLow() *low.Document {
	return d.low
}
