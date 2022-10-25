// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"crypto/sha256"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"strings"
)

// Server represents a low-level OpenAPI 3+ Server object.
//  - https://spec.openapis.org/oas/v3.1.0#server-object
type Server struct {
	URL         low.NodeReference[string]
	Description low.NodeReference[string]
	Variables   low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*ServerVariable]]
}

// FindVariable attempts to locate a ServerVariable instance using the supplied key.
func (s *Server) FindVariable(serverVar string) *low.ValueReference[*ServerVariable] {
	return low.FindItemInMap[*ServerVariable](serverVar, s.Variables.Value)
}

// Build will extract server variables from the supplied node.
func (s *Server) Build(root *yaml.Node, idx *index.SpecIndex) error {
	kn, vars := utils.FindKeyNode(VariablesLabel, root.Content)
	if vars == nil {
		return nil
	}
	variablesMap := make(map[low.KeyReference[string]]low.ValueReference[*ServerVariable])
	if utils.IsNodeMap(vars) {
		var currentNode string
		var keyNode *yaml.Node
		for i, varNode := range vars.Content {
			if i%2 == 0 {
				currentNode = varNode.Value
				keyNode = varNode
				continue
			}
			variable := ServerVariable{}
			_ = low.BuildModel(varNode, &variable)
			variablesMap[low.KeyReference[string]{
				Value:   currentNode,
				KeyNode: keyNode,
			}] = low.ValueReference[*ServerVariable]{
				ValueNode: varNode,
				Value:     &variable,
			}
		}
		s.Variables = low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*ServerVariable]]{
			KeyNode:   kn,
			ValueNode: vars,
			Value:     variablesMap,
		}
	}
	return nil
}

// Hash will return a consistent SHA256 Hash of the Server object
func (s *Server) Hash() [32]byte {
	var f []string
	for k := range s.Variables.Value {
		f = append(f, low.GenerateHashString(s.Variables.Value[k].Value))
	}
	if !s.URL.IsEmpty() {
		f = append(f, s.URL.Value)
	}
	if !s.Description.IsEmpty() {
		f = append(f, s.Description.Value)
	}
	return sha256.Sum256([]byte(strings.Join(f, "|")))
}
