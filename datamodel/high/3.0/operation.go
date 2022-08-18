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

func (o *Operation) GoLow() *low.Operation {
	return o.low
}
