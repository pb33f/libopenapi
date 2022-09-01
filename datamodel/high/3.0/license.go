// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	low "github.com/pb33f/libopenapi/datamodel/low/shared"
)

type License struct {
	Name string
	URL  string
	low  *low.License
}

func NewLicense(license *low.License) *License {
	l := new(License)
	l.low = license
	if !license.URL.IsEmpty() {
		l.URL = license.URL.Value
	}
	if !license.Name.IsEmpty() {
		l.Name = license.Name.Value
	}
	return l
}

func (l *License) GoLow() *low.License {
	return l.low
}
