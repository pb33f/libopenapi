// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import low "github.com/pb33f/libopenapi/datamodel/low/v2"

type Examples struct {
	Values map[string]any
	low    *low.Examples
}

func NewExamples(examples *low.Examples) *Examples {
	e := new(Examples)
	e.low = examples
	if len(examples.Values) > 0 {
		values := make(map[string]any)
		for k := range examples.Values {
			values[k.Value] = examples.Values[k].Value
		}
		e.Values = values
	}
	return e
}

func (e *Examples) GoLow() *low.Examples {
	return e.low
}
