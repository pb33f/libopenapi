// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	low "github.com/pb33f/libopenapi/datamodel/low/v3"
	"gopkg.in/yaml.v3"
)

const (
	get = iota
	put
	post
	del
	options
	head
	patch
	trace
)

// PathItem represents a high-level OpenAPI 3+ PathItem object backed by a low-level one.
//
// Describes the operations available on a single path. A Path Item MAY be empty, due to ACL constraints.
// The path itself is still exposed to the documentation viewer but they will not know which operations and parameters
// are available.
//   - https://spec.openapis.org/oas/v3.1.0#path-item-object
type PathItem struct {
	Description string         `json:"description,omitempty" yaml:"description,omitempty"`
	Summary     string         `json:"summary,omitempty" yaml:"summary,omitempty"`
	Get         *Operation     `json:"get,omitempty" yaml:"get,omitempty"`
	Put         *Operation     `json:"put,omitempty" yaml:"put,omitempty"`
	Post        *Operation     `json:"post,omitempty" yaml:"post,omitempty"`
	Delete      *Operation     `json:"delete,omitempty" yaml:"delete,omitempty"`
	Options     *Operation     `json:"options,omitempty" yaml:"options,omitempty"`
	Head        *Operation     `json:"head,omitempty" yaml:"head,omitempty"`
	Patch       *Operation     `json:"patch,omitempty" yaml:"patch,omitempty"`
	Trace       *Operation     `json:"trace,omitempty" yaml:"trace,omitempty"`
	Servers     []*Server      `json:"servers,omitempty" yaml:"servers,omitempty"`
	Parameters  []*Parameter   `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Extensions  map[string]any `json:"-" yaml:"-"`
	low         *low.PathItem
}

// NewPathItem creates a new high-level PathItem instance from a low-level one.
func NewPathItem(pathItem *low.PathItem) *PathItem {
	pi := new(PathItem)
	pi.low = pathItem
	pi.Description = pathItem.Description.Value
	pi.Summary = pathItem.Summary.Value
	pi.Extensions = high.ExtractExtensions(pathItem.Extensions)
	var servers []*Server
	for _, ser := range pathItem.Servers.Value {
		servers = append(servers, NewServer(ser.Value))
	}
	pi.Servers = servers

	// build operation async
	type opResult struct {
		method int
		op     *Operation
	}
	opChan := make(chan opResult)
	var buildOperation = func(method int, op *low.Operation, c chan opResult) {
		if op == nil {
			c <- opResult{method: method, op: nil}
			return
		}
		c <- opResult{method: method, op: NewOperation(op)}
	}
	// build out operations async.
	go buildOperation(get, pathItem.Get.Value, opChan)
	go buildOperation(put, pathItem.Put.Value, opChan)
	go buildOperation(post, pathItem.Post.Value, opChan)
	go buildOperation(del, pathItem.Delete.Value, opChan)
	go buildOperation(options, pathItem.Options.Value, opChan)
	go buildOperation(head, pathItem.Head.Value, opChan)
	go buildOperation(patch, pathItem.Patch.Value, opChan)
	go buildOperation(trace, pathItem.Trace.Value, opChan)

	if !pathItem.Parameters.IsEmpty() {
		params := make([]*Parameter, len(pathItem.Parameters.Value))
		for i := range pathItem.Parameters.Value {
			params[i] = NewParameter(pathItem.Parameters.Value[i].Value)
		}
		pi.Parameters = params
	}

	complete := false
	opCount := 0
	for !complete {
		select {
		case opRes := <-opChan:
			switch opRes.method {
			case get:
				pi.Get = opRes.op
			case put:
				pi.Put = opRes.op
			case post:
				pi.Post = opRes.op
			case del:
				pi.Delete = opRes.op
			case options:
				pi.Options = opRes.op
			case head:
				pi.Head = opRes.op
			case patch:
				pi.Patch = opRes.op
			case trace:
				pi.Trace = opRes.op
			}
		}
		opCount++
		if opCount == 8 {
			complete = true
		}
	}
	return pi
}

// GoLow returns the low level instance of PathItem, used to build the high-level one.
func (p *PathItem) GoLow() *low.PathItem {
	return p.low
}

// GoLowUntyped will return the low-level PathItem instance that was used to create the high-level one, with no type
func (p *PathItem) GoLowUntyped() any {
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
	if p.Trace != nil {
		o[low.TraceLabel] = p.Trace
	}
	return o
}

// Render will return a YAML representation of the PathItem object as a byte slice.
func (p *PathItem) Render() ([]byte, error) {
	return yaml.Marshal(p)
}

func (p *PathItem) RenderInline() ([]byte, error) {
	d, _ := p.MarshalYAMLInline()
	return yaml.Marshal(d)
}

// MarshalYAML will create a ready to render YAML representation of the PathItem object.
func (p *PathItem) MarshalYAML() (interface{}, error) {
	nb := high.NewNodeBuilder(p, p.low)
	return nb.Render(), nil
}

func (p *PathItem) MarshalYAMLInline() (interface{}, error) {
	nb := high.NewNodeBuilder(p, p.low)

	nb.Resolve = true

	return nb.Render(), nil
}
