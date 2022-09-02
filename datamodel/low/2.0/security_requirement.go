// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"gopkg.in/yaml.v3"
)

const (
	SecurityLabel = "security"
)

type SecurityRequirement struct {
	Values low.NodeReference[[]low.ValueReference[string]]
}

func (s *SecurityRequirement) Build(_ *yaml.Node, _ *index.SpecIndex) error {
	// not implemented.
	return nil
}
