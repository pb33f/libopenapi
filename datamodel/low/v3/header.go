// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strconv"

	"github.com/pkg-base/libopenapi/datamodel/low"
	"github.com/pkg-base/libopenapi/datamodel/low/base"
	"github.com/pkg-base/libopenapi/index"
	"github.com/pkg-base/libopenapi/orderedmap"
	"github.com/pkg-base/libopenapi/utils"
	"github.com/pkg-base/yaml"
)

// Header represents a low-level OpenAPI 3+ Header object.
//   - https://spec.openapis.org/oas/v3.1.0#header-object
type Header struct {
	Description     low.NodeReference[string]
	Required        low.NodeReference[bool]
	Deprecated      low.NodeReference[bool]
	AllowEmptyValue low.NodeReference[bool]
	Style           low.NodeReference[string]
	Explode         low.NodeReference[bool]
	AllowReserved   low.NodeReference[bool]
	Schema          low.NodeReference[*base.SchemaProxy]
	Example         low.NodeReference[*yaml.Node]
	Examples        low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[*base.Example]]]
	Content         low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[*MediaType]]]
	Extensions      *orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]]
	KeyNode         *yaml.Node
	RootNode        *yaml.Node
	index           *index.SpecIndex
	context         context.Context
	*low.Reference
	low.NodeMap
}

// GetIndex returns the index.SpecIndex instance attached to the Header object
func (h *Header) GetIndex() *index.SpecIndex {
	return h.index
}

// GetContext returns the context.Context instance used when building the Header object
func (h *Header) GetContext() context.Context {
	return h.context
}

// FindExtension will attempt to locate an extension with the supplied name
func (h *Header) FindExtension(ext string) *low.ValueReference[*yaml.Node] {
	return low.FindItemInOrderedMap(ext, h.Extensions)
}

// FindExample will attempt to locate an Example with a specified name
func (h *Header) FindExample(eType string) *low.ValueReference[*base.Example] {
	return low.FindItemInOrderedMap[*base.Example](eType, h.Examples.Value)
}

// FindContent will attempt to locate a MediaType definition, with a specified name
func (h *Header) FindContent(ext string) *low.ValueReference[*MediaType] {
	return low.FindItemInOrderedMap[*MediaType](ext, h.Content.Value)
}

// GetRootNode returns the root yaml node of the Header object
func (h *Header) GetRootNode() *yaml.Node {
	return h.RootNode
}

// GetKeyNode returns the key yaml node of the Header object
func (h *Header) GetKeyNode() *yaml.Node {
	return h.KeyNode
}

// GetExtensions returns all Header extensions and satisfies the low.HasExtensions interface.
func (h *Header) GetExtensions() *orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]] {
	return h.Extensions
}

// Hash will return a consistent SHA256 Hash of the Header object
func (h *Header) Hash() [32]byte {
	// Use string builder pool
	sb := low.GetStringBuilder()
	defer low.PutStringBuilder(sb)

	if h.Description.Value != "" {
		sb.WriteString(h.Description.Value)
		sb.WriteByte('|')
	}
	sb.WriteString(strconv.FormatBool(h.Required.Value))
	sb.WriteByte('|')
	sb.WriteString(strconv.FormatBool(h.Deprecated.Value))
	sb.WriteByte('|')
	sb.WriteString(strconv.FormatBool(h.AllowEmptyValue.Value))
	sb.WriteByte('|')
	if h.Style.Value != "" {
		sb.WriteString(h.Style.Value)
		sb.WriteByte('|')
	}
	sb.WriteString(strconv.FormatBool(h.Explode.Value))
	sb.WriteByte('|')
	sb.WriteString(strconv.FormatBool(h.AllowReserved.Value))
	sb.WriteByte('|')
	if h.Schema.Value != nil {
		sb.WriteString(low.GenerateHashString(h.Schema.Value))
		sb.WriteByte('|')
	}
	if h.Example.Value != nil && !h.Example.Value.IsZero() {
		sb.WriteString(low.GenerateHashString(h.Example.Value))
		sb.WriteByte('|')
	}
	for k, v := range orderedmap.SortAlpha(h.Examples.Value).FromOldest() {
		sb.WriteString(fmt.Sprintf("%s-%x", k.Value, v.Value.Hash()))
		sb.WriteByte('|')
	}
	for k, v := range orderedmap.SortAlpha(h.Content.Value).FromOldest() {
		sb.WriteString(fmt.Sprintf("%s-%x", k.Value, v.Value.Hash()))
		sb.WriteByte('|')
	}
	for _, ext := range low.HashExtensions(h.Extensions) {
		sb.WriteString(ext)
		sb.WriteByte('|')
	}
	return sha256.Sum256([]byte(sb.String()))
}

