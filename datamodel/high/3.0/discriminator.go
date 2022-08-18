// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import low "github.com/pb33f/libopenapi/datamodel/low/3.0"

type Discriminator struct {
	PropertyName string
	Mapping      map[string]string
	low          *low.Discriminator
}

func (d *Discriminator) GoLow() *low.Discriminator {
	return d.low
}
