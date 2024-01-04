// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

// Encoding represents a low-level OpenAPI 3+ Encoding object
//   - https://spec.openapis.org/oas/v3.1.0#encoding-object
type Encoding struct {
	ContentType   low.NodeReference[string]
	Headers       low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[*Header]]]
	Style         low.NodeReference[string]
	Explode       low.NodeReference[bool]
	AllowReserved low.NodeReference[bool]
	KeyNode       *yaml.Node
	RootNode      *yaml.Node
	*low.Reference
}

// FindHeader attempts to locate a Header with the supplied name
func (en *Encoding) FindHeader(hType string) *low.ValueReference[*Header] {
	return low.FindItemInOrderedMap[*Header](hType, en.Headers.Value)
}

// Hash will return a consistent SHA256 Hash of the Encoding object
func (en *Encoding) Hash() [32]byte {
	var f []string
	if en.ContentType.Value != "" {
		f = append(f, en.ContentType.Value)
	}
	for pair := orderedmap.First(orderedmap.SortAlpha(en.Headers.Value)); pair != nil; pair = pair.Next() {
		f = append(f, fmt.Sprintf("%s-%x", pair.Key().Value, pair.Value().Value.Hash()))
	}
	if en.Style.Value != "" {
		f = append(f, en.Style.Value)
	}
	f = append(f, fmt.Sprint(sha256.Sum256([]byte(fmt.Sprint(en.Explode.Value)))))
	f = append(f, fmt.Sprint(sha256.Sum256([]byte(fmt.Sprint(en.AllowReserved.Value)))))
	return sha256.Sum256([]byte(strings.Join(f, "|")))
}

// Build will extract all Header objects from supplied node.
func (en *Encoding) Build(ctx context.Context, keyNode, root *yaml.Node, idx *index.SpecIndex) error {
	en.KeyNode = keyNode
	root = utils.NodeAlias(root)
	en.RootNode = root
	utils.CheckForMergeNodes(root)
	en.Reference = new(low.Reference)
	headers, hL, hN, err := low.ExtractMap[*Header](ctx, HeadersLabel, root, idx)
	if err != nil {
		return err
	}
	if headers != nil {
		en.Headers = low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[*Header]]]{
			Value:     headers,
			KeyNode:   hL,
			ValueNode: hN,
		}
	}
	return nil
}