// Build will extract extensions, examples, schema and content/media types from node.
func (h *Header) Build(ctx context.Context, keyNode, root *yaml.Node, idx *index.SpecIndex) error {
	h.KeyNode = keyNode
	h.Reference = new(low.Reference)
	if ok, _, ref := utils.IsNodeRefValue(root); ok {
		h.SetReference(ref, root)
	}
	root = utils.NodeAlias(root)
	h.RootNode = root
	utils.CheckForMergeNodes(root)
	h.Nodes = low.ExtractNodes(ctx, root)
	h.Extensions = low.ExtractExtensions(root)
	h.context = ctx
	h.index = idx

	low.ExtractExtensionNodes(ctx, h.Extensions, h.Nodes)
	// handle example if set.
	_, expLabel, expNode := utils.FindKeyNodeFull(base.ExampleLabel, root.Content)
	if expNode != nil {
		h.Example = low.NodeReference[*yaml.Node]{
			Value:     expNode,
			ValueNode: expNode,
			KeyNode:   expLabel,
		}
		h.Nodes.Store(expLabel.Line, expLabel)
		m := low.ExtractNodes(ctx, expNode)
		m.Range(func(key, value any) bool {
			h.Nodes.Store(key, value)
			return true
		})
	}

	// handle examples if set.
	exps, expsL, expsN, eErr := low.ExtractMap[*base.Example](ctx, base.ExamplesLabel, root, idx)
	if eErr != nil {
		return eErr
	}
	if exps != nil {
		h.Examples = low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[*base.Example]]]{
			Value:     exps,
			KeyNode:   expsL,
			ValueNode: expsN,
		}
		h.Nodes.Store(expsL.Line, expsL)
	}

	// handle schema
	sch, sErr := base.ExtractSchema(ctx, root, idx)
	if sErr != nil {
		return sErr
	}
	if sch != nil {
		h.Schema = *sch
	}

	// handle content, if set.
	con, cL, cN, cErr := low.ExtractMap[*MediaType](ctx, ContentLabel, root, idx)
	if cErr != nil {
		return cErr
	}
	h.Content = low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[*MediaType]]]{
		Value:     con,
		KeyNode:   cL,
		ValueNode: cN,
	}
	if cL != nil {
		h.Nodes.Store(cL.Line, cL)
	}
	return nil
}

// Getter methods to satisfy OpenAPIHeader interface.

func (h *Header) GetDescription() *low.NodeReference[string] {
	return &h.Description
}

func (h *Header) GetRequired() *low.NodeReference[bool] {
	return &h.Required
}

func (h *Header) GetDeprecated() *low.NodeReference[bool] {
	return &h.Deprecated
}

func (h *Header) GetAllowEmptyValue() *low.NodeReference[bool] {
	return &h.AllowEmptyValue
}

func (h *Header) GetSchema() *low.NodeReference[any] {
	i := low.NodeReference[any]{
		KeyNode:   h.Schema.KeyNode,
		ValueNode: h.Schema.ValueNode,
		Value:     h.Schema.Value,
	}
	return &i
}

func (h *Header) GetStyle() *low.NodeReference[string] {
	return &h.Style
}

func (h *Header) GetAllowReserved() *low.NodeReference[bool] {
	return &h.AllowReserved
}

func (h *Header) GetExplode() *low.NodeReference[bool] {
	return &h.Explode
}

func (h *Header) GetExample() *low.NodeReference[*yaml.Node] {
	return &h.Example
}

func (h *Header) GetExamples() *low.NodeReference[any] {
	i := low.NodeReference[any]{
		KeyNode:   h.Examples.KeyNode,
		ValueNode: h.Examples.ValueNode,
		Value:     h.Examples.Value,
	}
	return &i
}

func (h *Header) GetContent() *low.NodeReference[any] {
	c := low.NodeReference[any]{
		KeyNode:   h.Content.KeyNode,
		ValueNode: h.Content.ValueNode,
		Value:     h.Content.Value,
	}
	return &c
}
