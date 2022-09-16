// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	low "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
)

type Document struct {
	Version           string
	Info              *base.Info
	Servers           []*Server
	Paths             *Paths
	Components        *Components
	Security          *SecurityRequirement
	Tags              []*base.Tag
	ExternalDocs      *base.ExternalDoc
	Extensions        map[string]any
	Index             *index.SpecIndex
	JsonSchemaDialect string
	Webhooks          map[string]*PathItem
	low               *low.Document
}

func NewDocument(document *low.Document) *Document {
	d := new(Document)
	d.low = document
	d.Index = document.Index
	if !document.Info.IsEmpty() {
		d.Info = base.NewInfo(document.Info.Value)
	}
	if !document.Version.IsEmpty() {
		d.Version = document.Version.Value
	}
	var servers []*Server
	for _, ser := range document.Servers.Value {
		servers = append(servers, NewServer(ser.Value))
	}
	d.Servers = servers
	var tags []*base.Tag
	for _, tag := range document.Tags.Value {
		tags = append(tags, base.NewTag(tag.Value))
	}
	d.Tags = tags
	if !document.ExternalDocs.IsEmpty() {
		d.ExternalDocs = base.NewExternalDoc(document.ExternalDocs.Value)
	}
	if len(document.Extensions) > 0 {
		d.Extensions = high.ExtractExtensions(document.Extensions)
	}
	if !document.Components.IsEmpty() {
		d.Components = NewComponents(document.Components.Value)
	}
	if !document.Paths.IsEmpty() {
		d.Paths = NewPaths(document.Paths.Value)
	}
	if !document.JsonSchemaDialect.IsEmpty() {
		d.JsonSchemaDialect = document.JsonSchemaDialect.Value
	}
	if !document.Webhooks.IsEmpty() {
		hooks := make(map[string]*PathItem)
		for h := range document.Webhooks.Value {
			hooks[h.Value] = NewPathItem(document.Webhooks.Value[h].Value)
		}
		d.Webhooks = hooks
	}
	return d
}

func (d *Document) GoLow() *low.Document {
	return d.low
}
