// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import low "github.com/pb33f/libopenapi/datamodel/low/3.0"

type SecurityRequirement struct {
	ValueRequirements []map[string][]string
	low               *low.SecurityRequirement
}

func (s *SecurityRequirement) GoLow() *low.SecurityRequirement {
	return s.low
}
