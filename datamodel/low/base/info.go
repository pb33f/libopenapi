// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"crypto/sha256"
	"fmt"
	"github.com/pb33f/libopenapi/utils"
	"sort"
	"strings"

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
	Summary        low.NodeReference[string]
	Description    low.NodeReference[string]
	TermsOfService low.NodeReference[string]
	Contact        low.NodeReference[*Contact]
	License        low.NodeReference[*License]
	Version        low.NodeReference[string]
	Extensions     map[low.KeyReference[string]]low.ValueReference[any]
	*low.Reference
}

// FindExtension attempts to locate an extension with the supplied key
func (i *Info) FindExtension(ext string) *low.ValueReference[any] {
	return low.FindItemInMap(ext, i.Extensions)
}

// GetExtensions returns all extensions for Info
func (i *Info) GetExtensions() map[low.KeyReference[string]]low.ValueReference[any] {
	return i.Extensions
}

// Build will extract out the Contact and Info objects from the supplied root node.
func (i *Info) Build(_, root *yaml.Node, idx *index.SpecIndex) error {
	root = utils.NodeAlias(root)
	utils.CheckForMergeNodes(root)
	i.Reference = new(low.Reference)
	i.Extensions = low.ExtractExtensions(root)

	// extract contact
	contact, _ := low.ExtractObject[*Contact](ContactLabel, root, idx)
	i.Contact = contact

	// extract license
	lic, _ := low.ExtractObject[*License](LicenseLabel, root, idx)
	i.License = lic
	return nil
}

// Hash will return a consistent SHA256 Hash of the Info object
func (i *Info) Hash() [32]byte {
	var f []string

	if !i.Title.IsEmpty() {
		f = append(f, i.Title.Value)
	}
	if !i.Summary.IsEmpty() {
		f = append(f, i.Summary.Value)
	}
	if !i.Description.IsEmpty() {
		f = append(f, i.Description.Value)
	}
	if !i.TermsOfService.IsEmpty() {
		f = append(f, i.TermsOfService.Value)
	}
	if !i.Contact.IsEmpty() {
		f = append(f, low.GenerateHashString(i.Contact.Value))
	}
	if !i.License.IsEmpty() {
		f = append(f, low.GenerateHashString(i.License.Value))
	}
	if !i.Version.IsEmpty() {
		f = append(f, i.Version.Value)
	}
	keys := make([]string, len(i.Extensions))
	z := 0
	for k := range i.Extensions {
		keys[z] = fmt.Sprintf("%s-%x", k.Value, sha256.Sum256([]byte(fmt.Sprint(i.Extensions[k].Value))))
		z++
	}
	sort.Strings(keys)
	f = append(f, keys...)
	return sha256.Sum256([]byte(strings.Join(f, "|")))
}
