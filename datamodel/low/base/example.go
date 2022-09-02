// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"strconv"
)

const (
	ExamplesLabel = "examples"
	ExampleLabel  = "example"
	ValueLabel    = "value"
)

type Example struct {
	Summary       low.NodeReference[string]
	Description   low.NodeReference[string]
	Value         low.NodeReference[any]
	ExternalValue low.NodeReference[string]
	Extensions    map[low.KeyReference[string]]low.ValueReference[any]
}

func (ex *Example) FindExtension(ext string) *low.ValueReference[any] {
	return low.FindItemInMap[any](ext, ex.Extensions)
}

func (ex *Example) Build(root *yaml.Node, idx *index.SpecIndex) error {
	ex.Extensions = low.ExtractExtensions(root)
	_, ln, vn := utils.FindKeyNodeFull(ValueLabel, root.Content)

	if vn != nil {
		var n map[string]interface{}
		err := vn.Decode(&n)
		if err != nil {
			// if not a map, then try an array
			var k []interface{}
			err = vn.Decode(&k)
			if err != nil {
				// lets just default to interface
				var j interface{}
				_ = vn.Decode(&j)
				ex.Value = low.NodeReference[any]{
					Value:     j,
					KeyNode:   ln,
					ValueNode: vn,
				}
				return nil
			}
			ex.Value = low.NodeReference[any]{
				Value:     k,
				KeyNode:   ln,
				ValueNode: vn,
			}
			return nil
		}
		ex.Value = low.NodeReference[any]{
			Value:     n,
			KeyNode:   ln,
			ValueNode: vn,
		}
		return nil
	}
	return nil
}

func ExtractExampleValue(exp *yaml.Node) any {
	if utils.IsNodeBoolValue(exp) {
		v, _ := strconv.ParseBool(exp.Value)
		return v
	}
	if utils.IsNodeIntValue(exp) {
		v, _ := strconv.ParseInt(exp.Value, 10, 64)
		return v
	}
	if utils.IsNodeFloatValue(exp) {
		v, _ := strconv.ParseFloat(exp.Value, 64)
		return v
	}
	return exp.Value
}
