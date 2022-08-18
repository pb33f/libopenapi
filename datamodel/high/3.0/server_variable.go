// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import low "github.com/pb33f/libopenapi/datamodel/low/3.0"

type ServerVariable struct {
	Enum        []string
	Default     string
	Description string
	low         *low.ServerVariable
}

func (s *ServerVariable) GoLow() *low.ServerVariable {
	return s.low
}
