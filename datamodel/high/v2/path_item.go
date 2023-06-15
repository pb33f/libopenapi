// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
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
	var buildOperation = func(method string, op *low.Operation, resChan chan<- asyncResult[*Operation]) {
		resChan <- asyncResult[*Operation]{
			key:    method,
			result: NewOperation(op),
		}
	}
	totalOperations := 0
	resChan := make(chan asyncResult[*Operation])
	if !pathItem.Get.IsEmpty() {
		totalOperations++
		go buildOperation(low.GetLabel, pathItem.Get.Value, resChan)
	}
	if !pathItem.Put.IsEmpty() {
		totalOperations++
		go buildOperation(low.PutLabel, pathItem.Put.Value, resChan)
	}
	if !pathItem.Post.IsEmpty() {
		totalOperations++
		go buildOperation(low.PostLabel, pathItem.Post.Value, resChan)
	}
	if !pathItem.Patch.IsEmpty() {
		totalOperations++
		go buildOperation(low.PatchLabel, pathItem.Patch.Value, resChan)
	}
	if !pathItem.Delete.IsEmpty() {
		totalOperations++
		go buildOperation(low.DeleteLabel, pathItem.Delete.Value, resChan)
	}
	if !pathItem.Head.IsEmpty() {
		totalOperations++
		go buildOperation(low.HeadLabel, pathItem.Head.Value, resChan)
	}
	if !pathItem.Options.IsEmpty() {
		totalOperations++
		go buildOperation(low.OptionsLabel, pathItem.Options.Value, resChan)
	}
	completedOperations := 0
	for completedOperations < totalOperations {
		select {
		case r := <-resChan:
			switch r.key {
			case low.GetLabel:
				completedOperations++
				p.Get = r.result
			case low.PutLabel:
				completedOperations++
				p.Put = r.result
			case low.PostLabel:
				completedOperations++
				p.Post = r.result
			case low.PatchLabel:
				completedOperations++
				p.Patch = r.result
			case low.DeleteLabel:
				completedOperations++
				p.Delete = r.result
			case low.HeadLabel:
				completedOperations++
				p.Head = r.result
			case low.OptionsLabel:
				completedOperations++
				p.Options = r.result
			}
		}
	}
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
