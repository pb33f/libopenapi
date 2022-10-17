// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"crypto/sha256"
	"fmt"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"strings"
)

// Parameter represents a high-level OpenAPI 3+ Parameter object, that is backed by a low-level one.
//
// A unique parameter is defined by a combination of a name and location.
//  - https://spec.openapis.org/oas/v3.1.0#parameter-object
type Parameter struct {
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
	Example         low.NodeReference[any]
	Examples        low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*base.Example]]
	Content         low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*MediaType]]
	Extensions      map[low.KeyReference[string]]low.ValueReference[any]
}

// FindContent will attempt to locate a MediaType instance using the specified name.
func (p *Parameter) FindContent(cType string) *low.ValueReference[*MediaType] {
	return low.FindItemInMap[*MediaType](cType, p.Content.Value)
}

// FindExample will attempt to locate a base.Example instance using the specified name.
func (p *Parameter) FindExample(eType string) *low.ValueReference[*base.Example] {
	return low.FindItemInMap[*base.Example](eType, p.Examples.Value)
}

// FindExtension attempts to locate an extension using the specified name.
func (p *Parameter) FindExtension(ext string) *low.ValueReference[any] {
	return low.FindItemInMap[any](ext, p.Extensions)
}

// Build will extract examples, extensions and content/media types.
func (p *Parameter) Build(root *yaml.Node, idx *index.SpecIndex) error {
	p.Extensions = low.ExtractExtensions(root)

	// handle example if set.
	_, expLabel, expNode := utils.FindKeyNodeFull(base.ExampleLabel, root.Content)
	if expNode != nil {
		p.Example = low.ExtractExample(expNode, expLabel)
	}

	// handle schema
	sch, sErr := base.ExtractSchema(root, idx)
	if sErr != nil {
		return sErr
	}
	if sch != nil {
		p.Schema = *sch
	}

	// handle examples if set.
	exps, expsL, expsN, eErr := low.ExtractMap[*base.Example](base.ExamplesLabel, root, idx)
	if eErr != nil {
		return eErr
	}
	if exps != nil {
		p.Examples = low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*base.Example]]{
			Value:     exps,
			KeyNode:   expsL,
			ValueNode: expsN,
		}
	}

	// handle content, if set.
	con, cL, cN, cErr := low.ExtractMap[*MediaType](ContentLabel, root, idx)
	if cErr != nil {
		return cErr
	}
	p.Content = low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*MediaType]]{
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
	if p.Schema.Value != nil {
		f = append(f, fmt.Sprintf("%x", p.Schema.Value.Schema().Hash()))
	}
	if p.Example.Value != nil {
		f = append(f, fmt.Sprintf("%x", p.Example.Value))
	}
	if len(p.Examples.Value) > 0 {
		for k := range p.Examples.Value {
			f = append(f, fmt.Sprintf("%s-%x", k.Value, p.Examples.Value[k].Value.Hash()))
		}
	}
	if len(p.Extensions) > 0 {
		for k := range p.Extensions {
			f = append(f, fmt.Sprintf("%v-%x", k.Value, p.Extensions[k].Value))
		}
	}
	if len(p.Content.Value) > 0 {
		for k := range p.Content.Value {
			f = append(f, fmt.Sprintf("%v-%x", k.Value, p.Content.Value[k].Value.Hash()))
		}
	}
	return sha256.Sum256([]byte(strings.Join(f, "|")))
}

// IsParameter compliance methods.

func (p *Parameter) GetName() *low.NodeReference[string] {
	return &p.Name
}
func (p *Parameter) GetIn() *low.NodeReference[string] {
	return &p.In
}
func (p *Parameter) GetType() *low.NodeReference[string] {
	return nil // not implemented
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
		Value:     p.Schema.KeyNode,
	}
	return &i
}
func (p *Parameter) GetFormat() *low.NodeReference[string] {
	return nil
}
func (p *Parameter) GetItems() *low.NodeReference[any] {
	return nil
}
func (p *Parameter) GetStyle() *low.NodeReference[string] {
	return &p.Style
}
func (p *Parameter) GetCollectionFormat() *low.NodeReference[string] {
	return nil
}
func (p *Parameter) GetDefault() *low.NodeReference[any] {
	return nil
}
func (p *Parameter) GetAllowReserved() *low.NodeReference[bool] {
	return &p.AllowReserved
}
func (p *Parameter) GetExplode() *low.NodeReference[bool] {
	return &p.Explode
}
func (p *Parameter) GetMaximum() *low.NodeReference[int] {
	return nil
}
func (p *Parameter) GetExclusiveMaximum() *low.NodeReference[bool] {
	return nil
}
func (p *Parameter) GetMinimum() *low.NodeReference[int] {
	return nil
}
func (p *Parameter) GetExclusiveMinimum() *low.NodeReference[bool] {
	return nil
}
func (p *Parameter) GetMaxLength() *low.NodeReference[int] {
	return nil
}
func (p *Parameter) GetMinLength() *low.NodeReference[int] {
	return nil
}
func (p *Parameter) GetPattern() *low.NodeReference[string] {
	return nil
}
func (p *Parameter) GetMaxItems() *low.NodeReference[int] {
	return nil
}
func (p *Parameter) GetMinItems() *low.NodeReference[int] {
	return nil
}
func (p *Parameter) GetUniqueItems() *low.NodeReference[bool] {
	return nil
}
func (p *Parameter) GetEnum() *low.NodeReference[[]low.ValueReference[string]] {
	return nil
}
func (p *Parameter) GetMultipleOf() *low.NodeReference[int] {
	return nil
}
func (p *Parameter) GetExample() *low.NodeReference[any] {
	return &p.Example
}
func (p *Parameter) GetExamples() *low.NodeReference[any] {
	i := low.NodeReference[any]{
		KeyNode:   p.Examples.KeyNode,
		ValueNode: p.Examples.ValueNode,
		Value:     p.Examples.KeyNode,
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
