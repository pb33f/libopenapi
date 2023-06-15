// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
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
