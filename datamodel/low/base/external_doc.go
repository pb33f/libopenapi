// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"gopkg.in/yaml.v3"
)

// ExternalDoc represents a low-level External Documentation object as defined by OpenAPI 2 and 3
//
// Allows referencing an external resource for extended documentation.
//  v2 - https://swagger.io/specification/v2/#externalDocumentationObject
//  v3 - https://spec.openapis.org/oas/v3.1.0#external-documentation-object
type ExternalDoc struct {
	Description low.NodeReference[string]
	URL         low.NodeReference[string]
	Extensions  map[low.KeyReference[string]]low.ValueReference[any]
}

// FindExtension returns a ValueReference containing the extension value, if found.
func (ex *ExternalDoc) FindExtension(ext string) *low.ValueReference[any] {
	return low.FindItemInMap[any](ext, ex.Extensions)
}

// Build will extract extensions from the ExternalDoc instance.
func (ex *ExternalDoc) Build(root *yaml.Node, idx *index.SpecIndex) error {
	ex.Extensions = low.ExtractExtensions(root)
	return nil
}
