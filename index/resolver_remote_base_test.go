// Copyright 2026 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package index

import (
	"net/url"
	"testing"

	"github.com/pb33f/testify/require"
)

func TestRootRelativeLookupUsesRemoteBase(t *testing.T) {
	base, err := url.Parse("https://source.example/specs/")
	require.NoError(t, err)
	cfg := CreateOpenAPIIndexConfig()
	cfg.BaseURL = base
	resolver := NewResolver(NewSpecIndexWithConfig(nil, cfg))
	parent := &Reference{FullDefinition: "#/components/schemas/Item"}
	for _, tc := range []struct{ name, value, schemaBase, definition, full string }{
		{"fragment", "types.yaml#/Item", "", "#/Item", "https://source.example/specs/types.yaml#/Item"},
		{"whole file", "../types.yaml", "", "../types.yaml", "https://source.example/types.yaml"},
		{"schema ID", "types.yaml#/Item", "https://schema.example/models/owner.json", "#/Item", "https://schema.example/models/types.yaml#/Item"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			definition, full := resolver.buildRelativeLookupDefinitions(parent, tc.value, tc.schemaBase)
			require.Equal(t, tc.definition, definition)
			require.Equal(t, tc.full, full)
		})
	}
}
