// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import "github.com/pb33f/libopenapi/datamodel/low"

type Scopes struct {
	Values     low.NodeReference[map[low.KeyReference[string]]low.ValueReference[string]]
	Extensions map[low.KeyReference[string]]low.ValueReference[any]
}
