// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"crypto/sha256"
	"fmt"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"sort"
	"strconv"
	"strings"
)

// Example represents a low-level Example object as defined by OpenAPI 3+
//
//	v3 - https://spec.openapis.org/oas/v3.1.0#example-object
type Example struct {
	Summary       low.NodeReference[string]
	Description   low.NodeReference[string]
	Value         low.NodeReference[any]
	ExternalValue low.NodeReference[string]
	Extensions    map[low.KeyReference[string]]low.ValueReference[any]
	*low.Reference
}

// FindExtension returns a ValueReference containing the extension value, if found.
func (ex *Example) FindExtension(ext string) *low.ValueReference[any] {
	return low.FindItemInMap[any](ext, ex.Extensions)
}

// Hash will return a consistent SHA256 Hash of the Discriminator object
func (ex *Example) Hash() [32]byte {
	var f []string
	if ex.Summary.Value != "" {
		f = append(f, ex.Summary.Value)
	}
	if ex.Description.Value != "" {
		f = append(f, ex.Description.Value)
	}
	if ex.Value.Value != "" {
		// this could be anything!
		f = append(f, fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprint(ex.Value.Value)))))
	}
	if ex.ExternalValue.Value != "" {
		f = append(f, ex.ExternalValue.Value)
	}
	keys := make([]string, len(ex.Extensions))
	z := 0
	for k := range ex.Extensions {
		keys[z] = fmt.Sprintf("%s-%x", k.Value, sha256.Sum256([]byte(fmt.Sprint(ex.Extensions[k].Value))))
		z++
	}
	sort.Strings(keys)
	f = append(f, keys...)
	return sha256.Sum256([]byte(strings.Join(f, "|")))
}

// Build extracts extensions and example value
func (ex *Example) Build(_, root *yaml.Node, idx *index.SpecIndex) error {
	root = utils.NodeAlias(root)
	utils.CheckForMergeNodes(root)
	ex.Reference = new(low.Reference)
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

// GetExtensions will return Example extensions to satisfy the HasExtensions interface.
func (ex *Example) GetExtensions() map[low.KeyReference[string]]low.ValueReference[any] {
	return ex.Extensions
}

// ExtractExampleValue will extract a primitive example value (if possible), or just the raw Value property if not.
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
	if utils.IsNodeMap(exp) {
		var m map[string]interface{}
		_ = exp.Decode(&m)
		return m
	}
	if utils.IsNodeArray(exp) {
		var m []interface{}
		_ = exp.Decode(&m)
		return m
	}
	return exp.Value
}
