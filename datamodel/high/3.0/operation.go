// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import low "github.com/pb33f/libopenapi/datamodel/low/3.0"

type Operation struct {
	Tags         []string
	Summary      string
	Description  string
	ExternalDocs *ExternalDoc
	OperationId  string
	Parameters   []*Parameter
	RequestBody  *RequestBody
	Responses    *Responses
	Callbacks    map[string]*Callback
	Deprecated   bool
	Security     *SecurityRequirement
	Servers      []*Server
	Extensions   map[string]any
	low          *low.Operation
}

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
	o.Description = operation.Description.Value
	if !operation.ExternalDocs.IsEmpty() {
		o.ExternalDocs = NewExternalDoc(operation.ExternalDocs.Value)
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
		o.Security = NewSecurityRequirement(operation.Security.Value)
	}

	return o
}

func (o *Operation) GoLow() *low.Operation {
	return o.low
}
