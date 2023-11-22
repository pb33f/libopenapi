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
	for i, _ := range r.indexes {
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
	return nil
}
