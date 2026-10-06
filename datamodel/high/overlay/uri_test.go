// Copyright 2022-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package overlay

import (
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"testing"
)

func TestOverlayResolveExtends(t *testing.T) {
	for _, tc := range []struct{ name, self, base, extends, want string }{
		{"self wins", "https://example.com/overlays/a.yaml", "file:///local/a.yaml", "../api.yaml", "https://example.com/api.yaml"},
		{"retrieval", "", "file:///local/overlays/a.yaml", "../api.yaml", "file:///local/api.yaml"},
		{"relative self", "../shared/a.yaml", "https://example.com/overlays/a.yaml", "../api.yaml", "https://example.com/api.yaml"},
		{"absolute extends", "", "", "https://example.com/api.yaml", "https://example.com/api.yaml"},
		{"unresolved relative", "", "", "../api.yaml", "../api.yaml"},
		{"relative self without base", "overlays/a.yaml", "", "../api.yaml", "../api.yaml"},
		{"parent self without base", "../overlays/a.yaml", "", "api.yaml", "api.yaml"},
		{"relative retrieval", "", "overlays/a.yaml", "../api.yaml", "../api.yaml"},
		{"empty", "https://example.com/a.yaml", "", "", ""},
		{"query", "https://example.com/a.yaml", "", "?v=2", "https://example.com/a.yaml?v=2"},
		{"encoded fragment", "", "", "https://example.com/a%23b.yaml", "https://example.com/a%23b.yaml"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := (&Overlay{Self: tc.self, Extends: tc.extends}).ResolveExtends(tc.base)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
	for _, value := range []string{"a.yaml#", "a.yaml#part", "%zz", "a b.yaml", "https://[bad"} {
		_, err := (&Overlay{Self: value}).ResolveExtends("")
		require.Error(t, err)
		_, err = (&Overlay{Extends: value}).ResolveExtends("")
		require.Error(t, err)
		_, err = (&Overlay{}).ResolveExtends(value)
		require.Error(t, err)
	}
}
