// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"crypto/sha256"
	"fmt"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"gopkg.in/yaml.v3"
	"sort"
	"strings"
)

// Operation is a low-level representation of an OpenAPI 3+ Operation object.
//
// An Operation is perhaps the most important object of the entire specification. Everything of value
// happens here. The entire being for existence of this library and the specification, is this Operation.
//  - https://spec.openapis.org/oas/v3.1.0#operation-object
type Operation struct {
	Tags         low.NodeReference[[]low.ValueReference[string]]
	Summary      low.NodeReference[string]
	Description  low.NodeReference[string]
	ExternalDocs low.NodeReference[*base.ExternalDoc]
	OperationId  low.NodeReference[string]
	Parameters   low.NodeReference[[]low.ValueReference[*Parameter]]
	RequestBody  low.NodeReference[*RequestBody]
	Responses    low.NodeReference[*Responses]
	Callbacks    low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Callback]]
	Deprecated   low.NodeReference[bool]
	Security     low.NodeReference[[]low.ValueReference[*base.SecurityRequirement]]
	Servers      low.NodeReference[[]low.ValueReference[*Server]]
	Extensions   map[low.KeyReference[string]]low.ValueReference[any]
	*low.Reference
}

// FindCallback will attempt to locate a Callback instance by the supplied name.
func (o *Operation) FindCallback(callback string) *low.ValueReference[*Callback] {
	return low.FindItemInMap[*Callback](callback, o.Callbacks.Value)
}

// FindSecurityRequirement will attempt to locate a security requirement string from a supplied name.
func (o *Operation) FindSecurityRequirement(name string) []low.ValueReference[string] {
	for k := range o.Security.Value {
		for i := range o.Security.Value[k].Value.Requirements.Value {
			if i.Value == name {
				return o.Security.Value[k].Value.Requirements.Value[i].Value
			}
		}
	}
	return nil
}

// Build will extract external docs, parameters, request body, responses, callbacks, security and servers.
func (o *Operation) Build(root *yaml.Node, idx *index.SpecIndex) error {
	o.Reference = new(low.Reference)
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

	// extract request body
	rBody, rErr := low.ExtractObject[*RequestBody](RequestBodyLabel, root, idx)
	if rErr != nil {
		return rErr
	}
	o.RequestBody = rBody

	// extract responses
	respBody, respErr := low.ExtractObject[*Responses](ResponsesLabel, root, idx)
	if respErr != nil {
		return respErr
	}
	o.Responses = respBody

	// extract callbacks
	callbacks, cbL, cbN, cbErr := low.ExtractMap[*Callback](CallbacksLabel, root, idx)
	if cbErr != nil {
		return cbErr
	}
	if callbacks != nil {
		o.Callbacks = low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Callback]]{
			Value:     callbacks,
			KeyNode:   cbL,
			ValueNode: cbN,
		}
	}

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

	// extract servers
	servers, sl, sn, serErr := low.ExtractArray[*Server](ServersLabel, root, idx)
	if serErr != nil {
		return serErr
	}
	if servers != nil {
		o.Servers = low.NodeReference[[]low.ValueReference[*Server]]{
			Value:     servers,
			KeyNode:   sl,
			ValueNode: sn,
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
	if !o.RequestBody.IsEmpty() {
		f = append(f, low.GenerateHashString(o.RequestBody.Value))
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
	if !o.Security.IsEmpty() {
		for k := range o.Security.Value {
			f = append(f, low.GenerateHashString(o.Security.Value[k].Value))
		}
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

	keys = make([]string, len(o.Servers.Value))
	for k := range o.Servers.Value {
		keys[k] = low.GenerateHashString(o.Servers.Value[k].Value)
	}
	sort.Strings(keys)
	f = append(f, keys...)

	keys = make([]string, len(o.Parameters.Value))
	for k := range o.Parameters.Value {
		keys[k] = low.GenerateHashString(o.Parameters.Value[k].Value)
	}
	sort.Strings(keys)
	f = append(f, keys...)

	keys = make([]string, len(o.Callbacks.Value))
	z := 0
	for k := range o.Callbacks.Value {
		keys[z] = low.GenerateHashString(o.Callbacks.Value[k].Value)
		z++
	}
	sort.Strings(keys)
	f = append(f, keys...)

	keys = make([]string, len(o.Extensions))
	z = 0
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
func (o *Operation) GetServers() low.NodeReference[any] {
	return low.NodeReference[any]{
		ValueNode: o.Servers.ValueNode,
		KeyNode:   o.Servers.KeyNode,
		Value:     o.Servers.Value,
	}
}
func (o *Operation) GetCallbacks() low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Callback]] {
	return o.Callbacks
}
