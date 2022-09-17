// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import low "github.com/pb33f/libopenapi/datamodel/low/v3"

// Callback represents a high-level Callback object for OpenAPI 3+.
//
// A map of possible out-of band callbacks related to the parent operation. Each value in the map is a
// PathItem Object that describes a set of requests that may be initiated by the API provider and the expected
// responses. The key value used to identify the path item object is an expression, evaluated at runtime,
// that identifies a URL to use for the callback operation.
//  - https://spec.openapis.org/oas/v3.1.0#callback-object
type Callback struct {
	Expression map[string]*PathItem
	Extensions map[string]any
	low        *low.Callback
}

// NewCallback creates a new high-level callback from a low-level one.
func NewCallback(lowCallback *low.Callback) *Callback {
	n := new(Callback)
	n.low = lowCallback
	n.Expression = make(map[string]*PathItem)
	for i := range lowCallback.Expression.Value {
		n.Expression[i.Value] = NewPathItem(lowCallback.Expression.Value[i].Value)
	}
	n.Extensions = make(map[string]any)
	for k, v := range lowCallback.Extensions {
		n.Extensions[k.Value] = v.Value
	}
	return n
}

// GoLow returns the low-level Callback instance used to create the high-level one.
func (c *Callback) GoLow() *low.Callback {
	return c.low
}
