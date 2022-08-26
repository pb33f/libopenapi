// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import low "github.com/pb33f/libopenapi/datamodel/low/3.0"

type Info struct {
	Title          string
	Description    string
	TermsOfService string
	Contact        *Contact
	License        *License
	Version        string
	low            *low.Info
}

func NewInfo(info *low.Info) *Info {
	i := new(Info)
	i.low = info
	i.Title = info.Title.Value
	i.Description = info.Description.Value
	i.TermsOfService = info.TermsOfService.Value
	if !info.Contact.IsEmpty() {
		i.Contact = NewContact(info.Contact.Value)
	}
	if !info.License.IsEmpty() {
		i.License = NewLicense(info.License.Value)
	}
	i.Version = info.Version.Value
	return i
}

func (i *Info) GoLow() *low.Info {
	return i.low
}
