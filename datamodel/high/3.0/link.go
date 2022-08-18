// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import low "github.com/pb33f/libopenapi/datamodel/low/3.0"

type Link struct {
	OperationRef string
	OperationId  string
	Parameters   map[string]string
	RequestBody  string
	Description  string
	Server       *Server
	Extensions   map[string]any
	low          *low.Link
}

func (l *Link) GoLow() *low.Link {
	return l.low
}
