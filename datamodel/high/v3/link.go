// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	low "github.com/pb33f/libopenapi/datamodel/low/v3"
)

// Link represents a high-level OpenAPI 3+ Link object that is backed by a low-level one.
//
// The Link object represents a possible design-time link for a response. The presence of a link does not guarantee the
// caller’s ability to successfully invoke it, rather it provides a known relationship and traversal mechanism between
// responses and other operations.
//
// Unlike dynamic links (i.e. links provided in the response payload), the OAS linking mechanism does not require
// link information in the runtime response.
//
// For computing links, and providing instructions to execute them, a runtime expression is used for accessing values
// in an operation and using them as parameters while invoking the linked operation.
//  - https://spec.openapis.org/oas/v3.1.0#link-object
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

// NewLink will create a new high-level Link instance from a low-level one.
func NewLink(link *low.Link) *Link {
	l := new(Link)
	l.low = link
	l.OperationRef = link.OperationRef.Value
	l.OperationId = link.OperationId.Value
	params := make(map[string]string)
	for k, v := range link.Parameters {
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

// GoLow will return the low-level Link instance used to create the high-level one.
func (l *Link) GoLow() *low.Link {
	return l.low
}
