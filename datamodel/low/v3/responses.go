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

// Responses represents a low-level OpenAPI 3+ Responses object.
//
// It's a container for the expected responses of an operation. The container maps an HTTP response code to the
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
//   - https://spec.openapis.org/oas/v3.1.0#responses-object
//
// This structure is identical to the v2 version, however they use different response types, hence
// the duplication. Perhaps in the future we could use generics here, but for now to keep things
// simple, they are broken out into individual versions.
type Responses struct {
	Codes      *orderedmap.Map[low.KeyReference[string], low.ValueReference[*Response]]
	Default    low.NodeReference[*Response]
	Extensions *orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]]
	KeyNode    *yaml.Node
	RootNode   *yaml.Node
	*low.Reference
}

// GetExtensions returns all Responses extensions and satisfies the low.HasExtensions interface.
func (r *Responses) GetExtensions() *orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]] {
	return r.Extensions
}

// Build will extract default response and all Response objects for each code
func (r *Responses) Build(ctx context.Context, keyNode, root *yaml.Node, idx *index.SpecIndex) error {
	r.KeyNode = keyNode
	root = utils.NodeAlias(root)
	r.RootNode = root
	r.Reference = new(low.Reference)
	r.Extensions = low.ExtractExtensions(root)
	utils.CheckForMergeNodes(root)
	if utils.IsNodeMap(root) {
		codes, err := low.ExtractMapNoLookup[*Response](ctx, root, idx)
		if err != nil {
			return err
		}
		if codes != nil {
			r.Codes = codes
		}

		def := r.getDefault()
		if def != nil {
			// default is bundled into codes, pull it out
			r.Default = *def
			// remove default from codes
			r.deleteCode(DefaultLabel)
		}
	} else {
		return fmt.Errorf("responses build failed: vn node is not a map! line %d, col %d",
			root.Line, root.Column)
	}
	return nil
}

func (r *Responses) getDefault() *low.NodeReference[*Response] {
	for pair := orderedmap.First(r.Codes); pair != nil; pair = pair.Next() {
		if strings.ToLower(pair.Key().Value) == DefaultLabel {
			return &low.NodeReference[*Response]{
				ValueNode: pair.Value().ValueNode,
				KeyNode:   pair.Key().KeyNode,
				Value:     pair.Value().Value,
			}
		}
	}
	return nil
}

// used to remove default from codes extracted by Build()
func (r *Responses) deleteCode(code string) {
	var key *low.KeyReference[string]
	for pair := orderedmap.First(r.Codes); pair != nil; pair = pair.Next() {
		if pair.Key().Value == code {
			key = pair.KeyPtr()
			break
		}
	}
	// should never be nil, but, you never know... science and all that!
	if key != nil {
		r.Codes.Delete(*key)
	}
}

// FindResponseByCode will attempt to locate a Response using an HTTP response code.
func (r *Responses) FindResponseByCode(code string) *low.ValueReference[*Response] {
	return low.FindItemInOrderedMap[*Response](code, r.Codes)
}

// Hash will return a consistent SHA256 Hash of the Examples object
func (r *Responses) Hash() [32]byte {
	var f []string
	for pair := orderedmap.First(orderedmap.SortAlpha(r.Codes)); pair != nil; pair = pair.Next() {
		f = append(f, fmt.Sprintf("%s-%s", pair.Key().Value, low.GenerateHashString(pair.Value().Value)))
	}
	if !r.Default.IsEmpty() {
		f = append(f, low.GenerateHashString(r.Default.Value))
	}
	f = append(f, low.HashExtensions(r.Extensions)...)
	return sha256.Sum256([]byte(strings.Join(f, "|")))
}
