// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"gopkg.in/yaml.v3"
)

// FindNodeOrigin searches all indexes for the origin of a node. If the node is found, a NodeOrigin
// is returned, otherwise nil is returned.
func (r *Rolodex) FindNodeOrigin(node *yaml.Node) *NodeOrigin {
	f := make(chan *NodeOrigin)
	d := make(chan bool)
	findNode := func(i int, node *yaml.Node) {
		n := r.indexes[i].FindNodeOrigin(node)
		if n != nil {
			f <- n
			return
		}
		d <- true
	}
	for i := range r.indexes {
		go findNode(i, node)
	}
	searched := 0
	for searched < len(r.indexes) {
		select {
		case n := <-f:
			return n
		case <-d:
			searched++
		}
	}
	return r.GetRootIndex().FindNodeOrigin(node)
}

// FindNodeOrigin searches this index for a matching node. If the node is found, a NodeOrigin
// is returned, otherwise nil is returned.
func (index *SpecIndex) FindNodeOrigin(node *yaml.Node) *NodeOrigin {
	if node != nil {
		if index.nodeMap[node.Line] != nil {
			if index.nodeMap[node.Line][node.Column] != nil {
				foundNode := index.nodeMap[node.Line][node.Column]
				if foundNode.Kind == yaml.DocumentNode {
					foundNode = foundNode.Content[0]
				}
				match := true
				if foundNode.Value != node.Value || foundNode.Kind != node.Kind || foundNode.Tag != node.Tag {
					match = false
				}
				if len(foundNode.Content) == len(node.Content) {
					for i := range foundNode.Content {
						if foundNode.Content[i].Value != node.Content[i].Value {
							match = false
						}
					}
				}
				if match {
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
	}
	return nil
}
