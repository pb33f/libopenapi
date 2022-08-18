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

func NewServerVariable(variable *low.ServerVariable) *ServerVariable {
	v := new(ServerVariable)
	v.low = variable
	var enums []string
	for _, enum := range variable.Enum {
		if enum.Value != "" {
			enums = append(enums, enum.Value)
		}
	}
	v.Default = variable.Default.Value
	v.Description = variable.Description.Value
	v.Enum = enums
	return v
}

func (s *ServerVariable) GoLow() *low.ServerVariable {
	return s.low
}
