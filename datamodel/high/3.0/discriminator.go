// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	low "github.com/pb33f/libopenapi/datamodel/low/base"
)

type Discriminator struct {
	PropertyName string
	Mapping      map[string]string
	low          *low.Discriminator
}

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

func (d *Discriminator) GoLow() *low.Discriminator {
	return d.low
}
