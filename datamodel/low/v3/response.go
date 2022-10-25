// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"crypto/sha256"
	"fmt"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"strings"
)

// Responses represents a low-level OpenAPI 3+ Responses object.
//
// It's a container for the expected responses of an operation. The container maps a HTTP response code to the
// expected response.
//
// The specification is not necessarily expected to cover all possible HTTP response codes because they may not be
// known in advance. However, documentation is expected to cover a successful operation response and any known errors.
//
// The default MAY be used as a default response object for all HTTP codes that are not covered individually by
// the Responses Object.
//
// The Responses Object MUST contain at least one response code, and if only one response code is provided it SHOULD
// be the response for a successful operation call.
//  - https://spec.openapis.org/oas/v3.1.0#responses-object
type Responses struct {
	Codes   map[low.KeyReference[string]]low.ValueReference[*Response]
	Default low.NodeReference[*Response]
}

// Build will extract default response and all Response objects for each code
func (r *Responses) Build(root *yaml.Node, idx *index.SpecIndex) error {
	if utils.IsNodeMap(root) {
		codes, err := low.ExtractMapNoLookup[*Response](root, idx)
		if err != nil {
			return err
		}
		if codes != nil {
			r.Codes = codes
		}

		def, derr := low.ExtractObject[*Response](DefaultLabel, root, idx)
		if derr != nil {
			return derr
		}
		if def.Value != nil {
			r.Default = def
		}
	} else {
		return fmt.Errorf("responses build failed: vn node is not a map! line %d, col %d",
			root.Line, root.Column)
	}
	return nil
}

// FindResponseByCode will attempt to locate a Response using an HTTP response code.
func (r *Responses) FindResponseByCode(code string) *low.ValueReference[*Response] {
	return low.FindItemInMap[*Response](code, r.Codes)
}

// Response represents a high-level OpenAPI 3+ Response object that is backed by a low-level one.
//
// Describes a single response from an API Operation, including design-time, static links to
// operations based on the response.
//  - https://spec.openapis.org/oas/v3.1.0#response-object
type Response struct {
	Description low.NodeReference[string]
	Headers     low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Header]]
	Content     low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*MediaType]]
	Extensions  map[low.KeyReference[string]]low.ValueReference[any]
	Links       low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Link]]
}

// FindExtension will attempt to locate an extension using the supplied key
func (r *Response) FindExtension(ext string) *low.ValueReference[any] {
	return low.FindItemInMap[any](ext, r.Extensions)
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
	r.Extensions = low.ExtractExtensions(root)

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
	for k := range r.Headers.Value {
		f = append(f, low.GenerateHashString(r.Headers.Value[k]))
	}
	for k := range r.Content.Value {
		f = append(f, low.GenerateHashString(r.Content.Value[k]))
	}
	for k := range r.Links.Value {
		f = append(f, low.GenerateHashString(r.Links.Value[k]))
	}
	for k := range r.Extensions {
		f = append(f, fmt.Sprintf("%s-%x", k.Value,
			sha256.Sum256([]byte(fmt.Sprint(r.Extensions[k].Value)))))
	}
	return sha256.Sum256([]byte(strings.Join(f, "|")))
}
