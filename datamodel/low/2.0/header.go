// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
    "github.com/pb33f/libopenapi/datamodel/low"
    "github.com/pb33f/libopenapi/index"
    "gopkg.in/yaml.v3"
)

type Header struct {
    Type             low.NodeReference[string]
    Format           low.NodeReference[string]
    Description      low.NodeReference[string]
    Items            low.NodeReference[*Items]
    CollectionFormat low.NodeReference[string]
    Default          low.NodeReference[any]
    Maximum          low.NodeReference[int]
    ExclusiveMaximum low.NodeReference[bool]
    Minimum          low.NodeReference[int]
    ExclusiveMinimum low.NodeReference[bool]
    MaxLength        low.NodeReference[int]
    MinLength        low.NodeReference[int]
    Pattern          low.NodeReference[string]
    MaxItems         low.NodeReference[int]
    MinItems         low.NodeReference[int]
    UniqueItems      low.NodeReference[bool]
    Enum             low.NodeReference[[]string]
    MultipleOf       low.NodeReference[int]
    Extensions       map[low.KeyReference[string]]low.ValueReference[any]
}

func (h *Header) FindExtension(ext string) *low.ValueReference[any] {
    return low.FindItemInMap[any](ext, h.Extensions)
}

func (h *Header) Build(root *yaml.Node, idx *index.SpecIndex) error {
    h.Extensions = low.ExtractExtensions(root)

    // TODO: build items.

    return nil
}

