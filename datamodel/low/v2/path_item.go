// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"gopkg.in/yaml.v3"
	"strings"
	"sync"
)

// PathItem represents a low-level Swagger / OpenAPI 2 PathItem object.
//
// Describes the operations available on a single path. A Path Item may be empty, due to ACL constraints.
// The path itself is still exposed to the tooling, but will not know which operations and parameters
// are available.
//
//  - https://swagger.io/specification/v2/#pathItemObject
type PathItem struct {
	Ref        low.NodeReference[string]
	Get        low.NodeReference[*Operation]
	Put        low.NodeReference[*Operation]
	Post       low.NodeReference[*Operation]
	Delete     low.NodeReference[*Operation]
	Options    low.NodeReference[*Operation]
	Head       low.NodeReference[*Operation]
	Patch      low.NodeReference[*Operation]
	Parameters low.NodeReference[[]low.ValueReference[*Parameter]]
	Extensions map[low.KeyReference[string]]low.ValueReference[any]
}

// FindExtension will attempt to locate an extension given a name.
func (p *PathItem) FindExtension(ext string) *low.ValueReference[any] {
	return low.FindItemInMap[any](ext, p.Extensions)
}

// Build will extract extensions, parameters and operations for all methods. Every method is handled
// asynchronously, in order to keep things moving quickly for complex operations.
func (p *PathItem) Build(root *yaml.Node, idx *index.SpecIndex) error {
	p.Extensions = low.ExtractExtensions(root)
	skip := false
	var currentNode *yaml.Node

	var wg sync.WaitGroup
	var errors []error

	var ops []low.NodeReference[*Operation]

	// extract parameters
	params, ln, vn, pErr := low.ExtractArray[*Parameter](ParametersLabel, root, idx)
	if pErr != nil {
		return pErr
	}
	if params != nil {
		p.Parameters = low.NodeReference[[]low.ValueReference[*Parameter]]{
			Value:     params,
			KeyNode:   ln,
			ValueNode: vn,
		}
	}

	for i, pathNode := range root.Content {
		if strings.HasPrefix(strings.ToLower(pathNode.Value), "x-") {
			skip = true
			continue
		}
		// because (for some reason) the spec for swagger docs allows for a '$ref' property for path items.
		// this is kinda nuts, because '$ref' is a reserved keyword for JSON references, which is ALSO used
		// in swagger. Why this choice was made, I do not know.
		if strings.Contains(strings.ToLower(pathNode.Value), "$ref") {
			rn := root.Content[i+1]
			p.Ref = low.NodeReference[string]{
				Value:     rn.Value,
				ValueNode: rn,
				KeyNode:   pathNode,
			}
			skip = true
			continue
		}
		if skip {
			skip = false
			continue
		}
		if i%2 == 0 {
			currentNode = pathNode
			continue
		}

		// the only thing we now care about is handling operations, filter out anything that's not a verb.
		switch currentNode.Value {
		case GetLabel:
			break
		case PostLabel:
			break
		case PutLabel:
			break
		case PatchLabel:
			break
		case DeleteLabel:
			break
		case HeadLabel:
			break
		case OptionsLabel:
			break
		default:
			continue // ignore everything else.
		}

		var op Operation

		wg.Add(1)

		go low.BuildModelAsync(pathNode, &op, &wg, &errors)

		opRef := low.NodeReference[*Operation]{
			Value:     &op,
			KeyNode:   currentNode,
			ValueNode: pathNode,
		}

		ops = append(ops, opRef)

		switch currentNode.Value {
		case GetLabel:
			p.Get = opRef
		case PostLabel:
			p.Post = opRef
		case PutLabel:
			p.Put = opRef
		case PatchLabel:
			p.Patch = opRef
		case DeleteLabel:
			p.Delete = opRef
		case HeadLabel:
			p.Head = opRef
		case OptionsLabel:
			p.Options = opRef
		}
	}

	//all operations have been superficially built,
	//now we need to build out the operation, we will do this asynchronously for speed.
	opBuildChan := make(chan bool)
	opErrorChan := make(chan error)

	var buildOpFunc = func(op low.NodeReference[*Operation], ch chan<- bool, errCh chan<- error) {
		er := op.Value.Build(op.ValueNode, idx)
		if er != nil {
			errCh <- er
		}
		ch <- true
	}

	if len(ops) <= 0 {
		return nil // nothing to do.
	}

	for _, op := range ops {
		go buildOpFunc(op, opBuildChan, opErrorChan)
	}

	n := 0
	total := len(ops)
	for n < total {
		select {
		case buildError := <-opErrorChan:
			return buildError
		case <-opBuildChan:
			n++
		}
	}

	// make sure we don't exit before the path is finished building.
	if len(ops) > 0 {
		wg.Wait()
	}

	return nil
}
