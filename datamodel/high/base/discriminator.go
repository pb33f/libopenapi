// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	low "github.com/pb33f/libopenapi/datamodel/low/base"
)

// Discriminator is only used by OpenAPI 3+ documents, it represents a polymorphic discriminator used for schemas
//  v3 - https://spec.openapis.org/oas/v3.1.0#discriminator-object
type Discriminator struct {
	PropertyName string
	Mapping      map[string]string
	low          *low.Discriminator
}

// NewDiscriminator will create a new high-level Discriminator from a low-level one.
func NewDiscriminator(disc *low.Discriminator) *Discriminator {
	d := new(Discriminator)
	d.low = disc
	d.PropertyName = disc.PropertyName.Value
	mapping := make(map[string]string)
	for k, v := range disc.Mapping {
		mapping[k.Value] = v.Value
	}
	d.Mapping = mapping
	return d
}

// GoLow returns the low-level Discriminator used to build the high-level one.
func (d *Discriminator) GoLow() *low.Discriminator {
	return d.low
}
