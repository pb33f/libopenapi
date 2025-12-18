// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestEncoding_MarshalYAML(t *testing.T) {
	explode := true
	encoding := &Encoding{
		ContentType: "application/json",
		Headers: orderedmap.ToOrderedMap(map[string]*Header{
			"x-pizza-time": {Description: "oh yes please"},
		}),
		Style:   "simple",
		Explode: &explode,
	}

	rend, _ := encoding.Render()

	desired := `contentType: application/json
headers:
    x-pizza-time:
        description: oh yes please
style: simple
explode: true`

	assert.Equal(t, desired, strings.TrimSpace(string(rend)))

	explode = false
	encoding.Explode = &explode
	rend, _ = encoding.Render()

	desired = `contentType: application/json
headers:
    x-pizza-time:
        description: oh yes please
style: simple`

	assert.Equal(t, desired, strings.TrimSpace(string(rend)))

	encoding.Explode = nil
	rend, _ = encoding.Render()

	desired = `contentType: application/json
headers:
    x-pizza-time:
        description: oh yes please
style: simple`

	assert.Equal(t, desired, strings.TrimSpace(string(rend)))

	encoding.Explode = &explode
	rend, _ = encoding.Render()

	desired = `contentType: application/json
headers:
    x-pizza-time:
        description: oh yes please
style: simple`

	assert.Equal(t, desired, strings.TrimSpace(string(rend)))
}

func TestEncoding_MarshalYAMLInlineWithContext(t *testing.T) {
	explode := true
	encoding := &Encoding{
		ContentType: "application/json",
		Headers: orderedmap.ToOrderedMap(map[string]*Header{
			"x-pizza-time": {Description: "oh yes please"},
		}),
		Style:   "simple",
		Explode: &explode,
	}

	ctx := base.NewInlineRenderContext()
	node, err := encoding.MarshalYAMLInlineWithContext(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, node)

	rend, _ := yaml.Marshal(node)

	desired := `contentType: application/json
headers:
    x-pizza-time:
        description: oh yes please
style: simple
explode: true`

	assert.Equal(t, desired, strings.TrimSpace(string(rend)))
}
