// Copyright 2023-2024 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

// FindNodeOriginWithValue searches all indexes for the origin of a node with a specific value. If the node is found, a NodeOrigin
// is returned, otherwise nil is returned. The key and the value have to be provided. If the refNode and refValue are provided, the
// returned value will be the key origin, not the value origin.
func (r *Rolodex) FindNodeOriginWithValue(key, value, refNode *yaml.Node, refValue string) *NodeOrigin {
	if key == nil {
		return nil
	}
	keyOrigin := r.FindNodeOrigin(key)
	var valueOrigin *NodeOrigin
	var valueHash string
	if value != nil {
		if keyOrigin != nil && keyOrigin.AbsoluteLocation == r.GetRootIndex().specAbsolutePath {
			valueOrigin = r.GetRootIndex().FindNodeOrigin(value)
			valueHash = HashNode(value)
			if refNode != nil && refValue != "" {
				return keyOrigin
			}
			origin, done := checkOrigin(originCheck{
				valueOrigin: valueOrigin,
				valueHash:   valueHash,
				keyOrigin:   keyOrigin,
				value:       value,
				rolodex:     r,
				ref:         refValue,
				refNode:     refNode,
			})
			if done {
				return origin
			} else {
				return nil
			}
		}
	}
	return keyOrigin
}

// FindNodeOrigin searches all indexes for the origin of a node. If the node is found, a NodeOrigin
// is returned, otherwise nil is returned.
func (r *Rolodex) FindNodeOrigin(node *yaml.Node) *NodeOrigin {
	if node == nil {
		return nil
	}
	found := r.GetRootIndex().FindNodeOrigin(node)
	if found != nil {
		return found
	}
	for i := range r.indexes {
		idx := r.indexes[i]
		n := idx.FindNodeOrigin(node)
		if n != nil {
			return n
		}
	}
	return nil
}

// FindNodeOrigin searches this index for a matching node. If the node is found, a NodeOrigin
// is returned, otherwise nil is returned.
func (index *SpecIndex) FindNodeOrigin(node *yaml.Node) *NodeOrigin {
	if node != nil {
		index.nodeMapLock.RLock()
		if index.nodeMap[node.Line] != nil {
			if index.nodeMap[node.Line][node.Column] != nil {
				foundNode := index.nodeMap[node.Line][node.Column]
				if foundNode.Kind == yaml.DocumentNode {
					foundNode = foundNode.Content[0]
				}
				match := false

				if foundNode == node {
					match = true
				}

				// if the found node is a map. iterate through the content until we locate the node at that position
				if !match && (utils.IsNodeMap(foundNode) ||
					utils.IsNodeArray(foundNode)) && (utils.IsNodeMap(node) || utils.IsNodeArray(node)) {
					if len(node.Content) == len(foundNode.Content) {
						// hash node and found node
						match = checkHash(node, foundNode)
					}
				} else {
					if !match {
						// hash node and found node
						match = checkHash(node, foundNode)
					}
				}

				if match {
					index.nodeMapLock.RUnlock()
					return &NodeOrigin{
						Node:             foundNode,
						Line:             node.Line,
						Column:           node.Column,
						AbsoluteLocation: index.specAbsolutePath,
						Index:            index,
					}
				}
			}
		}
		index.nodeMapLock.RUnlock()
	}
	return nil
}

type originCheck struct {
	valueOrigin *NodeOrigin
	valueHash   string
	keyOrigin   *NodeOrigin
	rolodex     *Rolodex
	value       *yaml.Node
	ref         string
	refNode     *yaml.Node
}

func checkHash(node, foundNode *yaml.Node) bool {
	nodeHash := HashNode(node)
	foundNodeHash := HashNode(foundNode)
	if nodeHash == foundNodeHash {
		return true
	}
	return false
}

func checkOrigin(check originCheck) (*NodeOrigin, bool) {
	if check.valueOrigin != nil {
		// hash value and value origin
		valueOriginHash := HashNode(check.valueOrigin.Node)
		if check.valueHash == valueOriginHash {
			return check.keyOrigin, true
		}
	} else {
		// no hit on the root, but we know the value is in the spec, so we need to search all indexes
		for i := range check.rolodex.indexes {
			idx := check.rolodex.indexes[i]
			n := idx.FindNodeOrigin(check.value)
			if n != nil && n.Node != nil {
				// do the hashes match?
				valueOriginHash := HashNode(n.Node)
				if check.valueHash == valueOriginHash {
					if check.keyOrigin.AbsoluteLocation != n.AbsoluteLocation {
						if check.refNode == nil && check.ref == "" {
							check.keyOrigin.AbsoluteLocation = n.AbsoluteLocation
						}
					}
					return check.keyOrigin, true
				}
			}
		}
	}
	return nil, false
}
