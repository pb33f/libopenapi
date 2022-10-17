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

// Header represents a low-level OpenAPI 3+ Header object.
//  - https://spec.openapis.org/oas/v3.1.0#header-object
type Header struct {
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

// FindExtension will attempt to locate an extension with the supplied name
func (h *Header) FindExtension(ext string) *low.ValueReference[any] {
	return low.FindItemInMap[any](ext, h.Extensions)
}

// FindExample will attempt to locate an Example with a specified name
func (h *Header) FindExample(eType string) *low.ValueReference[*base.Example] {
	return low.FindItemInMap[*base.Example](eType, h.Examples.Value)
}

// FindContent will attempt to locate a MediaType definition, with a specified name
func (h *Header) FindContent(ext string) *low.ValueReference[*MediaType] {
	return low.FindItemInMap[*MediaType](ext, h.Content.Value)
}

// Hash will return a consistent SHA256 Hash of the Header object
func (h *Header) Hash() [32]byte {
	var f []string
	if h.Description.Value != "" {
		f = append(f, h.Description.Value)
	}
	f = append(f, fmt.Sprint(sha256.Sum256([]byte(fmt.Sprint(h.Required.Value)))))
	f = append(f, fmt.Sprint(sha256.Sum256([]byte(fmt.Sprint(h.Deprecated.Value)))))
	f = append(f, fmt.Sprint(sha256.Sum256([]byte(fmt.Sprint(h.AllowEmptyValue.Value)))))
	if h.Style.Value != "" {
		f = append(f, h.Style.Value)
	}
	f = append(f, fmt.Sprint(sha256.Sum256([]byte(fmt.Sprint(h.Explode.Value)))))
	f = append(f, fmt.Sprint(sha256.Sum256([]byte(fmt.Sprint(h.AllowReserved.Value)))))
	if h.Schema.Value != nil {
		f = append(f, fmt.Sprint(h.Schema.Value.Schema().Hash()))
	}
	if h.Example.Value != nil {
		f = append(f, fmt.Sprint(sha256.Sum256([]byte(fmt.Sprint(h.Example.Value)))))
	}
	if len(h.Examples.Value) > 0 {
		for k := range h.Examples.Value {
			f = append(f, fmt.Sprintf("%s-%x", k.Value, h.Examples.Value[k].Value.Hash()))
		}
	}
	if len(h.Content.Value) > 0 {
		for k := range h.Content.Value {
			f = append(f, fmt.Sprintf("%s-%x", k.Value, h.Content.Value[k].Value.Hash()))
		}
	}
	for k := range h.Extensions {
		f = append(f, fmt.Sprintf("%s-%v", k.Value, h.Extensions[k].Value))
	}
	return sha256.Sum256([]byte(strings.Join(f, "|")))
}

// Build will extract extensions, examples, schema and content/media types from node.
func (h *Header) Build(root *yaml.Node, idx *index.SpecIndex) error {
	h.Extensions = low.ExtractExtensions(root)

	// handle example if set.
	_, expLabel, expNode := utils.FindKeyNodeFull(base.ExampleLabel, root.Content)
	if expNode != nil {
		h.Example = low.ExtractExample(expNode, expLabel)
	}

	// handle examples if set.
	exps, expsL, expsN, eErr := low.ExtractMap[*base.Example](base.ExamplesLabel, root, idx)
	if eErr != nil {
		return eErr
	}
	if exps != nil {
		h.Examples = low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*base.Example]]{
			Value:     exps,
			KeyNode:   expsL,
			ValueNode: expsN,
		}
	}

	// handle schema
	sch, sErr := base.ExtractSchema(root, idx)
	if sErr != nil {
		return sErr
	}
	if sch != nil {
		h.Schema = *sch
	}

	// handle content, if set.
	con, cL, cN, cErr := low.ExtractMap[*MediaType](ContentLabel, root, idx)
	if cErr != nil {
		return cErr
	}
	h.Content = low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*MediaType]]{
		Value:     con,
		KeyNode:   cL,
		ValueNode: cN,
	}
	return nil
}
