// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import low "github.com/pb33f/libopenapi/datamodel/low/v3"

// SecurityRequirement is a high-level representation of an OpenAPI 3+ SecurityRequirement object that is backed
// by a low-level one.
//
// It lists the required security schemes to execute this operation. The name used for each property MUST correspond
// to a security scheme declared in the Security Schemes under the Components Object.
//
// Security Requirement Objects that contain multiple schemes require that all schemes MUST be satisfied for a
// request to be authorized. This enables support for scenarios where multiple query parameters or HTTP headers are
// required to convey security information.
//
// When a list of Security Requirement Objects is defined on the OpenAPI Object or Operation Object, only one of the
// Security Requirement Objects in the list needs to be satisfied to authorize the request.
//  - https://spec.openapis.org/oas/v3.1.0#security-requirement-object
type SecurityRequirement struct {
	ValueRequirements []map[string][]string
	low               *low.SecurityRequirement
}

// NewSecurityRequirement will create a new high-level SecurityRequirement instance, from a low-level one.
func NewSecurityRequirement(req *low.SecurityRequirement) *SecurityRequirement {
	r := new(SecurityRequirement)
	r.low = req
	var values []map[string][]string
	for i := range req.ValueRequirements {
		valmap := make(map[string][]string)
		for k, v := range req.ValueRequirements[i].Value {
			var mItems []string
			for h := range v {
				mItems = append(mItems, v[h].Value)
			}
			valmap[k.Value] = mItems
		}
		values = append(values, valmap)
	}
	r.ValueRequirements = values
	return r
}

// GoLow returns the low-level SecurityRequirement instance used to create the high-level one.
func (s *SecurityRequirement) GoLow() *low.SecurityRequirement {
	return s.low
}
