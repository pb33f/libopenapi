// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"gopkg.in/yaml.v3"
)

type License struct {
	Name low.NodeReference[string]
	URL  low.NodeReference[string]
}

func (l *License) Build(root *yaml.Node, idx *index.SpecIndex) error {
	return nil
}
