// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"gopkg.in/yaml.v3"
)

const (
	ScopesLabel = "scopes"
)

type SecurityScheme struct {
	Type             low.NodeReference[string]
	Description      low.NodeReference[string]
	Name             low.NodeReference[string]
	In               low.NodeReference[string]
	Flow             low.NodeReference[string]
	AuthorizationUrl low.NodeReference[string]
	TokenUrl         low.NodeReference[string]
	Scopes           low.NodeReference[*Scopes]
	Extensions       map[low.KeyReference[string]]low.ValueReference[any]
}

func (ss *SecurityScheme) Build(root *yaml.Node, idx *index.SpecIndex) error {
	ss.Extensions = low.ExtractExtensions(root)

	scopes, sErr := low.ExtractObject[*Scopes](ScopesLabel, root, idx)
	if sErr != nil {
		return sErr
	}
	ss.Scopes = scopes
	return nil
}
