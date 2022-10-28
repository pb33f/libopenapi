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

// Responses is a low-level representation of a Swagger / OpenAPI 2 Responses object.
type Responses struct {
	Codes      map[low.KeyReference[string]]low.ValueReference[*Response]
	Default    low.NodeReference[*Response]
	Extensions map[low.KeyReference[string]]low.ValueReference[any]
}

// Build will extract default value and extensions from node.
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
		if re := r.FindResponseByCode(DefaultLabel); re != nil {
			r.Default = low.NodeReference[*Response]{
				Value:     re.Value,
				ValueNode: re.ValueNode,
				KeyNode:   re.ValueNode,
			}
			r.deleteCode(DefaultLabel)
		}
	} else {
		return fmt.Errorf("responses build failed: vn node is not a map! line %d, col %d", root.Line, root.Column)
	}
	return nil
}

func (r *Responses) deleteCode(code string) {
	var key *low.KeyReference[string]
	if r.Codes != nil {
		for k := range r.Codes {
			if k.Value == code {
				key = &k
			}
		}
	}
	if key != nil {
		delete(r.Codes, *key)
	}
}

// FindResponseByCode will attempt to locate a Response instance using an HTTP response code string.
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
