// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"crypto/sha256"
	"fmt"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"sort"
	"strings"
)

// Operation represents a low-level Swagger / OpenAPI 2 Operation object.
//
// It describes a single API operation on a path.
//   - https://swagger.io/specification/v2/#operationObject
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
	Security     low.NodeReference[[]low.ValueReference[*base.SecurityRequirement]]
	Extensions   map[low.KeyReference[string]]low.ValueReference[any]
}

// Build will extract external docs, extensions, parameters, responses and security requirements.
func (o *Operation) Build(_, root *yaml.Node, idx *index.SpecIndex) error {
	root = utils.NodeAlias(root)
	utils.CheckForMergeNodes(root)
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

	// extract security
	sec, sln, svn, sErr := low.ExtractArray[*base.SecurityRequirement](SecurityLabel, root, idx)
	if sErr != nil {
		return sErr
	}
	if sec != nil {
		o.Security = low.NodeReference[[]low.ValueReference[*base.SecurityRequirement]]{
			Value:     sec,
			KeyNode:   sln,
			ValueNode: svn,
		}
	}
	return nil
}

// Hash will return a consistent SHA256 Hash of the Operation object
func (o *Operation) Hash() [32]byte {
	var f []string
	if !o.Summary.IsEmpty() {
		f = append(f, o.Summary.Value)
	}
	if !o.Description.IsEmpty() {
		f = append(f, o.Description.Value)
	}
	if !o.OperationId.IsEmpty() {
		f = append(f, o.OperationId.Value)
	}
	if !o.Summary.IsEmpty() {
		f = append(f, o.Summary.Value)
	}
	if !o.ExternalDocs.IsEmpty() {
		f = append(f, low.GenerateHashString(o.ExternalDocs.Value))
	}
	if !o.Responses.IsEmpty() {
		f = append(f, low.GenerateHashString(o.Responses.Value))
	}
	if !o.Deprecated.IsEmpty() {
		f = append(f, fmt.Sprint(o.Deprecated.Value))
	}
	var keys []string
	keys = make([]string, len(o.Tags.Value))
	for k := range o.Tags.Value {
		keys[k] = o.Tags.Value[k].Value
	}
	sort.Strings(keys)
	f = append(f, keys...)

	keys = make([]string, len(o.Consumes.Value))
	for k := range o.Consumes.Value {
		keys[k] = o.Consumes.Value[k].Value
	}
	sort.Strings(keys)
	f = append(f, keys...)

	keys = make([]string, len(o.Produces.Value))
	for k := range o.Produces.Value {
		keys[k] = o.Produces.Value[k].Value
	}
	sort.Strings(keys)
	f = append(f, keys...)

	keys = make([]string, len(o.Schemes.Value))
	for k := range o.Schemes.Value {
		keys[k] = o.Schemes.Value[k].Value
	}
	sort.Strings(keys)
	f = append(f, keys...)

	keys = make([]string, len(o.Parameters.Value))
	for k := range o.Parameters.Value {
		keys[k] = low.GenerateHashString(o.Parameters.Value[k].Value)
	}
	sort.Strings(keys)
	f = append(f, keys...)

	keys = make([]string, len(o.Security.Value))
	for k := range o.Security.Value {
		keys[k] = low.GenerateHashString(o.Security.Value[k].Value)
	}
	sort.Strings(keys)
	f = append(f, keys...)
	keys = make([]string, len(o.Extensions))
	z := 0
	for k := range o.Extensions {
		keys[z] = fmt.Sprintf("%s-%x", k.Value, sha256.Sum256([]byte(fmt.Sprint(o.Extensions[k].Value))))
		z++
	}
	sort.Strings(keys)
	f = append(f, keys...)
	return sha256.Sum256([]byte(strings.Join(f, "|")))
}

// methods to satisfy swagger operations interface

func (o *Operation) GetTags() low.NodeReference[[]low.ValueReference[string]] {
	return o.Tags
}
func (o *Operation) GetSummary() low.NodeReference[string] {
	return o.Summary
}
func (o *Operation) GetDescription() low.NodeReference[string] {
	return o.Description
}
func (o *Operation) GetExternalDocs() low.NodeReference[any] {
	return low.NodeReference[any]{
		ValueNode: o.ExternalDocs.ValueNode,
		KeyNode:   o.ExternalDocs.KeyNode,
		Value:     o.ExternalDocs.Value,
	}
}
func (o *Operation) GetOperationId() low.NodeReference[string] {
	return o.OperationId
}
func (o *Operation) GetDeprecated() low.NodeReference[bool] {
	return o.Deprecated
}
func (o *Operation) GetExtensions() map[low.KeyReference[string]]low.ValueReference[any] {
	return o.Extensions
}
func (o *Operation) GetResponses() low.NodeReference[any] {
	return low.NodeReference[any]{
		ValueNode: o.Responses.ValueNode,
		KeyNode:   o.Responses.KeyNode,
		Value:     o.Responses.Value,
	}
}
func (o *Operation) GetParameters() low.NodeReference[any] {
	return low.NodeReference[any]{
		ValueNode: o.Parameters.ValueNode,
		KeyNode:   o.Parameters.KeyNode,
		Value:     o.Parameters.Value,
	}
}
func (o *Operation) GetSecurity() low.NodeReference[any] {
	return low.NodeReference[any]{
		ValueNode: o.Security.ValueNode,
		KeyNode:   o.Security.KeyNode,
		Value:     o.Security.Value,
	}
}
func (o *Operation) GetSchemes() low.NodeReference[[]low.ValueReference[string]] {
	return o.Schemes
}
func (o *Operation) GetProduces() low.NodeReference[[]low.ValueReference[string]] {
	return o.Produces
}
func (o *Operation) GetConsumes() low.NodeReference[[]low.ValueReference[string]] {
	return o.Consumes
}
