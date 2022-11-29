// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
    "github.com/pb33f/libopenapi/datamodel/high"
    low "github.com/pb33f/libopenapi/datamodel/low/v3"
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
//  - https://spec.openapis.org/oas/v3.1.0#path-item-object
type PathItem struct {
    Description string
    Summary     string
    Get         *Operation
    Put         *Operation
    Post        *Operation
    Delete      *Operation
    Options     *Operation
    Head        *Operation
    Patch       *Operation
    Trace       *Operation
    Servers     []*Server
    Parameters  []*Parameter
    Extensions  map[string]any
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
