// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

// Operation is a low-level representation of an OpenAPI 3+ Operation object.
//
// An Operation is perhaps the most important object of the entire specification. Everything of value
// happens here. The entire being for existence of this library and the specification, is this Operation.
//   - https://spec.openapis.org/oas/v3.1.0#operation-object
type Operation struct {
	Tags         low.NodeReference[[]low.ValueReference[string]]
	Summary      low.NodeReference[string]
	Description  low.NodeReference[string]
	ExternalDocs low.NodeReference[*base.ExternalDoc]
	OperationId  low.NodeReference[string]
	Parameters   low.NodeReference[[]low.ValueReference[*Parameter]]
	RequestBody  low.NodeReference[*RequestBody]
	Responses    low.NodeReference[*Responses]
	Callbacks    low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[*Callback]]]
	Deprecated   low.NodeReference[bool]
	Security     low.NodeReference[[]low.ValueReference[*base.SecurityRequirement]]
	Servers      low.NodeReference[[]low.ValueReference[*Server]]
	Extensions   *orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]]
	KeyNode      *yaml.Node
	RootNode     *yaml.Node
	*low.Reference
}

// FindCallback will attempt to locate a Callback instance by the supplied name.
func (o *Operation) FindCallback(callback string) *low.ValueReference[*Callback] {
	return low.FindItemInOrderedMap(callback, o.Callbacks.GetValue())
}

// FindSecurityRequirement will attempt to locate a security requirement string from a supplied name.
func (o *Operation) FindSecurityRequirement(name string) []low.ValueReference[string] {
	for k := range o.Security.Value {
		requirements := o.Security.Value[k].Value.Requirements
		for pair := orderedmap.First(requirements.Value); pair != nil; pair = pair.Next() {
			if pair.Key().Value == name {
				return pair.Value().Value
			}
		}
	}
	return nil
}

// GetRootNode returns the root yaml node of the Operation object
func (o *Operation) GetRootNode() *yaml.Node {
	return o.RootNode
}

// GetKeyNode returns the key yaml node of the Operation object
func (o *Operation) GetKeyNode() *yaml.Node {
	return o.KeyNode
}

// Build will extract external docs, parameters, request body, responses, callbacks, security and servers.
func (o *Operation) Build(ctx context.Context, keyNode, root *yaml.Node, idx *index.SpecIndex) error {
	o.KeyNode = keyNode
	o.RootNode = root
	root = utils.NodeAlias(root)
	utils.CheckForMergeNodes(root)
	o.Reference = new(low.Reference)
	o.Extensions = low.ExtractExtensions(root)

	// extract externalDocs
	extDocs, dErr := low.ExtractObject[*base.ExternalDoc](ctx, base.ExternalDocsLabel, root, idx)
	if dErr != nil {
		return dErr
	}
	o.ExternalDocs = extDocs

	// extract parameters
	params, ln, vn, pErr := low.ExtractArray[*Parameter](ctx, ParametersLabel, root, idx)
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
	rBody, rErr := low.ExtractObject[*RequestBody](ctx, RequestBodyLabel, root, idx)
	if rErr != nil {
		return rErr
	}
	o.RequestBody = rBody

	// extract responses
	respBody, respErr := low.ExtractObject[*Responses](ctx, ResponsesLabel, root, idx)
	if respErr != nil {
		return respErr
	}
	o.Responses = respBody

	// extract callbacks
	callbacks, cbL, cbN, cbErr := low.ExtractMap[*Callback](ctx, CallbacksLabel, root, idx)
	if cbErr != nil {
		return cbErr
	}
	if callbacks != nil {
		o.Callbacks = low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[*Callback]]]{
			Value:     callbacks,
			KeyNode:   cbL,
			ValueNode: cbN,
		}
	}

	// extract security
	sec, sln, svn, sErr := low.ExtractArray[*base.SecurityRequirement](ctx, SecurityLabel, root, idx)
	if sErr != nil {
		return sErr
	}

	// if security is defined and requirements are provided.
	if sln != nil && len(svn.Content) > 0 && sec != nil {
		o.Security = low.NodeReference[[]low.ValueReference[*base.SecurityRequirement]]{
			Value:     sec,
			KeyNode:   sln,
			ValueNode: svn,
		}
	}

	// if security is set, but no requirements are defined.
	// https://github.com/pb33f/libopenapi/issues/111
	if sln != nil && len(svn.Content) == 0 && sec == nil {
		o.Security = low.NodeReference[[]low.ValueReference[*base.SecurityRequirement]]{
			Value:     []low.ValueReference[*base.SecurityRequirement]{}, // empty
			KeyNode:   sln,
			ValueNode: svn,
		}
	}

	// extract servers
	servers, sl, sn, serErr := low.ExtractArray[*Server](ctx, ServersLabel, root, idx)
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

	for pair := orderedmap.First(orderedmap.SortAlpha(o.Callbacks.Value)); pair != nil; pair = pair.Next() {
		f = append(f, low.GenerateHashString(pair.Value().Value))
	}
	f = append(f, low.HashExtensions(o.Extensions)...)

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

func (o *Operation) GetExtensions() *orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]] {
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

func (o *Operation) GetCallbacks() low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[*Callback]]] {
	return o.Callbacks
}
