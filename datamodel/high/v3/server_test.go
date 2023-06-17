// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestServer_MarshalYAML(t *testing.T) {

	server := &Server{
		URL:         "https://pb33f.io",
		Description: "the b33f",
	}

	desired := `url: https://pb33f.io
description: the b33f`

	rend, _ := server.Render()
	assert.Equal(t, desired, strings.TrimSpace(string(rend)))

	// mutate
	server.Variables = map[string]*ServerVariable{
		"rainbow": {
			Enum: []string{"one", "two", "three"},
		},
	}

	desired = `url: https://pb33f.io
description: the b33f
variables:
    rainbow:
        enum:
            - one
            - two
            - three`

	rend, _ = server.Render()
	assert.Equal(t, desired, strings.TrimSpace(string(rend)))
}
