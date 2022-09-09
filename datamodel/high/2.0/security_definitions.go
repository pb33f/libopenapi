// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import low "github.com/pb33f/libopenapi/datamodel/low/2.0"

type SecurityDefinitions struct {
	Values map[string]*SecurityScheme
	low    *low.SecurityDefinitions
}

func NewSecurityDefinitions(definitions *low.SecurityDefinitions) *SecurityDefinitions {
	sd := new(SecurityDefinitions)
	sd.low = definitions
	schemes := make(map[string]*SecurityScheme)
	for k := range definitions.Definitions {
		schemes[k.Value] = NewSecurityScheme(definitions.Definitions[k].Value)
	}
	sd.Values = schemes
	return sd
}

func (sd *SecurityDefinitions) GoLow() *low.SecurityDefinitions {
	return sd.low
}
