// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"testing"

	"github.com/pb33f/testify/assert"
)

func TestCanonicalReferenceIdentity(t *testing.T) {
	assert.Equal(t, "#/Thing", CanonicalReferenceIdentity("#/Thing"))
	assert.Equal(t, "/schemas/thing.yaml#/Thing", CanonicalReferenceIdentity("/schemas/./thing.yaml#/Thing"))
	assert.Equal(t, "/schemas/thing.yaml#/Thing", CanonicalReferenceIdentity(`C:\schemas\thing.yaml#/Thing`))
	assert.Equal(t, "https://example.com/schemas/thing.yaml#/Thing",
		CanonicalReferenceIdentity("https://example.com/schemas/other/../thing.yaml#/Thing"))
}
