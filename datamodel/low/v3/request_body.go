// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"crypto/sha256"
	"fmt"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"gopkg.in/yaml.v3"
	"sort"
	"strings"
)

// RequestBody represents a low-level OpenAPI 3+ RequestBody object.
//  - https://spec.openapis.org/oas/v3.1.0#request-body-object
type RequestBody struct {
	Description low.NodeReference[string]
	Content     low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*MediaType]]
	Required    low.NodeReference[bool]
	Extensions  map[low.KeyReference[string]]low.ValueReference[any]
	*low.Reference
}

// FindExtension attempts to locate an extension using the provided name.
func (rb *RequestBody) FindExtension(ext string) *low.ValueReference[any] {
	return low.FindItemInMap[any](ext, rb.Extensions)
}

// GetExtensions returns all RequestBody extensions and satisfies the low.HasExtensions interface.
func (rb *RequestBody) GetExtensions() map[low.KeyReference[string]]low.ValueReference[any] {
	return rb.Extensions
}

// FindContent attempts to find content/MediaType defined using a specified name.
func (rb *RequestBody) FindContent(cType string) *low.ValueReference[*MediaType] {
	return low.FindItemInMap[*MediaType](cType, rb.Content.Value)
}

// Build will extract extensions and MediaType objects from the node.
func (rb *RequestBody) Build(root *yaml.Node, idx *index.SpecIndex) error {
	rb.Reference = new(low.Reference)
	rb.Extensions = low.ExtractExtensions(root)

	// handle content, if set.
	con, cL, cN, cErr := low.ExtractMap[*MediaType](ContentLabel, root, idx)
	if cErr != nil {
		return cErr
	}
	if con != nil {
		rb.Content = low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*MediaType]]{
			Value:     con,
			KeyNode:   cL,
			ValueNode: cN,
		}
	}
	return nil
}

// Hash will return a consistent SHA256 Hash of the RequestBody object
func (rb *RequestBody) Hash() [32]byte {
	var f []string
	if rb.Description.Value != "" {
		f = append(f, rb.Description.Value)
	}
	if !rb.Required.IsEmpty() {
		f = append(f, fmt.Sprint(rb.Required.Value))
	}
	for k := range rb.Content.Value {
		f = append(f, low.GenerateHashString(rb.Content.Value[k].Value))
	}

	keys := make([]string, len(rb.Extensions))
	z := 0
	for k := range rb.Extensions {
		keys[z] = fmt.Sprintf("%s-%x", k.Value, sha256.Sum256([]byte(fmt.Sprint(rb.Extensions[k].Value))))
		z++
	}
	sort.Strings(keys)
	f = append(f, keys...)

	return sha256.Sum256([]byte(strings.Join(f, "|")))
}
