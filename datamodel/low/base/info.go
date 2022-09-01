// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"gopkg.in/yaml.v3"
)

const (
	InfoLabel    = "info"
	ContactLabel = "contact"
	LicenseLabel = "license"
)

type Info struct {
	Title          low.NodeReference[string]
	Description    low.NodeReference[string]
	TermsOfService low.NodeReference[string]
	Contact        low.NodeReference[*Contact]
	License        low.NodeReference[*License]
	Version        low.NodeReference[string]
}

func (i *Info) Build(root *yaml.Node, idx *index.SpecIndex) error {
	// extract contact
	contact, _ := low.ExtractObject[*Contact](ContactLabel, root, idx)
	i.Contact = contact

	// extract license
	lic, _ := low.ExtractObject[*License](LicenseLabel, root, idx)
	i.License = lic
	return nil
}
