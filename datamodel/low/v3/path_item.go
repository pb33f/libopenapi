// Copyright 2022-2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/pkg-base/libopenapi/datamodel"
	"github.com/pkg-base/libopenapi/datamodel/low"
	"github.com/pkg-base/libopenapi/index"
	"github.com/pkg-base/libopenapi/orderedmap"
	"github.com/pkg-base/libopenapi/utils"
	"github.com/pkg-base/yaml"
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
	Extensions  *orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]]
	KeyNode     *yaml.Node
	RootNode    *yaml.Node
	index       *index.SpecIndex
	context     context.Context
	*low.Reference
	low.NodeMap
}

// GetIndex returns the index.SpecIndex instance attached to the PathItem object.
func (p *PathItem) GetIndex() *index.SpecIndex {
	return p.index
}

// GetContext returns the context.Context instance used when building the PathItem object.
func (p *PathItem) GetContext() context.Context {
	return p.context
}

// Hash will return a consistent SHA256 Hash of the PathItem object
func (p *PathItem) Hash() [32]byte {
	// Use string builder pool
	sb := low.GetStringBuilder()
	defer low.PutStringBuilder(sb)

	if !p.Description.IsEmpty() {
		sb.WriteString(p.Description.Value)
		sb.WriteByte('|')
	}
	if !p.Summary.IsEmpty() {
		sb.WriteString(p.Summary.Value)
		sb.WriteByte('|')
	}
	if !p.Get.IsEmpty() {
		sb.WriteString(fmt.Sprintf("%s-%s", GetLabel, low.GenerateHashString(p.Get.Value)))
		sb.WriteByte('|')
	}
	if !p.Put.IsEmpty() {
		sb.WriteString(fmt.Sprintf("%s-%s", PutLabel, low.GenerateHashString(p.Put.Value)))
		sb.WriteByte('|')
	}
	if !p.Post.IsEmpty() {
		sb.WriteString(fmt.Sprintf("%s-%s", PostLabel, low.GenerateHashString(p.Post.Value)))
		sb.WriteByte('|')
	}
	if !p.Delete.IsEmpty() {
		sb.WriteString(fmt.Sprintf("%s-%s", DeleteLabel, low.GenerateHashString(p.Delete.Value)))
		sb.WriteByte('|')
	}
	if !p.Options.IsEmpty() {
		sb.WriteString(fmt.Sprintf("%s-%s", OptionsLabel, low.GenerateHashString(p.Options.Value)))
		sb.WriteByte('|')
	}
	if !p.Head.IsEmpty() {
		sb.WriteString(fmt.Sprintf("%s-%s", HeadLabel, low.GenerateHashString(p.Head.Value)))
		sb.WriteByte('|')
	}
	if !p.Patch.IsEmpty() {
		sb.WriteString(fmt.Sprintf("%s-%s", PatchLabel, low.GenerateHashString(p.Patch.Value)))
		sb.WriteByte('|')
	}
	if !p.Trace.IsEmpty() {
		sb.WriteString(fmt.Sprintf("%s-%s", TraceLabel, low.GenerateHashString(p.Trace.Value)))
		sb.WriteByte('|')
	}

	// Process Parameters with pre-allocation and sorting
	if len(p.Parameters.Value) > 0 {
		keys := make([]string, len(p.Parameters.Value))
		for k := range p.Parameters.Value {
			keys[k] = low.GenerateHashString(p.Parameters.Value[k].Value)
		}
		sort.Strings(keys)
		for _, key := range keys {
			sb.WriteString(key)
			sb.WriteByte('|')
		}
	}

	// Process Servers with pre-allocation and sorting
	if len(p.Servers.Value) > 0 {
		keys := make([]string, len(p.Servers.Value))
		for k := range p.Servers.Value {
			keys[k] = low.GenerateHashString(p.Servers.Value[k].Value)
		}
		sort.Strings(keys)
		for _, key := range keys {
			sb.WriteString(key)
			sb.WriteByte('|')
		}
	}

	for _, ext := range low.HashExtensions(p.Extensions) {
		sb.WriteString(ext)
		sb.WriteByte('|')
	}
	return sha256.Sum256([]byte(sb.String()))
}

// GetRootNode returns the root yaml node of the PathItem object
func (p *PathItem) GetRootNode() *yaml.Node {
	return p.RootNode
}

// GetKeyNode returns the key yaml node of the PathItem object
func (p *PathItem) GetKeyNode() *yaml.Node {
	return p.KeyNode
}

// FindExtension attempts to find an extension
func (p *PathItem) FindExtension(ext string) *low.ValueReference[*yaml.Node] {
	return low.FindItemInOrderedMap(ext, p.Extensions)
}

