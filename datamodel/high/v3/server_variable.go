// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import low "github.com/pb33f/libopenapi/datamodel/low/v3"

// ServerVariable represents a high-level OpenAPI 3+ ServerVariable object, that is backed by a low-level one.
//
// ServerVariable is an object representing a Server Variable for server URL template substitution.
// - https://spec.openapis.org/oas/v3.1.0#server-variable-object
type ServerVariable struct {
	Enum        []string
	Default     string
	Description string
	low         *low.ServerVariable
}

// NewServerVariable will return a new high-level instance of a ServerVariable from a low-level one.
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

// GoLow returns the low-level ServerVariable used to to create the high\-level one.
func (s *ServerVariable) GoLow() *low.ServerVariable {
	return s.low
}
