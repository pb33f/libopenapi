// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"fmt"
	"gopkg.in/yaml.v3"
)

func (r *Rolodex) FindNodeOrigin(node *yaml.Node) *NodeOrigin {
	//f := make(chan *NodeOrigin)
	//d := make(chan bool)
	//findNode := func(i int, node *yaml.Node) {
	//	n := r.indexes[i].FindNodeOrigin(node)
	//	if n != nil {
	//		f <- n
	//		return
	//	}
	//	d <- true
	//}
	//for i, _ := range r.indexes {
	//	go findNode(i, node)
	//}
	//searched := 0
	//for searched < len(r.indexes) {
	//	select {
	//	case n := <-f:
	//		return n
	//	case <-d:
	//		searched++
	//	}
	//}
	//return nil

	if len(r.indexes) == 0 {
		fmt.Println("NO FUCKING WAY MAN")
	} else {
		//fmt.Printf("searching %d files\n", len(r.indexes))
	}
	for i := range r.indexes {
		n := r.indexes[i].FindNodeOrigin(node)
		if n != nil {
			return n
		}
	}
	//	if n != nil {
	//		f <- n
	//		return
	//	}
	fmt.Println("my FUCKING ARSE")
	return nil

}
