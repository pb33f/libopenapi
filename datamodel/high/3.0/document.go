// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import low "github.com/pb33f/libopenapi/datamodel/low/3.0"

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
	return d
}

func (d *Document) GoLow() *low.Document {
	return d.low
}
