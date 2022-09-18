// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import low "github.com/pb33f/libopenapi/datamodel/low/v2"

// Example represents a high-level Swagger / OpenAPI 2 Example object, backed by a low level one.
// Allows sharing examples for operation responses
//  - https://swagger.io/specification/v2/#exampleObject
type Example struct {
	Values map[string]any
	low    *low.Examples
}

// NewExample creates a new high-level Example instance from a low-level one.
func NewExample(examples *low.Examples) *Example {
	e := new(Example)
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

// GoLow returns the low-level Example used to create the high-level one.
func (e *Example) GoLow() *low.Examples {
	return e.low
}
