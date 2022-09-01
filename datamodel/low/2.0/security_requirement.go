// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import "github.com/pb33f/libopenapi/datamodel/low"

type SecurityRequirement struct {
	Values low.NodeReference[[]low.ValueReference[string]]
}
