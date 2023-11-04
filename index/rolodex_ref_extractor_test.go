// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExtractedRef_GetFile(t *testing.T) {

	a := &ExtractedRef{Location: "#/components/schemas/One", Type: Local}
	assert.Equal(t, "#/components/schemas/One", a.GetFile())

	a = &ExtractedRef{Location: "pizza.yaml#/components/schemas/One", Type: File}
	assert.Equal(t, "pizza.yaml", a.GetFile())

	a = &ExtractedRef{Location: "https://api.pb33f.io/openapi.yaml#/components/schemas/One", Type: File}
	assert.Equal(t, "https://api.pb33f.io/openapi.yaml", a.GetFile())

}

func TestExtractedRef_GetReference(t *testing.T) {

	a := &ExtractedRef{Location: "#/components/schemas/One", Type: Local}
	assert.Equal(t, "#/components/schemas/One", a.GetReference())

	a = &ExtractedRef{Location: "pizza.yaml#/components/schemas/One", Type: File}
	assert.Equal(t, "#/components/schemas/One", a.GetReference())

	a = &ExtractedRef{Location: "https://api.pb33f.io/openapi.yaml#/components/schemas/One", Type: File}
	assert.Equal(t, "#/components/schemas/One", a.GetReference())

}
