// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	low "github.com/pb33f/libopenapi/datamodel/low/v2"
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

	resultChan := make(chan asyncResult[*PathItem])
	var buildPath = func(path string, pi *low.PathItem, rChan chan<- asyncResult[*PathItem]) {
		rChan <- asyncResult[*PathItem]{
			key:    path,
			result: NewPathItem(pi),
		}
	}
	if len(paths.PathItems) > 0 {
		pathItems := make(map[string]*PathItem)
		totalPaths := len(paths.PathItems)
		for k := range paths.PathItems {
			go buildPath(k.Value, paths.PathItems[k].Value, resultChan)
		}
		completedPaths := 0
		for completedPaths < totalPaths {
			select {
			case res := <-resultChan:
				completedPaths++
				pathItems[res.key] = res.result
			}
		}
		p.PathItems = pathItems
	}
	return p
}

func (p *Paths) GoLow() *low.Paths {
	return p.low
}
