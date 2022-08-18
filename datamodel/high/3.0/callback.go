// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import low "github.com/pb33f/libopenapi/datamodel/low/3.0"

type Callback struct {
    Expression map[string]*PathItem
    Extensions map[string]any
    low        *low.Callback
}

func (c *Callback) GoLow() *low.Callback {
    return c.low
}
