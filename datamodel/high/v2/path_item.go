// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"sync"

	"github.com/pb33f/libopenapi/datamodel/high"
	low "github.com/pb33f/libopenapi/datamodel/low/v2"
)

// PathItem represents a high-level Swagger / OpenAPI 2 PathItem object backed by a low-level one.
//
// Describes the operations available on a single path. A Path Item may be empty, due to ACL constraints.
// The path itself is still exposed to the tooling, but will not know which operations and parameters
// are available.
//   - https://swagger.io/specification/v2/#pathItemObject
type PathItem struct {
	Ref        string
	Get        *Operation
	Put        *Operation
	Post       *Operation
	Delete     *Operation
	Options    *Operation
	Head       *Operation
	Patch      *Operation
	Parameters []*Parameter
	Extensions map[string]any
	low        *low.PathItem
}

// NewPathItem will create a new high-level PathItem from a low-level one. All paths are built out asynchronously.
func NewPathItem(pathItem *low.PathItem) *PathItem {
	p := new(PathItem)
	p.low = pathItem
	p.Extensions = high.ExtractExtensions(pathItem.Extensions)
	if !pathItem.Parameters.IsEmpty() {
		var params []*Parameter
		for k := range pathItem.Parameters.Value {
			params = append(params, NewParameter(pathItem.Parameters.Value[k].Value))
		}
		p.Parameters = params
	}
	var buildOperation = func(method string, op *low.Operation) *Operation {
		return NewOperation(op)
	}

	var wg sync.WaitGroup
	if !pathItem.Get.IsEmpty() {
		wg.Add(1)
		go func() {
			p.Get = buildOperation(low.GetLabel, pathItem.Get.Value)
			wg.Done()
		}()
	}
	if !pathItem.Put.IsEmpty() {
		wg.Add(1)
		go func() {
			p.Put = buildOperation(low.PutLabel, pathItem.Put.Value)
			wg.Done()
		}()
	}
	if !pathItem.Post.IsEmpty() {
		wg.Add(1)
		go func() {
			p.Post = buildOperation(low.PostLabel, pathItem.Post.Value)
			wg.Done()
		}()
	}
	if !pathItem.Patch.IsEmpty() {
		wg.Add(1)
		go func() {
			p.Patch = buildOperation(low.PatchLabel, pathItem.Patch.Value)
			wg.Done()
		}()
	}
	if !pathItem.Delete.IsEmpty() {
		wg.Add(1)
		go func() {
			p.Delete = buildOperation(low.DeleteLabel, pathItem.Delete.Value)
			wg.Done()
		}()
	}
	if !pathItem.Head.IsEmpty() {
		wg.Add(1)
		go func() {
			p.Head = buildOperation(low.HeadLabel, pathItem.Head.Value)
			wg.Done()
		}()
	}
	if !pathItem.Options.IsEmpty() {
		wg.Add(1)
		go func() {
			p.Options = buildOperation(low.OptionsLabel, pathItem.Options.Value)
			wg.Done()
		}()
	}
	wg.Wait()
	return p
}

// GoLow returns the low-level PathItem used to create the high-level one.
func (p *PathItem) GoLow() *low.PathItem {
	return p.low
}

func (p *PathItem) GetOperations() map[string]*Operation {
	o := make(map[string]*Operation)
	if p.Get != nil {
		o[low.GetLabel] = p.Get
	}
	if p.Put != nil {
		o[low.PutLabel] = p.Put
	}
	if p.Post != nil {
		o[low.PostLabel] = p.Post
	}
	if p.Delete != nil {
		o[low.DeleteLabel] = p.Delete
	}
	if p.Options != nil {
		o[low.OptionsLabel] = p.Options
	}
	if p.Head != nil {
		o[low.HeadLabel] = p.Head
	}
	if p.Patch != nil {
		o[low.PatchLabel] = p.Patch
	}
	return o
}
