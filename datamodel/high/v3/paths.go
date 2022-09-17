// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	low "github.com/pb33f/libopenapi/datamodel/low/v3"
)

// Paths represents a high-level OpenAPI 3+ Paths object, that is backed by a low-level one.
//
// Holds the relative paths to the individual endpoints and their operations. The path is appended to the URL from the
// Server Object in order to construct the full URL. The Paths MAY be empty, due to Access Control List (ACL)
// constraints.
//  - https://spec.openapis.org/oas/v3.1.0#paths-object
type Paths struct {
	PathItems  map[string]*PathItem
	Extensions map[string]any
	low        *low.Paths
}

// NewPaths creates a new high-level instance of Paths from a low-level one.
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

// GoLow returns the low-level Paths instance used to create the high-level one.
func (p *Paths) GoLow() *low.Paths {
	return p.low
}
