// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"gopkg.in/yaml.v3"
)

// Encoding represents a low-level OpenAPI 3+ Encoding object
//  - https://spec.openapis.org/oas/v3.1.0#encoding-object
type Encoding struct {
	ContentType   low.NodeReference[string]
	Headers       low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Header]]
	Style         low.NodeReference[string]
	Explode       low.NodeReference[bool]
	AllowReserved low.NodeReference[bool]
}

// FindHeader attempts to locate a Header with the supplied name
func (en *Encoding) FindHeader(hType string) *low.ValueReference[*Header] {
	return low.FindItemInMap[*Header](hType, en.Headers.Value)
}

// Build will extract all Header objects from supplied node.
func (en *Encoding) Build(root *yaml.Node, idx *index.SpecIndex) error {
	headers, hL, hN, err := low.ExtractMap[*Header](HeadersLabel, root, idx)
	if err != nil {
		return err
	}
	if headers != nil {
		en.Headers = low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Header]]{
			Value:     headers,
			KeyNode:   hL,
			ValueNode: hN,
		}
	}
	return nil
}
