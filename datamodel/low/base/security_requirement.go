// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

// SecurityRequirement is a low-level representation of a Swagger / OpenAPI 3 SecurityRequirement object.
//
// SecurityRequirement lists the required security schemes to execute this operation. The object can have multiple
// security schemes declared in it which are all required (that is, there is a logical AND between the schemes).
//
// The name used for each property MUST correspond to a security scheme declared in the Security Definitions
//   - https://swagger.io/specification/v2/#securityDefinitionsObject
//   - https://swagger.io/specification/#security-requirement-object
type SecurityRequirement struct {
	Requirements             low.ValueReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[[]low.ValueReference[string]]]]
	KeyNode                  *yaml.Node
	RootNode                 *yaml.Node
	ContainsEmptyRequirement bool // if a requirement is empty (this means it's optional)
	*low.Reference
}

// Build will extract security requirements from the node (the structure is odd, to be honest)
func (s *SecurityRequirement) Build(_ context.Context, keyNode, root *yaml.Node, _ *index.SpecIndex) error {
	s.KeyNode = keyNode
	root = utils.NodeAlias(root)
	s.RootNode = root
	utils.CheckForMergeNodes(root)
	s.Reference = new(low.Reference)
	var labelNode *yaml.Node
	valueMap := orderedmap.New[low.KeyReference[string], low.ValueReference[[]low.ValueReference[string]]]()
	var arr []low.ValueReference[string]
	for i := range root.Content {
		if i%2 == 0 {
			labelNode = root.Content[i]
			arr = []low.ValueReference[string]{} // reset roles.
			continue
		}
		for j := range root.Content[i].Content {
			if root.Content[i].Content[j].Value == "" {
				s.ContainsEmptyRequirement = true
			}
			arr = append(arr, low.ValueReference[string]{
				Value:     root.Content[i].Content[j].Value,
				ValueNode: root.Content[i].Content[j],
			})
		}
		valueMap.Set(
			low.KeyReference[string]{
				Value:   labelNode.Value,
				KeyNode: labelNode,
			},
			low.ValueReference[[]low.ValueReference[string]]{
				Value:     arr,
				ValueNode: root.Content[i],
			},
		)
	}
	if len(root.Content) == 0 {
		s.ContainsEmptyRequirement = true
	}
	s.Requirements = low.ValueReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[[]low.ValueReference[string]]]]{
		Value:     valueMap,
		ValueNode: root,
	}
	return nil
}

// GetRootNode will return the root yaml node of the SecurityRequirement object
func (s *SecurityRequirement) GetRootNode() *yaml.Node {
	return s.RootNode
}

// GetKeyNode will return the key yaml node of the SecurityRequirement object
func (s *SecurityRequirement) GetKeyNode() *yaml.Node {
	return s.KeyNode
}

// FindRequirement will attempt to locate a security requirement string from a supplied name.
func (s *SecurityRequirement) FindRequirement(name string) []low.ValueReference[string] {
	for pair := orderedmap.First(s.Requirements.Value); pair != nil; pair = pair.Next() {
		if pair.Key().Value == name {
			return pair.Value().Value
		}
	}
	return nil
}

// GetKeys returns a string slice of all the keys used in the requirement.
func (s *SecurityRequirement) GetKeys() []string {
	keys := make([]string, orderedmap.Len(s.Requirements.Value))
	z := 0
	for pair := orderedmap.First(s.Requirements.Value); pair != nil; pair = pair.Next() {
		keys[z] = pair.Key().Value
	}
	return keys
}

// Hash will return a consistent SHA256 Hash of the SecurityRequirement object
func (s *SecurityRequirement) Hash() [32]byte {
	var f []string
	for pair := orderedmap.First(orderedmap.SortAlpha(s.Requirements.Value)); pair != nil; pair = pair.Next() {
		var vals []string
		for y := range pair.Value().Value {
			vals = append(vals, pair.Value().Value[y].Value)
		}
		sort.Strings(vals)

		f = append(f, fmt.Sprintf("%s-%s", pair.Key().Value, strings.Join(vals, "|")))
	}
	return sha256.Sum256([]byte(strings.Join(f, "|")))
}
