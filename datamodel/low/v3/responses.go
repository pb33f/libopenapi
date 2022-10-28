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
//
// This structure is identical to the v2 version, however they use different response types, hence
// the duplication. Perhaps in the future we could use generics here, but for now to keep things
// simple, they are broken out into individual versions.
type Responses struct {
	Codes      map[low.KeyReference[string]]low.ValueReference[*Response]
	Default    low.NodeReference[*Response]
	Extensions map[low.KeyReference[string]]low.ValueReference[any]
}

// Build will extract default response and all Response objects for each code
func (r *Responses) Build(root *yaml.Node, idx *index.SpecIndex) error {
	r.Extensions = low.ExtractExtensions(root)
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

// Hash will return a consistent SHA256 Hash of the Examples object
func (r *Responses) Hash() [32]byte {
	var f []string
	for k := range r.Codes {
		f = append(f, low.GenerateHashString(r.Codes[k].Value))
	}
	if !r.Default.IsEmpty() {
		f = append(f, low.GenerateHashString(r.Default.Value))
	}
	for k := range r.Extensions {
		f = append(f, fmt.Sprintf("%s-%x", k.Value,
			sha256.Sum256([]byte(fmt.Sprint(r.Extensions[k].Value)))))
	}
	return sha256.Sum256([]byte(strings.Join(f, "|")))
}
