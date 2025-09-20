// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
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
	server.Variables = orderedmap.ToOrderedMap(map[string]*ServerVariable{
		"rainbow": {
			Enum: []string{"one", "two", "three"},
		},
	})

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

func TestServer_Name_OpenAPI32(t *testing.T) {
	server := &Server{
		Name:        "Production Server",
		URL:         "https://api.example.com",
		Description: "Main production API server",
	}

	desired := `name: Production Server
url: https://api.example.com
description: Main production API server`

	rend, _ := server.Render()
	assert.Equal(t, desired, strings.TrimSpace(string(rend)))
}

func TestServer_Name_WithVariables(t *testing.T) {
	server := &Server{
		Name:        "Staging Server",
		URL:         "https://{environment}.api.example.com",
		Description: "Staging environment server",
		Variables: orderedmap.ToOrderedMap(map[string]*ServerVariable{
			"environment": {
				Default:     "staging",
				Enum:        []string{"staging", "dev", "test"},
				Description: "The environment name",
			},
		}),
	}

	rend, _ := server.Render()
	rendStr := strings.TrimSpace(string(rend))

	// Test that all required fields are present
	assert.Contains(t, rendStr, "name: Staging Server")
	assert.Contains(t, rendStr, "url: https://{environment}.api.example.com")
	assert.Contains(t, rendStr, "description: Staging environment server")
	assert.Contains(t, rendStr, "default: staging")
	assert.Contains(t, rendStr, "description: The environment name")
	assert.Contains(t, rendStr, "- staging")
	assert.Contains(t, rendStr, "- dev")
	assert.Contains(t, rendStr, "- test")
}

func TestServer_NoName(t *testing.T) {
	server := &Server{
		URL:         "https://api.example.com",
		Description: "API server without name",
	}

	desired := `url: https://api.example.com
description: API server without name`

	rend, _ := server.Render()
	assert.Equal(t, desired, strings.TrimSpace(string(rend)))

	// Verify Name is empty
	assert.Equal(t, "", server.Name)
}
