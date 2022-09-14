// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	low "github.com/pb33f/libopenapi/datamodel/low/base"
)

// Contact represents a high-level representation of the Contact definitions found at
//  v2 - https://swagger.io/specification/v2/#contactObject
//  v3 - https://spec.openapis.org/oas/v3.1.0#contact-object
type Contact struct {
	Name  string
	URL   string
	Email string
	low   *low.Contact
}

// NewContact will create a new Contact instance using a low-level Contact
func NewContact(contact *low.Contact) *Contact {
	c := new(Contact)
	c.low = contact
	c.URL = contact.URL.Value
	c.Name = contact.Name.Value
	c.Email = contact.Email.Value
	return c
}

// GoLow returns the low level Contact object used to create the high-level one.
func (c *Contact) GoLow() *low.Contact {
	return c.low
}
