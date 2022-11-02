// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"github.com/pb33f/libopenapi/datamodel/low/base"
)

// SecurityRequirement is a high-level representation of a Swagger / OpenAPI 2 SecurityRequirement object.
//
// SecurityRequirement lists the required security schemes to execute this operation. The object can have multiple
// security schemes declared in it which are all required (that is, there is a logical AND between the schemes).
//
// The name used for each property MUST correspond to a security scheme declared in the Security Definitions
//  - https://swagger.io/specification/v2/#securityDefinitionsObject
type SecurityRequirement struct {
	Requirements map[string][]string
	low          *base.SecurityRequirement
}

// NewSecurityRequirement creates a new high-level SecurityRequirement from a low-level one.
func NewSecurityRequirement(req *base.SecurityRequirement) *SecurityRequirement {
	r := new(SecurityRequirement)
	r.low = req
	values := make(map[string][]string)
	// to keep things fast, avoiding copying anything - makes it a little hard to read.
	for reqK := range req.Requirements.Value {
		var vals []string
		for valK := range req.Requirements.Value[reqK].Value {
			vals = append(vals, req.Requirements.Value[reqK].Value[valK].Value)
		}
		values[reqK.Value] = vals
	}
	r.Requirements = values
	return r
}

// GoLow returns the low-level SecurityRequirement used to create the high-level one.
func (s *SecurityRequirement) GoLow() *base.SecurityRequirement {
	return s.low
}
