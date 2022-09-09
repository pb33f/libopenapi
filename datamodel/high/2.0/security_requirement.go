// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import low "github.com/pb33f/libopenapi/datamodel/low/2.0"

type SecurityRequirement struct {
	Requirements map[string][]string
	low          *low.SecurityRequirement
}

func NewSecurityRequirement(req *low.SecurityRequirement) *SecurityRequirement {
	r := new(SecurityRequirement)
	r.low = req
	var values map[string][]string
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

func (s *SecurityRequirement) GoLow() *low.SecurityRequirement {
	return s.low
}
