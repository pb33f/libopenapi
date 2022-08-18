// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import low "github.com/pb33f/libopenapi/datamodel/low/3.0"

type Example struct {
	Summary       string
	Description   string
	Value         any
	ExternalValue string
	Extensions    map[string]any
	low           *low.Example
}

func (e *Example) GoLow() *low.Example {
	return e.low
}
