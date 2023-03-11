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

// License is a low-level representation of a License object as defined by OpenAPI 2 and OpenAPI 3
//  v2 - https://swagger.io/specification/v2/#licenseObject
//  v3 - https://spec.openapis.org/oas/v3.1.0#license-object
type License struct {
	Name low.NodeReference[string]
	URL  low.NodeReference[string]
	*low.Reference
}

// Build is not implemented for License (there is nothing to build)
func (l *License) Build(root *yaml.Node, idx *index.SpecIndex) error {
	l.Reference = new(low.Reference)
	return nil
}

// Hash will return a consistent SHA256 Hash of the License object
func (l *License) Hash() [32]byte {
	var f []string
	if !l.Name.IsEmpty() {
		f = append(f, l.Name.Value)
	}
	if !l.URL.IsEmpty() {
		f = append(f, l.URL.Value)
	}
	return sha256.Sum256([]byte(strings.Join(f, "|")))
}
