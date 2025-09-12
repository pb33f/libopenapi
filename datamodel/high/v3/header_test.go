// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestHeader_MarshalYAML(t *testing.T) {
	ext := orderedmap.New[string, *yaml.Node]()
	ext.Set("x-burgers", utils.CreateStringNode("why not?"))

	header := &Header{
		Description:     "A header",
		Required:        true,
		Deprecated:      true,
		AllowEmptyValue: true,
		Style:           "simple",
		Explode:         true,
		AllowReserved:   true,
		Example:         utils.CreateStringNode("example"),
		Examples: orderedmap.ToOrderedMap(map[string]*base.Example{
			"example": {Value: utils.CreateStringNode("example")},
		}),
		Extensions: ext,
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
