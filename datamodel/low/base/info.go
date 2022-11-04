// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"gopkg.in/yaml.v3"
)

// Info represents a low-level Info object as defined by both OpenAPI 2 and OpenAPI 3.
//
// The object provides metadata about the API. The metadata MAY be used by the clients if needed, and MAY be presented
// in editing or documentation generation tools for convenience.
//
//	v2 - https://swagger.io/specification/v2/#infoObject
//	v3 - https://spec.openapis.org/oas/v3.1.0#info-object
type Info struct {
	Title          low.NodeReference[string]
	Description    low.NodeReference[string]
	TermsOfService low.NodeReference[string]
	Contact        low.NodeReference[*Contact]
	License        low.NodeReference[*License]
	Version        low.NodeReference[string]
	Extensions     map[low.KeyReference[string]]low.ValueReference[any]
}

// FindExtension attempts to locate an extension with the supplied key
func (i *Info) FindExtension(ext string) *low.ValueReference[any] {
	return low.FindItemInMap(ext, i.Extensions)
}

// Build will extract out the Contact and Info objects from the supplied root node.
func (i *Info) Build(root *yaml.Node, idx *index.SpecIndex) error {
	i.Extensions = low.ExtractExtensions(root)

	// extract contact
	contact, _ := low.ExtractObject[*Contact](ContactLabel, root, idx)
	i.Contact = contact

	// extract license
	lic, _ := low.ExtractObject[*License](LicenseLabel, root, idx)
	i.License = lic
	return nil
}
