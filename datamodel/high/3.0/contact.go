// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import low "github.com/pb33f/libopenapi/datamodel/low/3.0"

type Contact struct {
	Name  string
	URL   string
	Email string
	low   *low.Contact
}

func (c *Contact) GoLow() *low.Contact {
	return c.low
}
