// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package datamodel

import (
	"strings"
	"testing"

	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

// TestCheckDuplicateMappingKeys_MatchesDecoder is a differential test: every case is
// run through both checkDuplicateMappingKeys and the yaml v4 decoder (the previous
// source of duplicate-key errors). The walker must agree with the decoder on whether
// an error occurs AND on the exact construct error text.
func TestCheckDuplicateMappingKeys_MatchesDecoder(t *testing.T) {
	cases := []struct {
		name string
		yml  string
	}{
		{"no duplicates", "a: 1\nb: 2\nc:\n  d: 3\n  e: 4\n"},
		{"simple duplicate", "a: 1\nb: 2\na: 3\n"},
		{"nested duplicate", "root:\n  x: 1\n  y: 2\n  x: 3\n"},
		{"duplicate inside sequence", "items:\n  - k: 1\n    k: 2\n  - ok: 1\n"},
		{"multiple duplicates", "a: 1\na: 2\nb: 3\nb: 4\n"},
		{"triple duplicate", "a: 1\na: 2\na: 3\n"},
		{"alias keys", "anchored: &k value\nmap:\n  *k : 1\n  *k : 2\n"},
		{"alias key vs literal", "anchored: &k value\nmap:\n  *k : 1\n  value: 2\n"},
		{"merge keys", "base: &base\n  a: 1\nuses:\n  <<: *base\n  a: 2\n"},
		{"duplicate merge keys", "b1: &b1\n  a: 1\nb2: &b2\n  b: 2\nuses:\n  <<: *b1\n  <<: *b2\n"},
		{"tagged int vs plain", "m:\n  !!str 1: a\n  1: b\n"},
		{"quoted vs plain same value", "m:\n  \"true\": a\n  true: b\n"},
		{"flow map key", "m:\n  {a: 1}: x\n  {a: 1}: y\n"},
		{"duplicate under duplicate", "a:\n  inner: 1\n  inner: 2\na:\n  other: 1\n"},
		{"numeric keys", "m:\n  9: a\n  9: b\n"},
		{"empty mapping", "{}\n"},
		{"deeply nested clean", "a:\n  b:\n    c:\n      - d: 1\n        e: 2\n"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var node yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(tc.yml), &node), "corpus cases must parse")

			// the decoder's view (previous behavior).
			var decoded map[string]interface{}
			decodeErr := node.Decode(&decoded)

			// the walker's view (new behavior).
			walkErr := checkDuplicateMappingKeys(&node)

			if decodeErr != nil {
				require.Error(t, walkErr, "decoder errored but walker did not: %s", decodeErr)
				assert.Equal(t, decodeErr.Error(), walkErr.Error(), "error text must match decoder byte for byte")
			} else {
				assert.NoError(t, walkErr, "walker errored but decoder did not")
			}
		})
	}
}

// TestCheckDuplicateMappingKeys_AliasedAnchorDivergence pins the one KNOWN,
// INTENTIONAL divergence from the decoder: an anchored mapping with duplicate
// keys that is aliased elsewhere is reported once per definition by the
// walker, but once per construction visit (anchor + each alias) by the
// decoder. The duplicate itself is identical; only the repeat count differs.
// If this test starts failing on the walker side, the decoder's construction
// semantics changed - re-verify the whole corpus above.
func TestCheckDuplicateMappingKeys_AliasedAnchorDivergence(t *testing.T) {
	yml := "a: &x {k: 1, k: 2}\nb: *x\nc: *x\n"
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yml), &node))

	var decoded map[string]interface{}
	decodeErr := node.Decode(&decoded)
	walkErr := checkDuplicateMappingKeys(&node)

	require.Error(t, decodeErr)
	require.Error(t, walkErr)

	dupLine := `line 1: mapping key "k" already defined at line 1`
	// decoder: one report per construction visit (anchor + two aliases).
	assert.Equal(t, 3, strings.Count(decodeErr.Error(), dupLine), "decoder reports per visit")
	// walker: one report per definition.
	assert.Equal(t, 1, strings.Count(walkErr.Error(), dupLine), "walker reports once")
}

// TestCheckDuplicateMappingKeys_OffendingMappingNotDescended pins the decoder-matching
// behavior that children of a mapping with duplicate keys are not walked: the decoder
// stops constructing that mapping, so nested duplicates below it never surface.
func TestCheckDuplicateMappingKeys_OffendingMappingNotDescended(t *testing.T) {
	yml := "a: 1\na:\n  x: 1\n  x: 2\n"
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yml), &node))

	var decoded map[string]interface{}
	decodeErr := node.Decode(&decoded)
	walkErr := checkDuplicateMappingKeys(&node)

	require.Error(t, decodeErr)
	require.Error(t, walkErr)
	assert.Equal(t, decodeErr.Error(), walkErr.Error())
}
