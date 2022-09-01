// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	low "github.com/pb33f/libopenapi/datamodel/low/shared"
)

type Contact struct {
	Name  string
	URL   string
	Email string
	low   *low.Contact
}

func NewContact(contact *low.Contact) *Contact {
	c := new(Contact)
	c.low = contact
	c.URL = contact.URL.Value
	c.Name = contact.Name.Value
	c.Email = contact.Email.Value
	return c
}

func (c *Contact) GoLow() *low.Contact {
	return c.low
}
