// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
)

func TestHeader_MarshalYAML(t *testing.T) {

	header := &Header{
		Description:     "A header",
		Required:        true,
		Deprecated:      true,
		AllowEmptyValue: true,
		Style:           "simple",
		Explode:         true,
		AllowReserved:   true,
		Example:         "example",
		Examples: orderedmap.ToOrderedMap(map[string]*base.Example{
			"example": {Value: "example"},
		}),
		Extensions: map[string]interface{}{"x-burgers": "why not?"},
	}

	rend, _ := header.Render()

	desired := `description: A header
required: true
deprecated: true
allowEmptyValue: true
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
