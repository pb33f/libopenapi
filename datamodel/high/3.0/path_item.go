// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import low "github.com/pb33f/libopenapi/datamodel/low/3.0"

type PathItem struct {
	Description string
	Summary     string
	Get         *Operation
	Put         *Operation
	Post        *Operation
	Delete      *Operation
	Options     *Operation
	Head        *Operation
	Patch       *Operation
	Trace       *Operation
	Servers     []*Server
	Parameters  []*Parameter
	Extensions  map[string]any
	low         *low.PathItem
}

func NewPathItem(lowPathItem *low.PathItem) *PathItem {
	pi := new(PathItem)
	pi.Description = lowPathItem.Description.Value
	pi.Summary = lowPathItem.Summary.Value
	return pi
}

func (p *PathItem) GoLow() *low.PathItem {
	return p.low
}
