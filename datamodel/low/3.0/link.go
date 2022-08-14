// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"gopkg.in/yaml.v3"
)

type Link struct {
	OperationRef low.NodeReference[string]
	OperationId  low.NodeReference[string]
	Parameters   low.KeyReference[map[low.KeyReference[string]]low.ValueReference[string]]
	RequestBody  low.NodeReference[string]
	Description  low.NodeReference[string]
	Server       low.NodeReference[*Server]
	Extensions   map[low.KeyReference[string]]low.ValueReference[any]
}

func (l *Link) FindParameter(pName string) *low.ValueReference[string] {
	return low.FindItemInMap[string](pName, l.Parameters.Value)
}

func (l *Link) FindExtension(ext string) *low.ValueReference[any] {
	return low.FindItemInMap[any](ext, l.Extensions)
}

func (l *Link) Build(root *yaml.Node, idx *index.SpecIndex) error {
	l.Extensions = low.ExtractExtensions(root)

	// extract server.
	ser, sErr := low.ExtractObject[*Server](ServerLabel, root, idx)
	if sErr != nil {
		return sErr
	}
	l.Server = ser

	return nil
}
