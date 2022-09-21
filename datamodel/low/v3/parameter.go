// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
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
