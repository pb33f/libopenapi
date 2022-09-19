// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import low "github.com/pb33f/libopenapi/datamodel/low/v2"

// SecurityRequirement is a high-level representation of a Swagger / OpenAPI 2 SecurityRequirement object.
//
// SecurityRequirement lists the required security schemes to execute this operation. The object can have multiple
// security schemes declared in it which are all required (that is, there is a logical AND between the schemes).
//
// The name used for each property MUST correspond to a security scheme declared in the Security Definitions
//  - https://swagger.io/specification/v2/#securityDefinitionsObject
type SecurityRequirement struct {
	Requirements map[string][]string
	low          *low.SecurityRequirement
}

// NewSecurityRequirement creates a new high-level SecurityRequirement from a low-level one.
func NewSecurityRequirement(req *low.SecurityRequirement) *SecurityRequirement {
	r := new(SecurityRequirement)
	r.low = req
	values := make(map[string][]string)
	// to keep things fast, avoiding copying anything - makes it a little hard to read.
	for reqK := range req.Values.Value {
		var vals []string
		for valK := range req.Values.Value[reqK].Value {
			vals = append(vals, req.Values.Value[reqK].Value[valK].Value)
		}
		values[reqK.Value] = vals
	}
	r.Requirements = values
	return r
}

// GoLow returns the low-level SecurityRequirement used to create the high-level one.
func (s *SecurityRequirement) GoLow() *low.SecurityRequirement {
	return s.low
}
