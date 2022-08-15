// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

const (
	HeadersLabel = "headers"
)

type Header struct {
	Description     low.NodeReference[string]
	Required        low.NodeReference[bool]
	Deprecated      low.NodeReference[bool]
	AllowEmptyValue low.NodeReference[bool]
	Style           low.NodeReference[string]
	Explode         low.NodeReference[bool]
	AllowReserved   low.NodeReference[bool]
	Schema          low.NodeReference[*Schema]
	Example         low.NodeReference[any]
	Examples        map[low.KeyReference[string]]map[low.KeyReference[string]]low.ValueReference[*Example]
	Content         map[low.KeyReference[string]]map[low.KeyReference[string]]low.ValueReference[*MediaType]
	Extensions      map[low.KeyReference[string]]low.ValueReference[any]
}

func (h *Header) FindExtension(ext string) *low.ValueReference[any] {
	return low.FindItemInMap[any](ext, h.Extensions)
}

func (h *Header) FindExample(eType string) *low.ValueReference[*Example] {
	// there is only one item in here by design, so this can only ever loop once
	var k *low.ValueReference[*Example]
	for _, v := range h.Examples {
		k = low.FindItemInMap[*Example](eType, v)
	}
	return k
}

func (h *Header) FindContent(ext string) *low.ValueReference[*MediaType] {
	// there is only one item in here by design, so this can only ever loop once
	var k *low.ValueReference[*MediaType]
	for _, v := range h.Content {
		k = low.FindItemInMap[*MediaType](ext, v)
	}
	return k
}

func (h *Header) Build(root *yaml.Node, idx *index.SpecIndex) error {
	h.Extensions = low.ExtractExtensions(root)

	// handle example if set.
	_, expLabel, expNode := utils.FindKeyNodeFull(ExampleLabel, root.Content)
	if expNode != nil {
		h.Example = low.ExtractExample(expNode, expLabel)
	}

	// handle examples if set.
	exps, eErr := low.ExtractMap[*Example](ExamplesLabel, root, idx)
	if eErr != nil {
		return eErr
	}
	if exps != nil {
		h.Examples = exps
	}

	// handle schema
	sch, sErr := ExtractSchema(root, idx)
	if sErr != nil {
		return sErr
	}
	if sch != nil {
		h.Schema = *sch
	}

	// handle content, if set.
	con, cErr := low.ExtractMap[*MediaType](ContentLabel, root, idx)
	if cErr != nil {
		return cErr
	}
	h.Content = con

	return nil
}
