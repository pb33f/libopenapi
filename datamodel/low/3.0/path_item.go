// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"fmt"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"strings"
	"sync"
)

type PathItem struct {
	Description low.NodeReference[string]
	Summary     low.NodeReference[string]
	Get         low.NodeReference[*Operation]
	Put         low.NodeReference[*Operation]
	Post        low.NodeReference[*Operation]
	Delete      low.NodeReference[*Operation]
	Options     low.NodeReference[*Operation]
	Head        low.NodeReference[*Operation]
	Patch       low.NodeReference[*Operation]
	Trace       low.NodeReference[*Operation]
	Servers     low.NodeReference[[]low.ValueReference[*Server]]
	Parameters  low.NodeReference[[]low.ValueReference[*Parameter]]
	Extensions  map[low.KeyReference[string]]low.ValueReference[any]
}

func (p *PathItem) FindExtension(ext string) *low.ValueReference[any] {
	return low.FindItemInMap[any](ext, p.Extensions)
}

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

	_, ln, vn = utils.FindKeyNodeFull(ServersLabel, root.Content)
	if vn != nil {
		if utils.IsNodeArray(vn) {
			var servers []low.ValueReference[*Server]
			for _, srvN := range vn.Content {
				if utils.IsNodeMap(srvN) {
					srvr := new(Server)
					_ = low.BuildModel(srvN, srvr)
					srvr.Build(srvN, idx)
					servers = append(servers, low.ValueReference[*Server]{
						Value:     srvr,
						ValueNode: srvN,
					})
				}
			}
			p.Servers = low.NodeReference[[]low.ValueReference[*Server]]{
				Value:     servers,
				KeyNode:   ln,
				ValueNode: vn,
			}
		}
	}

	for i, pathNode := range root.Content {
		if strings.HasPrefix(strings.ToLower(pathNode.Value), "x-") {
			skip = true
			continue
		}
		if strings.HasPrefix(strings.ToLower(pathNode.Value), "parameters") {
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
		case TraceLabel:
			break
		default:
			continue // ignore everything else.
		}

		var op Operation

		wg.Add(1)

		if ok, _, _ := utils.IsNodeRefValue(pathNode); ok {
			r := low.LocateRefNode(pathNode, idx)
			if r != nil {
				pathNode = r
			} else {
				return fmt.Errorf("path item build failed: cannot find reference: %s at line %d, col %d",
					pathNode.Content[1].Value, pathNode.Content[1].Line, pathNode.Content[1].Column)
			}
		}

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
		case TraceLabel:
			p.Trace = opRef
		}
	}

	//all operations have been superficially built,
	//now we need to build out the operation, we will do this asynchronously for speed.
	opBuildChan := make(chan bool)
	opErrorChan := make(chan error)

	var buildOpFunc = func(op low.NodeReference[*Operation], ch chan<- bool, errCh chan<- error) {

		//build out the operation.
		if ok, _, _ := utils.IsNodeRefValue(op.ValueNode); ok {
			r := low.LocateRefNode(op.ValueNode, idx)
			if r != nil {
				op.ValueNode = r
			} else {
				// any reference would be the second node.
				errCh <- fmt.Errorf("cannot extract reference: %s", op.ValueNode.Content[1].Value)
			}
		}

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
