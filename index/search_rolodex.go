// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

type originCheck struct {
	valueOrigin *NodeOrigin
	valueHash   string
	keyOrigin   *NodeOrigin
	rolo        *Rolodex
	value       *yaml.Node
	skipIndex   *SpecIndex
	ref         string
	refNode     *yaml.Node
}

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
				rolo:        r,
				ref:         refValue,
				refNode:     refNode,
			})
			if done {
				return origin
			}
		}

		// not found in root, search all indexes
		for i := range r.indexes {
			idx := r.indexes[i]
			if keyOrigin == nil {
				keyOrigin = idx.FindNodeOrigin(key)
			}
			if keyOrigin != nil {
				valueOrigin = idx.FindNodeOrigin(value)
				valueHash = HashNode(value)
				if valueOrigin != nil && valueOrigin.Node != nil {
					// hash value and value origin
					valueOriginHash := HashNode(valueOrigin.Node)
					if valueHash == valueOriginHash {
						if keyOrigin.AbsoluteLocation != keyOrigin.AbsoluteLocation {
							if refNode == nil && refValue == "" {
								keyOrigin.AbsoluteLocation = valueOrigin.AbsoluteLocation
							}
						}
						return keyOrigin
					}
				} else {
					origin, done := checkOrigin(originCheck{
						valueHash: valueHash,
						keyOrigin: keyOrigin,
						rolo:      r,
						skipIndex: idx,
						value:     value,
						ref:       refValue,
						refNode:   refNode,
					})
					if done {
						return origin
					}

					// cannot be found!
				}
			}
		}
	}
	return keyOrigin
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
		for i := range check.rolo.indexes {
			idx := check.rolo.indexes[i]
			if idx == check.skipIndex {
				continue
			}
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

func (index *SpecIndex) FindNodeOriginWithValue(node, value *yaml.Node) *NodeOrigin {
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
				if !match && (utils.IsNodeMap(foundNode) || utils.IsNodeArray(foundNode)) && (utils.IsNodeMap(node) || utils.IsNodeArray(node)) {
					if len(node.Content) == len(foundNode.Content) {
						// hash node and found node
						nodeHash := HashNode(node)
						foundNodeHash := HashNode(foundNode)
						if nodeHash == foundNodeHash {
							match = true
						}
					}
				} else {
					if utils.IsNodeMap(foundNode) || utils.IsNodeArray(foundNode) {
						for _, n := range foundNode.Content {
							if n.Line == node.Line && n.Column == node.Column {
								if n.Value == node.Value {
									match = true
									foundNode = n
									break
								}
							}
						}
					}

					if !match && (foundNode.Value == node.Value) && (foundNode.Kind == node.Kind || foundNode.Tag != node.Tag) {
						match = true
					}
					if !match && len(foundNode.Content) == len(node.Content) {
						for i := range foundNode.Content {
							if foundNode.Content[i].Value == node.Content[i].Value {
								match = true
							}
						}
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
				if !match && (utils.IsNodeMap(foundNode) || utils.IsNodeArray(foundNode)) && (utils.IsNodeMap(node) || utils.IsNodeArray(node)) {
					if len(node.Content) == len(foundNode.Content) {

						// hash node and found node
						nodeHash := HashNode(node)
						foundNodeHash := HashNode(foundNode)
						if nodeHash == foundNodeHash {
							match = true
						}
					}
				} else {
					// hash node and found node
					nodeHash := HashNode(node)
					foundNodeHash := HashNode(foundNode)
					if nodeHash == foundNodeHash {
						match = true
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
