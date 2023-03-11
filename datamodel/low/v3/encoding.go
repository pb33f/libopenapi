// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"crypto/sha256"
	"fmt"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"gopkg.in/yaml.v3"
	"strings"
)

// Encoding represents a low-level OpenAPI 3+ Encoding object
//  - https://spec.openapis.org/oas/v3.1.0#encoding-object
type Encoding struct {
	ContentType   low.NodeReference[string]
	Headers       low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Header]]
	Style         low.NodeReference[string]
	Explode       low.NodeReference[bool]
	AllowReserved low.NodeReference[bool]
	*low.Reference
}

// FindHeader attempts to locate a Header with the supplied name
func (en *Encoding) FindHeader(hType string) *low.ValueReference[*Header] {
	return low.FindItemInMap[*Header](hType, en.Headers.Value)
}

// Hash will return a consistent SHA256 Hash of the Encoding object
func (en *Encoding) Hash() [32]byte {
	var f []string
	if en.ContentType.Value != "" {
		f = append(f, en.ContentType.Value)
	}
	if len(en.Headers.Value) > 0 {
		l := make([]string, len(en.Headers.Value))
		keys := make(map[string]low.ValueReference[*Header])
		z := 0
		for k := range en.Headers.Value {
			keys[k.Value] = en.Headers.Value[k]
			l[z] = k.Value
			z++
		}

		for k := range en.Headers.Value {
			f = append(f, fmt.Sprintf("%s-%x", k.Value, en.Headers.Value[k].Value.Hash()))
		}
	}
	if en.Style.Value != "" {
		f = append(f, en.Style.Value)
	}
	f = append(f, fmt.Sprint(sha256.Sum256([]byte(fmt.Sprint(en.Explode.Value)))))
	f = append(f, fmt.Sprint(sha256.Sum256([]byte(fmt.Sprint(en.AllowReserved.Value)))))
	return sha256.Sum256([]byte(strings.Join(f, "|")))
}

// Build will extract all Header objects from supplied node.
func (en *Encoding) Build(root *yaml.Node, idx *index.SpecIndex) error {
	en.Reference = new(low.Reference)
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
