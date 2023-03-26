// Copyright 2022-2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

// PathItem represents a low-level OpenAPI 3+ PathItem object.
//
// Describes the operations available on a single path. A Path Item MAY be empty, due to ACL constraints.
// The path itself is still exposed to the documentation viewer, but they will not know which operations and parameters
// are available.
//   - https://spec.openapis.org/oas/v3.1.0#path-item-object
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
	*low.Reference
}

// Hash will return a consistent SHA256 Hash of the PathItem object
func (p *PathItem) Hash() [32]byte {
	var f []string
	if !p.Description.IsEmpty() {
		f = append(f, p.Description.Value)
	}
	if !p.Summary.IsEmpty() {
		f = append(f, p.Summary.Value)
	}
	if !p.Get.IsEmpty() {
		f = append(f, fmt.Sprintf("%s-%s", GetLabel, low.GenerateHashString(p.Get.Value)))
	}
	if !p.Put.IsEmpty() {
		f = append(f, fmt.Sprintf("%s-%s", PutLabel, low.GenerateHashString(p.Put.Value)))
	}
	if !p.Post.IsEmpty() {
		f = append(f, fmt.Sprintf("%s-%s", PutLabel, low.GenerateHashString(p.Post.Value)))
	}
	if !p.Delete.IsEmpty() {
		f = append(f, fmt.Sprintf("%s-%s", DeleteLabel, low.GenerateHashString(p.Delete.Value)))
	}
	if !p.Options.IsEmpty() {
		f = append(f, fmt.Sprintf("%s-%s", OptionsLabel, low.GenerateHashString(p.Options.Value)))
	}
	if !p.Head.IsEmpty() {
		f = append(f, fmt.Sprintf("%s-%s", HeadLabel, low.GenerateHashString(p.Head.Value)))
	}
	if !p.Patch.IsEmpty() {
		f = append(f, fmt.Sprintf("%s-%s", PatchLabel, low.GenerateHashString(p.Patch.Value)))
	}
	if !p.Trace.IsEmpty() {
		f = append(f, fmt.Sprintf("%s-%s", TraceLabel, low.GenerateHashString(p.Trace.Value)))
	}
	keys := make([]string, len(p.Parameters.Value))
	for k := range p.Parameters.Value {
		keys[k] = low.GenerateHashString(p.Parameters.Value[k].Value)
	}
	sort.Strings(keys)
	f = append(f, keys...)
	keys = make([]string, len(p.Servers.Value))
	for k := range p.Servers.Value {
		keys[k] = low.GenerateHashString(p.Servers.Value[k].Value)
	}
	sort.Strings(keys)
	f = append(f, keys...)

	keys = make([]string, len(p.Extensions))
	z := 0
	for k := range p.Extensions {
		keys[z] = fmt.Sprintf("%s-%x", k.Value, sha256.Sum256([]byte(fmt.Sprint(p.Extensions[k].Value))))
		z++
	}
	sort.Strings(keys)
	f = append(f, keys...)
	return sha256.Sum256([]byte(strings.Join(f, "|")))
}

// FindExtension attempts to find an extension
func (p *PathItem) FindExtension(ext string) *low.ValueReference[any] {
	return low.FindItemInMap[any](ext, p.Extensions)
}

// GetExtensions returns all PathItem extensions and satisfies the low.HasExtensions interface.
func (p *PathItem) GetExtensions() map[low.KeyReference[string]]low.ValueReference[any] {
	return p.Extensions
}

// Build extracts extensions, parameters, servers and each http method defined.
// everything is extracted asynchronously for speed.
func (p *PathItem) Build(root *yaml.Node, idx *index.SpecIndex) error {
	p.Reference = new(low.Reference)
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

	_, ln, vn = utils.FindKeyNodeFullTop(ServersLabel, root.Content)
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
		opIsRef := false
		var opRefVal string
		if ok, _, ref := utils.IsNodeRefValue(pathNode); ok {
			// According to OpenAPI spec the only valid $ref for paths is
			// reference for the whole pathItem. Unfortunately, internet is full of invalid specs
			// even from trusted companies like DigitalOcean where they tend to
			// use file $ref for each respective operation:
			// /endpoint/call/name:
			//   post:
			//     $ref: 'file.yaml'
			// Check if that is the case and resolve such thing properly too.

			opIsRef = true
			opRefVal = ref
			r, err := low.LocateRefNode(pathNode, idx)
			if r != nil {
				if r.Kind == yaml.DocumentNode {
					r = r.Content[0]
				}
				pathNode = r
				if r.Tag == "" {
					// If it's a node from file, tag is empty
					pathNode = r.Content[0]
				}

				if err != nil {
					if !idx.AllowCircularReferenceResolving() {
						return fmt.Errorf("build schema failed: %s", err.Error())
					}
				}
			} else {
				return fmt.Errorf("path item build failed: cannot find reference: %s at line %d, col %d",
					pathNode.Content[1].Value, pathNode.Content[1].Line, pathNode.Content[1].Column)
			}
		}
		wg.Add(1)
		go low.BuildModelAsync(pathNode, &op, &wg, &errors)

		opRef := low.NodeReference[*Operation]{
			Value:     &op,
			KeyNode:   currentNode,
			ValueNode: pathNode,
		}
		if opIsRef {
			opRef.Reference = opRefVal
			opRef.ReferenceNode = true
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

	// all operations have been superficially built,
	// now we need to build out the operation, we will do this asynchronously for speed.
	opBuildChan := make(chan bool)
	opErrorChan := make(chan error)

	buildOpFunc := func(op low.NodeReference[*Operation], ch chan<- bool, errCh chan<- error, ref string) {
		er := op.Value.Build(op.ValueNode, idx)
		if ref != "" {
			op.Value.Reference.Reference = ref
		}
		if er != nil {
			errCh <- er
		}
		ch <- true
	}

	if len(ops) <= 0 {
		return nil // nothing to do.
	}

	for _, op := range ops {
		ref := ""
		if op.ReferenceNode {
			ref = op.Reference
		}
		go buildOpFunc(op, opBuildChan, opErrorChan, ref)
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