// GetExtensions returns all PathItem extensions and satisfies the low.HasExtensions interface.
func (p *PathItem) GetExtensions() *orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]] {
	return p.Extensions
}

// Build extracts extensions, parameters, servers and each http method defined.
// everything is extracted asynchronously for speed.
func (p *PathItem) Build(ctx context.Context, keyNode, root *yaml.Node, idx *index.SpecIndex) error {
	p.Reference = new(low.Reference)
	if ok, _, ref := utils.IsNodeRefValue(root); ok {
		p.SetReference(ref, root)
	}
	root = utils.NodeAlias(root)
	p.KeyNode = keyNode
	p.RootNode = root
	utils.CheckForMergeNodes(root)
	p.Nodes = low.ExtractNodes(ctx, root)
	p.Extensions = low.ExtractExtensions(root)
	p.index = idx
	p.context = ctx

	low.ExtractExtensionNodes(ctx, p.Extensions, p.Nodes)
	skip := false
	var currentNode *yaml.Node

	var wg sync.WaitGroup
	var errors []error
	var ops []low.NodeReference[*Operation]

	// extract parameters
	params, ln, vn, pErr := low.ExtractArray[*Parameter](ctx, ParametersLabel, root, idx)
	if pErr != nil {
		return pErr
	}
	if params != nil {
		p.Parameters = low.NodeReference[[]low.ValueReference[*Parameter]]{
			Value:     params,
			KeyNode:   ln,
			ValueNode: vn,
		}
		p.Nodes.Store(ln.Line, ln)
	}

	_, ln, vn = utils.FindKeyNodeFullTop(ServersLabel, root.Content)
	if vn != nil {
		if utils.IsNodeArray(vn) {
			var servers []low.ValueReference[*Server]
			for _, srvN := range vn.Content {
				if utils.IsNodeMap(srvN) {
					srvr := new(Server)
					_ = low.BuildModel(srvN, srvr)
					srvr.Build(ctx, ln, srvN, idx)
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
			p.Nodes.Store(ln.Line, ln)
		}
	}
	prevExt := false
	for i, pathNode := range root.Content {
		if strings.HasPrefix(strings.ToLower(pathNode.Value), "x-") {
			skip = true
			prevExt = true
			continue
		}
		// https://github.com/pkg-base/libopenapi/issues/388
		// in the case where a user has an extension with the value 'parameters', make sure we handle
		// it correctly, by not skipping.
		if strings.HasPrefix(strings.ToLower(pathNode.Value), "parameters") {
			if !prevExt { // this
				skip = true
				continue
			} else {
				prevExt = false
			}
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
		case PostLabel:
		case PutLabel:
		case PatchLabel:
		case DeleteLabel:
		case HeadLabel:
		case OptionsLabel:
		case TraceLabel:
		default:
			continue // ignore everything else.
		}

		foundContext := ctx
		var op Operation
		opIsRef := false
		var opRefVal string
		var opRefNode *yaml.Node
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
			opRefNode = pathNode
			r, newIdx, err, nCtx := low.LocateRefNodeWithContext(ctx, pathNode, idx)
			if r != nil {
				if r.Kind == yaml.DocumentNode {
					r = r.Content[0]
				}
				pathNode = r
				foundContext = nCtx
				foundContext = context.WithValue(foundContext, index.FoundIndexKey, newIdx)

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
		} else {
			foundContext = context.WithValue(foundContext, index.FoundIndexKey, idx)
		}
		wg.Add(1)
		low.BuildModelAsync(pathNode, &op, &wg, &errors)

		opRef := low.NodeReference[*Operation]{
			Value:     &op,
			KeyNode:   currentNode,
			ValueNode: pathNode,
			Context:   foundContext,
		}
		if opIsRef {
			opRef.SetReference(opRefVal, opRefNode)
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
	translateFunc := func(_ int, op low.NodeReference[*Operation]) (any, error) {
		ref := ""
		var refNode *yaml.Node
		if op.IsReference() {
			ref = op.GetReference()
			refNode = op.GetReferenceNode()
		}

		err := op.Value.Build(op.Context, op.KeyNode, op.ValueNode, op.Context.Value(index.FoundIndexKey).(*index.SpecIndex))
		if ref != "" {
			op.Value.Reference.SetReference(ref, refNode)
		}
		if err != nil {
			return nil, err
		}
		return nil, nil
	}
	err := datamodel.TranslateSliceParallel[low.NodeReference[*Operation], any](ops, translateFunc, nil)
	if err != nil {
		return err
	}
	return nil
}
