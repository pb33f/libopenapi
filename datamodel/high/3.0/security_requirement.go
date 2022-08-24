// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import low "github.com/pb33f/libopenapi/datamodel/low/3.0"

type SecurityRequirement struct {
	ValueRequirements []map[string][]string
	low               *low.SecurityRequirement
}

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

func (s *SecurityRequirement) GoLow() *low.SecurityRequirement {
	return s.low
}
