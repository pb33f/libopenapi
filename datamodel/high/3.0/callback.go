// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import low "github.com/pb33f/libopenapi/datamodel/low/3.0"

type Callback struct {
	Expression map[string]*PathItem
	Extensions map[string]any
	low        *low.Callback
}

func NewCallback(lowCallback *low.Callback) *Callback {
	n := new(Callback)
	n.low = lowCallback
	n.Expression = make(map[string]*PathItem)
	for i := range lowCallback.Expression.Value {
		n.Expression[i.Value] = NewPathItem(lowCallback.Expression.Value[i].Value)
	}
	n.Extensions = make(map[string]any)
	for k, v := range lowCallback.Extensions {
		n.Extensions[k.Value] = v
	}
	return n
}

func (c *Callback) GoLow() *low.Callback {
	return c.low
}
