// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"crypto/sha256"
	"fmt"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"sort"
	"strings"
)

// Response is a representation of a high-level Swagger / OpenAPI 2 Response object, backed by a low-level one.
//
// Response describes a single response from an API Operation
//   - https://swagger.io/specification/v2/#responseObject
type Response struct {
	Description low.NodeReference[string]
	Schema      low.NodeReference[*base.SchemaProxy]
	Headers     low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Header]]
	Examples    low.NodeReference[*Examples]
	Extensions  map[low.KeyReference[string]]low.ValueReference[any]
}

// FindExtension will attempt to locate an extension value given a key to lookup.
func (r *Response) FindExtension(ext string) *low.ValueReference[any] {
	return low.FindItemInMap[any](ext, r.Extensions)
}

// GetExtensions returns all Response extensions and satisfies the low.HasExtensions interface.
func (r *Response) GetExtensions() map[low.KeyReference[string]]low.ValueReference[any] {
	return r.Extensions
}

// FindHeader will attempt to locate a Header value, given a key
func (r *Response) FindHeader(hType string) *low.ValueReference[*Header] {
	return low.FindItemInMap[*Header](hType, r.Headers.Value)
}

// Build will extract schema, extensions, examples and headers from node
func (r *Response) Build(_, root *yaml.Node, idx *index.SpecIndex) error {
	root = utils.NodeAlias(root)
	utils.CheckForMergeNodes(root)
	r.Extensions = low.ExtractExtensions(root)
	s, err := base.ExtractSchema(root, idx)
	if err != nil {
		return err
	}
	if s != nil {
		r.Schema = *s
	}

	// extract examples
	examples, expErr := low.ExtractObject[*Examples](ExamplesLabel, root, idx)
	if expErr != nil {
		return expErr
	}
	r.Examples = examples

	//extract headers
	headers, lN, kN, err := low.ExtractMap[*Header](HeadersLabel, root, idx)
	if err != nil {
		return err
	}
	if headers != nil {
		r.Headers = low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Header]]{
			Value:     headers,
			KeyNode:   lN,
			ValueNode: kN,
		}
	}
	return nil
}

// Hash will return a consistent SHA256 Hash of the Response object
func (r *Response) Hash() [32]byte {
	var f []string
	if r.Description.Value != "" {
		f = append(f, r.Description.Value)
	}
	if !r.Schema.IsEmpty() {
		f = append(f, low.GenerateHashString(r.Schema.Value))
	}
	if !r.Examples.IsEmpty() {
		for k := range r.Examples.Value.Values {
			f = append(f, low.GenerateHashString(r.Examples.Value.Values[k].Value))
		}
	}
	keys := make([]string, len(r.Extensions))
	z := 0
	for k := range r.Extensions {
		keys[z] = fmt.Sprintf("%s-%x", k.Value, sha256.Sum256([]byte(fmt.Sprint(r.Extensions[k].Value))))
		z++
	}
	sort.Strings(keys)
	f = append(f, keys...)
	return sha256.Sum256([]byte(strings.Join(f, "|")))
}
