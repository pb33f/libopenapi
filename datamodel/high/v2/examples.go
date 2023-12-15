// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v2

import (
	low "github.com/pb33f/libopenapi/datamodel/low/v2"
	"github.com/pb33f/libopenapi/orderedmap"
	"gopkg.in/yaml.v3"
)

// Example represents a high-level Swagger / OpenAPI 2 Example object, backed by a low level one.
// Allows sharing examples for operation responses
//   - https://swagger.io/specification/v2/#exampleObject
type Example struct {
	Values *orderedmap.Map[string, *yaml.Node]
	low    *low.Examples
}

// NewExample creates a new high-level Example instance from a low-level one.
func NewExample(examples *low.Examples) *Example {
	e := new(Example)
	e.low = examples
	if orderedmap.Len(examples.Values) > 0 {
		values := orderedmap.New[string, *yaml.Node]()
		for pair := orderedmap.First(examples.Values); pair != nil; pair = pair.Next() {
			values.Set(pair.Key().Value, pair.Value().Value)
		}
		e.Values = values
	}
	return e
}

// GoLow returns the low-level Example used to create the high-level one.
func (e *Example) GoLow() *low.Examples {
	return e.low
}
