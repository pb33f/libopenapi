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

func (i *Info) GoLow() *low.Info {
	return i.low
}
