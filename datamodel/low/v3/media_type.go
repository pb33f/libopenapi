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

// MediaType represents a low-level OpenAPI MediaType object.
//
// Each Media Type Object provides schema and examples for the media type identified by its key.
//  - https://spec.openapis.org/oas/v3.1.0#media-type-object
type MediaType struct {
	Schema     low.NodeReference[*base.SchemaProxy]
	Example    low.NodeReference[any]
	Examples   low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*base.Example]]
	Encoding   low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Encoding]]
	Extensions map[low.KeyReference[string]]low.ValueReference[any]
}

// FindExtension will attempt to locate an extension with the supplied name.
func (mt *MediaType) FindExtension(ext string) *low.ValueReference[any] {
	return low.FindItemInMap[any](ext, mt.Extensions)
}

// FindPropertyEncoding will attempt to locate an Encoding value with a specific name.
func (mt *MediaType) FindPropertyEncoding(eType string) *low.ValueReference[*Encoding] {
	return low.FindItemInMap[*Encoding](eType, mt.Encoding.Value)
}

// FindExample will attempt to locate an Example with a specific name.
func (mt *MediaType) FindExample(eType string) *low.ValueReference[*base.Example] {
	return low.FindItemInMap[*base.Example](eType, mt.Examples.Value)
}

// GetAllExamples will extract all examples from the MediaType instance.
func (mt *MediaType) GetAllExamples() map[low.KeyReference[string]]low.ValueReference[*base.Example] {
	return mt.Examples.Value
}

// Build will extract examples, extensions, schema and encoding from node.
func (mt *MediaType) Build(root *yaml.Node, idx *index.SpecIndex) error {
	mt.Extensions = low.ExtractExtensions(root)

	// handle example if set.
	_, expLabel, expNode := utils.FindKeyNodeFull(base.ExampleLabel, root.Content)
	if expNode != nil {
		mt.Example = low.NodeReference[any]{Value: expNode.Value, KeyNode: expLabel, ValueNode: expNode}
	}

	//handle schema
	sch, sErr := base.ExtractSchema(root, idx)
	if sErr != nil {
		return sErr
	}
	if sch != nil {
		mt.Schema = *sch
	}

	// handle examples if set.
	exps, expsL, expsN, eErr := low.ExtractMap[*base.Example](base.ExamplesLabel, root, idx)
	if eErr != nil {
		return eErr
	}
	if exps != nil {
		mt.Examples = low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*base.Example]]{
			Value:     exps,
			KeyNode:   expsL,
			ValueNode: expsN,
		}
	}

	// handle encoding
	encs, encsL, encsN, encErr := low.ExtractMap[*Encoding](EncodingLabel, root, idx)
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

// Hash will return a consistent SHA256 Hash of the MediaType object
func (mt *MediaType) Hash() [32]byte {
	var f []string
	if mt.Schema.Value != nil {
		f = append(f, fmt.Sprintf("%x", mt.Schema.Value.Schema().Hash()))
	}
	if mt.Example.Value != nil {
		f = append(f, fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprint(mt.Example.Value)))))
	}
	if len(mt.Examples.Value) > 0 {
		for k := range mt.Examples.Value {
			f = append(f, fmt.Sprintf("%s-%x", k.Value, mt.Examples.Value[k].Value.Hash()))
		}
	}
	if len(mt.Encoding.Value) > 0 {
		for k := range mt.Encoding.Value {
			f = append(f, fmt.Sprintf("%s-%x", k.Value, mt.Encoding.Value[k].Value.Hash()))
		}
	}
	for k := range mt.Extensions {
		f = append(f, fmt.Sprintf("%s-%v", k.Value, mt.Extensions[k].Value))
	}
	return sha256.Sum256([]byte(strings.Join(f, "|")))
}
