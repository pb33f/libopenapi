// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
    "github.com/pb33f/libopenapi/datamodel/high/base"
    "github.com/stretchr/testify/assert"
    "strings"
    "testing"
)

func TestParameter_MarshalYAML(t *testing.T) {

    explode := true
    param := Parameter{
        Name:          "chicken",
        In:            "nuggets",
        Description:   "beefy",
        Deprecated:    true,
        Style:         "simple",
        Explode:       &explode,
        AllowReserved: true,
        Example:       "example",
        Examples:      map[string]*base.Example{"example": {Value: "example"}},
        Extensions:    map[string]interface{}{"x-burgers": "why not?"},
    }

    rend, _ := param.Render()

    desired := `name: chicken
in: nuggets
description: beefy
deprecated: true
style: simple
explode: true
allowReserved: true
example: example
examples:
    example:
        value: example
x-burgers: why not?`

    assert.Equal(t, desired, strings.TrimSpace(string(rend)))
}
