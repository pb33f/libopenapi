// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	low "github.com/pb33f/libopenapi/datamodel/low/2.0"
)

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

func (p *PathItem) GoLow() *low.PathItem {
	return p.low
}
