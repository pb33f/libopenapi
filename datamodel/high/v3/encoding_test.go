// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncoding_MarshalYAML(t *testing.T) {
	t.Parallel()
	explode := true
	encoding := &Encoding{
		ContentType: "application/json",
		Headers:     map[string]*Header{"x-pizza-time": {Description: "oh yes please"}},
		Style:       "simple",
		Explode:     &explode,
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
