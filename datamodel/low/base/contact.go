// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"gopkg.in/yaml.v3"
)

// Contact represents a low-level representation of the Contact definitions found at
//  v2 - https://swagger.io/specification/v2/#contactObject
//  v3 - https://spec.openapis.org/oas/v3.1.0#contact-object
type Contact struct {
	Name  low.NodeReference[string]
	URL   low.NodeReference[string]
	Email low.NodeReference[string]
}

// Build is not implemented for Contact (there is nothing to build).
func (c *Contact) Build(root *yaml.Node, idx *index.SpecIndex) error {
	// not implemented.
	return nil
}
