// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestServerVariable_MarshalYAML(t *testing.T) {
	svar := &ServerVariable{
		Enum:        []string{"one", "two", "three"},
		Description: "money day",
	}

	desired := `enum:
    - one
    - two
    - three
description: money day`

	svarRend, _ := svar.Render()

	assert.Equal(t, desired, strings.TrimSpace(string(svarRend)))

	// mutate

	svar.Default = "is moments away"

	desired = `enum:
    - one
    - two
    - three
default: is moments away
description: money day`

	svarRend, _ = svar.Render()

	assert.Equal(t, desired, strings.TrimSpace(string(svarRend)))
}

func TestServerVariableExtension_MarshalYAML(t *testing.T) {
	createExtension := func(value interface{}) *yaml.Node {
		node := &yaml.Node{}
		err := node.Encode(value)
		if err != nil {
			// Trate o erro conforme necessário
		}
		return node
	}

	svar := &ServerVariable{
		Extensions: orderedmap.New[string, *yaml.Node](),
	}
	transform := []map[string]interface{}{
		{
			"type":         "translate",
			"allowMissing": true,
			"translations": []map[string]string{
				{"from": "pt-br", "to": "en-us"},
			},
		},
	}
	svar.Extensions.Set("x-transforms", createExtension(transform))

	desired := `x-transforms:
    - allowMissing: true
      translations:
        - from: pt-br
          to: en-us
      type: translate`

	svarRend, _ := svar.Render()

	assert.Equal(t, desired, strings.TrimSpace(string(svarRend)))

	// mutate

	svar.Default = "es-mx"
	transform = []map[string]interface{}{
		{
			"type":         "translate",
			"allowMissing": true,
			"translations": []map[string]string{
				{"from": "es-mx", "to": "en-us"},
			},
		},
	}
	svar.Extensions.Set("x-transforms", createExtension(transform))

	desired = `default: es-mx
x-transforms:
    - allowMissing: true
      translations:
        - from: es-mx
          to: en-us
      type: translate`

	svarRend, _ = svar.Render()

	assert.Equal(t, desired, strings.TrimSpace(string(svarRend)))
}
