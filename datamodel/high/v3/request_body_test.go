// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestRequestBody_MarshalYAML(t *testing.T) {
	ext := orderedmap.New[string, *yaml.Node]()
	ext.Set("x-high-gravity", utils.CreateStringNode("why not?"))

	rb := true
	req := &RequestBody{
		Description: "beer",
		Required:    &rb,
		Extensions:  ext,
	}

	rend, _ := req.Render()

	desired := `description: beer
required: true
x-high-gravity: why not?`

	assert.Equal(t, desired, strings.TrimSpace(string(rend)))
}

func TestRequestBody_MarshalYAMLInline(t *testing.T) {
	ext := orderedmap.New[string, *yaml.Node]()
	ext.Set("x-high-gravity", utils.CreateStringNode("why not?"))

	rb := true
	req := &RequestBody{
		Description: "beer",
		Required:    &rb,
		Extensions:  ext,
	}

	rend, _ := req.RenderInline()

	desired := `description: beer
required: true
x-high-gravity: why not?`

	assert.Equal(t, desired, strings.TrimSpace(string(rend)))
}

func TestRequestBody_MarshalNoRequired(t *testing.T) {
	ext := orderedmap.New[string, *yaml.Node]()
	ext.Set("x-high-gravity", utils.CreateStringNode("why not?"))

	rb := false
	req := &RequestBody{
		Description: "beer",
		Required:    &rb,
		Extensions:  ext,
	}

	rend, _ := req.Render()

	desired := `description: beer
required: false
x-high-gravity: why not?`

	assert.Equal(t, desired, strings.TrimSpace(string(rend)))
}

func TestRequestBody_MarshalRequiredNil(t *testing.T) {
	ext := orderedmap.New[string, *yaml.Node]()
	ext.Set("x-high-gravity", utils.CreateStringNode("why not?"))

	req := &RequestBody{
		Description: "beer",
		Extensions:  ext,
	}

	rend, _ := req.Render()

	desired := `description: beer
x-high-gravity: why not?`

	assert.Equal(t, desired, strings.TrimSpace(string(rend)))
}
