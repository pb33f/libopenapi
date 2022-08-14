// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

type MediaType struct {
	Schema     low.NodeReference[*Schema]
	Example    low.NodeReference[any]
	Examples   low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Example]]
	Encoding   low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Encoding]]
	Extensions map[low.KeyReference[string]]low.ValueReference[any]
}

func (mt *MediaType) FindExtension(ext string) *low.ValueReference[any] {
	return low.FindItemInMap[any](ext, mt.Extensions)
}

func (mt *MediaType) FindPropertyEncoding(eType string) *low.ValueReference[*Encoding] {
	return low.FindItemInMap[*Encoding](eType, mt.Encoding.Value)
}

func (mt *MediaType) FindExample(eType string) *low.ValueReference[*Example] {
	return low.FindItemInMap[*Example](eType, mt.Examples.Value)
}

func (mt *MediaType) GetAllExamples() map[low.KeyReference[string]]low.ValueReference[*Example] {
	return mt.Examples.Value
}

func (mt *MediaType) Build(root *yaml.Node, idx *index.SpecIndex) error {
	mt.Extensions = low.ExtractExtensions(root)

	// handle example if set.
	_, expLabel, expNode := utils.FindKeyNodeFull(ExampleLabel, root.Content)
	if expNode != nil {
		mt.Example = low.NodeReference[any]{Value: expNode.Value, KeyNode: expLabel, ValueNode: expNode}
	}

	// handle schema
	sch, sErr := ExtractSchema(root, idx)
	if sErr != nil {
		return sErr
	}
	if sch != nil {
		mt.Schema = *sch
	}

	// handle examples if set.
	exps, expsL, expsN, eErr := low.ExtractMapFlat[*Example](ExamplesLabel, root, idx)
	if eErr != nil {
		return eErr
	}
	if exps != nil {
		mt.Examples = low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Example]]{
			Value:     exps,
			KeyNode:   expsL,
			ValueNode: expsN,
		}
	}

	// handle encoding
	encs, encsL, encsN, encErr := low.ExtractMapFlat[*Encoding](EncodingLabel, root, idx)
	if encErr != nil {
		return encErr
	}
	if encs != nil {
		mt.Encoding = low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Encoding]]{
			Value:     encs,
			KeyNode:   encsL,
			ValueNode: encsN,
		}
	}
	return nil
}
