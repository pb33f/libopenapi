// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import low "github.com/pb33f/libopenapi/datamodel/low/2.0"

type Examples struct {
	Values map[string]any
	low    *low.Examples
}

func NewExamples(examples *low.Examples) *Examples {
	e := new(Examples)
	e.low = examples
	if len(e.Values) > 0 {
		values := make(map[string]any)
		for k := range e.Values {
			values[k] = e.Values[k]
		}
		e.Values = values
	}
	return e
}

func (e *Examples) GoLow() *low.Examples {
	return e.low
}
