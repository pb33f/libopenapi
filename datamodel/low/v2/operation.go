// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"gopkg.in/yaml.v3"
)

// Operation represents a low-level Swagger / OpenAPI 2 Operation object.
//
// It describes a single API operation on a path.
//  - https://swagger.io/specification/v2/#operationObject
type Operation struct {
	Tags         low.NodeReference[[]low.ValueReference[string]]
	Summary      low.NodeReference[string]
	Description  low.NodeReference[string]
	ExternalDocs low.NodeReference[*base.ExternalDoc]
	OperationId  low.NodeReference[string]
	Consumes     low.NodeReference[[]low.ValueReference[string]]
	Produces     low.NodeReference[[]low.ValueReference[string]]
	Parameters   low.NodeReference[[]low.ValueReference[*Parameter]]
	Responses    low.NodeReference[*Responses]
	Schemes      low.NodeReference[[]low.ValueReference[string]]
	Deprecated   low.NodeReference[bool]
	Security     low.NodeReference[[]low.ValueReference[*SecurityRequirement]]
	Extensions   map[low.KeyReference[string]]low.ValueReference[any]
}

// Build will extract external docs, extensions, parameters, responses and security requirements.
func (o *Operation) Build(root *yaml.Node, idx *index.SpecIndex) error {
	o.Extensions = low.ExtractExtensions(root)

	// extract externalDocs
	extDocs, dErr := low.ExtractObject[*base.ExternalDoc](base.ExternalDocsLabel, root, idx)
	if dErr != nil {
		return dErr
	}
	o.ExternalDocs = extDocs

	// extract parameters
	params, ln, vn, pErr := low.ExtractArray[*Parameter](ParametersLabel, root, idx)
	if pErr != nil {
		return pErr
	}
	if params != nil {
		o.Parameters = low.NodeReference[[]low.ValueReference[*Parameter]]{
			Value:     params,
			KeyNode:   ln,
			ValueNode: vn,
		}
	}

	// extract responses
	respBody, respErr := low.ExtractObject[*Responses](ResponsesLabel, root, idx)
	if respErr != nil {
		return respErr
	}
	o.Responses = respBody

	// extract parameters
	sec, sln, svn, sErr := low.ExtractArray[*SecurityRequirement](SecurityLabel, root, idx)
	if sErr != nil {
		return sErr
	}
	if sec != nil {
		o.Security = low.NodeReference[[]low.ValueReference[*SecurityRequirement]]{
			Value:     sec,
			KeyNode:   sln,
			ValueNode: svn,
		}
	}
	return nil
}
