// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	low "github.com/pb33f/libopenapi/datamodel/low/3.0"
)

type Paths struct {
	PathItems  map[string]*PathItem
	Extensions map[string]any
	low        *low.Paths
}

func (p *Paths) GoLow() *low.Paths {
	return p.low
}
