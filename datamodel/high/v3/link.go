// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	low "github.com/pb33f/libopenapi/datamodel/low/v3"
)

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

func NewLink(link *low.Link) *Link {
	l := new(Link)
	l.low = link
	l.OperationRef = link.OperationRef.Value
	l.OperationId = link.OperationId.Value
	params := make(map[string]string)
	for k, v := range link.Parameters.Value {
		params[k.Value] = v.Value
	}
	l.Parameters = params
	l.RequestBody = link.RequestBody.Value
	l.Description = link.Description.Value
	if link.Server.Value != nil {
		l.Server = NewServer(link.Server.Value)
	}
	l.Extensions = high.ExtractExtensions(link.Extensions)
	return l
}

func (l *Link) GoLow() *low.Link {
	return l.low
}
