// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	low "github.com/pb33f/libopenapi/datamodel/low/3.0"
)

type Paths struct {
	PathItems  map[string]*PathItem
	Extensions map[string]any
	low        *low.Paths
}

func NewPaths(paths *low.Paths) *Paths {
	p := new(Paths)
	p.low = paths
	p.Extensions = high.ExtractExtensions(paths.Extensions)
	items := make(map[string]*PathItem)

	// build paths async for speed.
	type pRes struct {
		k string
		v *PathItem
	}
	var buildPathItem = func(key string, item *low.PathItem, c chan<- pRes) {
		c <- pRes{key, NewPathItem(item)}
	}
	rChan := make(chan pRes)
	for k := range paths.PathItems {
		go buildPathItem(k.Value, paths.PathItems[k].Value, rChan)
	}
	pathsBuilt := 0
	for pathsBuilt < len(paths.PathItems) {
		select {
		case r := <-rChan:
			pathsBuilt++
			items[r.k] = r.v
		}
	}
	p.PathItems = items
	return p
}

func (p *Paths) GoLow() *low.Paths {
	return p.low
}
