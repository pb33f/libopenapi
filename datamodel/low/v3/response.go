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

// Response represents a high-level OpenAPI 3+ Response object that is backed by a low-level one.
//
// Describes a single response from an API Operation, including design-time, static links to
// operations based on the response.
//   - https://spec.openapis.org/oas/v3.1.0#response-object
type Response struct {
	Description low.NodeReference[string]
	Headers     low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Header]]
	Content     low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*MediaType]]
	Extensions  map[low.KeyReference[string]]low.ValueReference[any]
	Links       low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Link]]
	*low.Reference
}

// FindExtension will attempt to locate an extension using the supplied key
func (r *Response) FindExtension(ext string) *low.ValueReference[any] {
	return low.FindItemInMap[any](ext, r.Extensions)
}

// GetExtensions returns all OAuthFlow extensions and satisfies the low.HasExtensions interface.
func (r *Response) GetExtensions() map[low.KeyReference[string]]low.ValueReference[any] {
	return r.Extensions
}

// FindContent will attempt to locate a MediaType instance using the supplied key.
func (r *Response) FindContent(cType string) *low.ValueReference[*MediaType] {
	return low.FindItemInMap[*MediaType](cType, r.Content.Value)
}

// FindHeader will attempt to locate a Header instance using the supplied key.
func (r *Response) FindHeader(hType string) *low.ValueReference[*Header] {
	return low.FindItemInMap[*Header](hType, r.Headers.Value)
}

// FindLink will attempt to locate a Link instance using the supplied key.
func (r *Response) FindLink(hType string) *low.ValueReference[*Link] {
	return low.FindItemInMap[*Link](hType, r.Links.Value)
}

// Build will extract headers, extensions, content and links from node.
func (r *Response) Build(root *yaml.Node, idx *index.SpecIndex) error {
	r.Reference = new(low.Reference)
	r.Extensions = low.ExtractExtensions(root)

	//extract headers
	headers, lN, kN, err := low.ExtractMapExtensions[*Header](HeadersLabel, root, idx, true)
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

	con, clN, cN, cErr := low.ExtractMap[*MediaType](ContentLabel, root, idx)
	if cErr != nil {
		return cErr
	}
	if con != nil {
		r.Content = low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*MediaType]]{
			Value:     con,
			KeyNode:   clN,
			ValueNode: cN,
		}
	}

	// handle links if set
	links, linkLabel, linkValue, lErr := low.ExtractMap[*Link](LinksLabel, root, idx)
	if lErr != nil {
		return lErr
	}
	if links != nil {
		r.Links = low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Link]]{
			Value:     links,
			KeyNode:   linkLabel,
			ValueNode: linkValue,
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
	keys := make([]string, len(r.Headers.Value))
	z := 0
	for k := range r.Headers.Value {
		keys[z] = fmt.Sprintf("%s-%s", k.Value, low.GenerateHashString(r.Headers.Value[k].Value))
		z++
	}
	sort.Strings(keys)
	f = append(f, keys...)
	keys = make([]string, len(r.Content.Value))
	z = 0
	for k := range r.Content.Value {
		keys[z] = fmt.Sprintf("%s-%s", k.Value, low.GenerateHashString(r.Content.Value[k].Value))
		z++
	}
	sort.Strings(keys)
	f = append(f, keys...)
	keys = make([]string, len(r.Links.Value))
	z = 0
	for k := range r.Links.Value {
		keys[z] = fmt.Sprintf("%s-%s", k.Value, low.GenerateHashString(r.Links.Value[k].Value))
		z++
	}
	sort.Strings(keys)
	f = append(f, keys...)
	keys = make([]string, len(r.Extensions))
	z = 0
	for k := range r.Extensions {
		keys[z] = fmt.Sprintf("%s-%x", k.Value, sha256.Sum256([]byte(fmt.Sprint(r.Extensions[k].Value))))
		z++
	}
	sort.Strings(keys)
	f = append(f, keys...)
	return sha256.Sum256([]byte(strings.Join(f, "|")))
}
