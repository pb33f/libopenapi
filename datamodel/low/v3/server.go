// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"crypto/sha256"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

// Server represents a low-level OpenAPI 3+ Server object.
//   - https://spec.openapis.org/oas/v3.1.0#server-object
type Server struct {
	URL         low.NodeReference[string]
	Description low.NodeReference[string]
	Variables   low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[*ServerVariable]]]
	Extensions  *orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]]
	KeyNode     *yaml.Node
	RootNode    *yaml.Node
	*low.Reference
}

// GetRootNode returns the root yaml node of the Server object.
func (s *Server) GetRootNode() *yaml.Node {
	return s.RootNode
}

// GetExtensions returns all Paths extensions and satisfies the low.HasExtensions interface.
func (s *Server) GetExtensions() *orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]] {
	return s.Extensions
}

// FindVariable attempts to locate a ServerVariable instance using the supplied key.
func (s *Server) FindVariable(serverVar string) *low.ValueReference[*ServerVariable] {
	return low.FindItemInOrderedMap[*ServerVariable](serverVar, s.Variables.Value)
}

// Build will extract server variables from the supplied node.
func (s *Server) Build(_ context.Context, keyNode, root *yaml.Node, _ *index.SpecIndex) error {
	s.KeyNode = keyNode
	root = utils.NodeAlias(root)
	s.RootNode = root
	utils.CheckForMergeNodes(root)
	s.Reference = new(low.Reference)
	s.Extensions = low.ExtractExtensions(root)
	kn, vars := utils.FindKeyNode(VariablesLabel, root.Content)
	if vars == nil {
		return nil
	}
	variablesMap := orderedmap.New[low.KeyReference[string], low.ValueReference[*ServerVariable]]()
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
			variable.Reference = new(low.Reference)
			_ = low.BuildModel(varNode, &variable)
			variablesMap.Set(
				low.KeyReference[string]{
					Value:   currentNode,
					KeyNode: keyNode,
				},
				low.ValueReference[*ServerVariable]{
					ValueNode: varNode,
					Value:     &variable,
				},
			)
		}
		s.Variables = low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[*ServerVariable]]]{
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
	for pair := orderedmap.First(orderedmap.SortAlpha(s.Variables.Value)); pair != nil; pair = pair.Next() {
		f = append(f, low.GenerateHashString(pair.Value().Value))
	}
	if !s.URL.IsEmpty() {
		f = append(f, s.URL.Value)
	}
	if !s.Description.IsEmpty() {
		f = append(f, s.Description.Value)
	}
	f = append(f, low.HashExtensions(s.Extensions)...)
	return sha256.Sum256([]byte(strings.Join(f, "|")))
}
