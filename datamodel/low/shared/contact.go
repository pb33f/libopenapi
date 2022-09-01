// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package shared

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"gopkg.in/yaml.v3"
)

type Contact struct {
	Name  low.NodeReference[string]
	URL   low.NodeReference[string]
	Email low.NodeReference[string]
}

func (c *Contact) Build(root *yaml.Node, idx *index.SpecIndex) error {
	// not implemented.
	return nil
}
