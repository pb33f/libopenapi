// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"crypto/sha256"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"gopkg.in/yaml.v3"
	"strings"
)

// Contact represents a low-level representation of the Contact definitions found at
//
//	v2 - https://swagger.io/specification/v2/#contactObject
//	v3 - https://spec.openapis.org/oas/v3.1.0#contact-object
type Contact struct {
	Name  low.NodeReference[string]
	URL   low.NodeReference[string]
	Email low.NodeReference[string]
	*low.Reference
}

// Build is not implemented for Contact (there is nothing to build).
func (c *Contact) Build(_, _ *yaml.Node, _ *index.SpecIndex) error {
	c.Reference = new(low.Reference)
	// not implemented.
	return nil
}

// Hash will return a consistent SHA256 Hash of the Contact object
func (c *Contact) Hash() [32]byte {
	var f []string
	if !c.Name.IsEmpty() {
		f = append(f, c.Name.Value)
	}
	if !c.URL.IsEmpty() {
		f = append(f, c.URL.Value)
	}
	if !c.Email.IsEmpty() {
		f = append(f, c.Email.Value)
	}
	return sha256.Sum256([]byte(strings.Join(f, "|")))
}
