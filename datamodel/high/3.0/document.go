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

func (d *Document) GoLow() *low.Document {
	return d.low
}
