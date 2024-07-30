// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

// Parameter represents a high-level OpenAPI 3+ Parameter object, that is backed by a low-level one.
//
// A unique parameter is defined by a combination of a name and location.
//   - https://spec.openapis.org/oas/v3.1.0#parameter-object
type Parameter struct {
	KeyNode         *yaml.Node
	RootNode        *yaml.Node
	Name            low.NodeReference[string]
	In              low.NodeReference[string]
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
	*low.Reference
}

// GetRootNode returns the root yaml node of the Parameter object.
func (p *Parameter) GetRootNode() *yaml.Node {
	return p.RootNode
}

// GetKeyNode returns the key yaml node of the Parameter object.
func (p *Parameter) GetKeyNode() *yaml.Node {
	return p.KeyNode
}

// FindContent will attempt to locate a MediaType instance using the specified name.
func (p *Parameter) FindContent(cType string) *low.ValueReference[*MediaType] {
	return low.FindItemInOrderedMap[*MediaType](cType, p.Content.Value)
}

// FindExample will attempt to locate a base.Example instance using the specified name.
func (p *Parameter) FindExample(eType string) *low.ValueReference[*base.Example] {
	return low.FindItemInOrderedMap[*base.Example](eType, p.Examples.Value)
}

// FindExtension attempts to locate an extension using the specified name.
func (p *Parameter) FindExtension(ext string) *low.ValueReference[*yaml.Node] {
	return low.FindItemInOrderedMap(ext, p.Extensions)
}

// GetExtensions returns all extensions for Parameter.
func (p *Parameter) GetExtensions() *orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]] {
	return p.Extensions
}

// Build will extract examples, extensions and content/media types.
func (p *Parameter) Build(ctx context.Context, keyNode, root *yaml.Node, idx *index.SpecIndex) error {
	root = utils.NodeAlias(root)
	p.KeyNode = keyNode
	p.RootNode = root
	utils.CheckForMergeNodes(root)
	p.Reference = new(low.Reference)
	p.Extensions = low.ExtractExtensions(root)

	// handle example if set.
	_, expLabel, expNode := utils.FindKeyNodeFullTop(base.ExampleLabel, root.Content)
	if expNode != nil {
		p.Example = low.NodeReference[*yaml.Node]{Value: expNode, KeyNode: expLabel, ValueNode: expNode}
	}

	// handle schema
	sch, sErr := base.ExtractSchema(ctx, root, idx)
	if sErr != nil {
		return sErr
	}
	if sch != nil {
		p.Schema = *sch
	}

	// handle examples if set.
	exps, expsL, expsN, eErr := low.ExtractMap[*base.Example](ctx, base.ExamplesLabel, root, idx)
	if eErr != nil {
		return eErr
	}
	// Only consider examples if they are defined in the root node.
	if exps != nil && slices.Contains(root.Content, expsL) {
		p.Examples = low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[*base.Example]]]{
			Value:     exps,
			KeyNode:   expsL,
			ValueNode: expsN,
		}
	}

	// handle content, if set.
	con, cL, cN, cErr := low.ExtractMap[*MediaType](ctx, ContentLabel, root, idx)
	if cErr != nil {
		return cErr
	}
	p.Content = low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[*MediaType]]]{
		Value:     con,
		KeyNode:   cL,
		ValueNode: cN,
	}
	return nil
}

// Hash will return a consistent SHA256 Hash of the Parameter object
func (p *Parameter) Hash() [32]byte {
	var f []string
	if p.Name.Value != "" {
		f = append(f, p.Name.Value)
	}
	if p.In.Value != "" {
		f = append(f, p.In.Value)
	}
	if p.Description.Value != "" {
		f = append(f, p.Description.Value)
	}
	f = append(f, fmt.Sprint(p.Required.Value))
	f = append(f, fmt.Sprint(p.Deprecated.Value))
	f = append(f, fmt.Sprint(p.AllowEmptyValue.Value))
	if p.Style.Value != "" {
		f = append(f, fmt.Sprint(p.Style.Value))
	}
	f = append(f, fmt.Sprint(p.Explode.Value))
	f = append(f, fmt.Sprint(p.AllowReserved.Value))
	if p.Schema.Value != nil && p.Schema.Value.Schema() != nil {
		f = append(f, fmt.Sprintf("%x", p.Schema.Value.Schema().Hash()))
	}
	if p.Example.Value != nil && !p.Example.Value.IsZero() {
		f = append(f, low.GenerateHashString(p.Example.Value))
	}
	for pair := orderedmap.First(orderedmap.SortAlpha(p.Examples.Value)); pair != nil; pair = pair.Next() {
		f = append(f, low.GenerateHashString(pair.Value().Value))
	}
	for pair := orderedmap.First(orderedmap.SortAlpha(p.Content.Value)); pair != nil; pair = pair.Next() {
		f = append(f, low.GenerateHashString(pair.Value().Value))
	}
	f = append(f, low.HashExtensions(p.Extensions)...)
	return sha256.Sum256([]byte(strings.Join(f, "|")))
}

// IsParameter compliance methods.

func (p *Parameter) GetName() *low.NodeReference[string] {
	return &p.Name
}

func (p *Parameter) GetIn() *low.NodeReference[string] {
	return &p.In
}

func (p *Parameter) GetDescription() *low.NodeReference[string] {
	return &p.Description
}

func (p *Parameter) GetRequired() *low.NodeReference[bool] {
	return &p.Required
}

func (p *Parameter) GetDeprecated() *low.NodeReference[bool] {
	return &p.Deprecated
}

func (p *Parameter) GetAllowEmptyValue() *low.NodeReference[bool] {
	return &p.AllowEmptyValue
}

func (p *Parameter) GetSchema() *low.NodeReference[any] {
	i := low.NodeReference[any]{
		KeyNode:   p.Schema.KeyNode,
		ValueNode: p.Schema.ValueNode,
		Value:     p.Schema.Value,
	}
	return &i
}

func (p *Parameter) GetStyle() *low.NodeReference[string] {
	return &p.Style
}

func (p *Parameter) GetAllowReserved() *low.NodeReference[bool] {
	return &p.AllowReserved
}

func (p *Parameter) GetExplode() *low.NodeReference[bool] {
	return &p.Explode
}

func (p *Parameter) GetExample() *low.NodeReference[*yaml.Node] {
	return &p.Example
}

func (p *Parameter) GetExamples() *low.NodeReference[any] {
	i := low.NodeReference[any]{
		KeyNode:   p.Examples.KeyNode,
		ValueNode: p.Examples.ValueNode,
		Value:     p.Examples.Value,
	}
	return &i
}

func (p *Parameter) GetContent() *low.NodeReference[any] {
	c := low.NodeReference[any]{
		KeyNode:   p.Content.KeyNode,
		ValueNode: p.Content.ValueNode,
		Value:     p.Content.Value,
	}
	return &c
}
