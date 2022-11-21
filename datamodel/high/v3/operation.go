// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	low "github.com/pb33f/libopenapi/datamodel/low/v3"
)

// Operation is a high-level representation of an OpenAPI 3+ Operation object, backed by a low-level one.
//
// An Operation is perhaps the most important object of the entire specification. Everything of value
// happens here. The entire being for existence of this library and the specification, is this Operation.
//   - https://spec.openapis.org/oas/v3.1.0#operation-object
type Operation struct {
	Tags         []string
	Summary      string
	Description  string
	ExternalDocs *base.ExternalDoc
	OperationId  string
	Parameters   []*Parameter
	RequestBody  *RequestBody
	Responses    *Responses
	Callbacks    map[string]*Callback
	Deprecated   *bool
	Security     []*base.SecurityRequirement
	Servers      []*Server
	Extensions   map[string]any
	low          *low.Operation
}

// NewOperation will create a new Operation instance from a low-level one.
func NewOperation(operation *low.Operation) *Operation {
	o := new(Operation)
	o.low = operation
	var tags []string
	if !operation.Tags.IsEmpty() {
		for i := range operation.Tags.Value {
			tags = append(tags, operation.Tags.Value[i].Value)
		}
	}
	o.Tags = tags
	o.Summary = operation.Summary.Value
	o.Deprecated = &operation.Deprecated.Value
	o.Description = operation.Description.Value
	if !operation.ExternalDocs.IsEmpty() {
		o.ExternalDocs = base.NewExternalDoc(operation.ExternalDocs.Value)
	}
	o.OperationId = operation.OperationId.Value
	if !operation.Parameters.IsEmpty() {
		params := make([]*Parameter, len(operation.Parameters.Value))
		for i := range operation.Parameters.Value {
			params[i] = NewParameter(operation.Parameters.Value[i].Value)
		}
		o.Parameters = params
	}
	if !operation.RequestBody.IsEmpty() {
		o.RequestBody = NewRequestBody(operation.RequestBody.Value)
	}
	if !operation.Responses.IsEmpty() {
		o.Responses = NewResponses(operation.Responses.Value)
	}
	if !operation.Security.IsEmpty() {
		var sec []*base.SecurityRequirement
		for s := range operation.Security.Value {
			sec = append(sec, base.NewSecurityRequirement(operation.Security.Value[s].Value))
		}
		o.Security = sec
	}
	var servers []*Server
	for i := range operation.Servers.Value {
		servers = append(servers, NewServer(operation.Servers.Value[i].Value))
	}
	o.Servers = servers
	o.Extensions = high.ExtractExtensions(operation.Extensions)
	return o
}

// GoLow will return the low-level Operation instance that was used to create the high-level one.
func (o *Operation) GoLow() *low.Operation {
	return o.low
}
