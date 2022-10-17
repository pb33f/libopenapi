// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"crypto/sha256"
	"fmt"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"strings"
)

// Items is a low-level representation of a Swagger / OpenAPI 2 Items object.
//
// Items is a limited subset of JSON-Schema's items object. It is used by parameter definitions that are not
// located in "body"
//  - https://swagger.io/specification/v2/#itemsObject
type Items struct {
	Type             low.NodeReference[string]
	Format           low.NodeReference[string]
	CollectionFormat low.NodeReference[string]
	Items            low.NodeReference[*Items]
	Default          low.NodeReference[any]
	Maximum          low.NodeReference[int]
	ExclusiveMaximum low.NodeReference[bool]
	Minimum          low.NodeReference[int]
	ExclusiveMinimum low.NodeReference[bool]
	MaxLength        low.NodeReference[int]
	MinLength        low.NodeReference[int]
	Pattern          low.NodeReference[string]
	MaxItems         low.NodeReference[int]
	MinItems         low.NodeReference[int]
	UniqueItems      low.NodeReference[bool]
	Enum             low.NodeReference[[]low.ValueReference[string]]
	MultipleOf       low.NodeReference[int]
}

// Hash will return a consistent SHA256 Hash of the Items object
func (i *Items) Hash() [32]byte {
	var f []string
	if i.Type.Value != "" {
		f = append(f, i.Type.Value)
	}
	if i.Format.Value != "" {
		f = append(f, i.Format.Value)
	}
	if i.CollectionFormat.Value != "" {
		f = append(f, i.CollectionFormat.Value)
	}
	if i.Default.Value != "" {
		f = append(f, fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprint(i.Default.Value)))))
	}
	f = append(f, fmt.Sprint(i.Maximum.Value))
	f = append(f, fmt.Sprint(i.Minimum.Value))
	f = append(f, fmt.Sprint(i.ExclusiveMinimum.Value))
	f = append(f, fmt.Sprint(i.ExclusiveMaximum.Value))
	f = append(f, fmt.Sprint(i.MinLength.Value))
	f = append(f, fmt.Sprint(i.MaxLength.Value))
	f = append(f, fmt.Sprint(i.MinItems.Value))
	f = append(f, fmt.Sprint(i.MaxItems.Value))
	f = append(f, fmt.Sprint(i.MultipleOf.Value))
	f = append(f, fmt.Sprint(i.UniqueItems.Value))
	if i.Pattern.Value != "" {
		f = append(f, fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprint(i.Pattern.Value)))))
	}
	if len(i.Enum.Value) > 0 {
		for k := range i.Enum.Value {
			f = append(f, fmt.Sprint(i.Enum.Value[k].Value))
		}
	}
	if i.Items.Value != nil {
		f = append(f, fmt.Sprintf("%x", i.Items.Value.Hash()))
	}
	return sha256.Sum256([]byte(strings.Join(f, "|")))
}

// Build will build out items and default value.
func (i *Items) Build(root *yaml.Node, idx *index.SpecIndex) error {
	items, iErr := low.ExtractObject[*Items](ItemsLabel, root, idx)
	if iErr != nil {
		return iErr
	}
	i.Items = items

	_, ln, vn := utils.FindKeyNodeFull(DefaultLabel, root.Content)
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
				i.Default = low.NodeReference[any]{
					Value:     j,
					KeyNode:   ln,
					ValueNode: vn,
				}
				return nil
			}
			i.Default = low.NodeReference[any]{
				Value:     k,
				KeyNode:   ln,
				ValueNode: vn,
			}
			return nil
		}
		i.Default = low.NodeReference[any]{
			Value:     n,
			KeyNode:   ln,
			ValueNode: vn,
		}
		return nil
	}
	return nil
}
