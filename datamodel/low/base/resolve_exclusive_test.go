// Copyright 2022-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"testing"

	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

// resolveExclusive decides how to read exclusiveMinimum/exclusiveMaximum, which mean different
// things per version: 3.0 treats the keyword as a boolean modifier on minimum/maximum, 3.1
// treats it as the bound itself. These cases exercise the paths no document fixture reaches,
// because a real parser only ever produces 2.0, 3.0 or 3.1 as a version.
func TestResolveExclusive_KnownVersions(t *testing.T) {
	scalar := func(tag, value string) *yaml.Node {
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: value}
	}

	t.Run("nil value node yields nothing", func(t *testing.T) {
		assert.Nil(t, resolveExclusive(3.1, true, nil))
	})

	t.Run("3.1 reads the value as the bound", func(t *testing.T) {
		got := resolveExclusive(3.1, true, scalar("!!int", "5"))
		require.NotNil(t, got)
		assert.Equal(t, 1, got.N)
		assert.Equal(t, float64(5), got.B)
	})

	t.Run("3.0 reads the value as a boolean modifier", func(t *testing.T) {
		got := resolveExclusive(3.0, true, scalar("!!bool", "true"))
		require.NotNil(t, got)
		assert.Equal(t, 0, got.N)
		assert.True(t, got.A)
	})

	// A value written in the other version's form is deliberately coerced rather than rejected:
	// the parse error is discarded, so a 3.0 boolean under 3.1 becomes the bound 0.
	t.Run("3.1 coerces a boolean to a zero bound", func(t *testing.T) {
		got := resolveExclusive(3.1, true, scalar("!!bool", "true"))
		require.NotNil(t, got)
		assert.Equal(t, 1, got.N)
		assert.Equal(t, float64(0), got.B)
	})

	t.Run("3.0 coerces a number to false", func(t *testing.T) {
		got := resolveExclusive(3.0, true, scalar("!!int", "5"))
		require.NotNil(t, got)
		assert.Equal(t, 0, got.N)
		assert.False(t, got.A)
	})

	// A version strictly between 3.0 and 3.1 matches neither form. No parser emits one, but the
	// value arrives from SpecInfo, so the function must not guess.
	t.Run("a version between 3.0 and 3.1 yields nothing", func(t *testing.T) {
		assert.Nil(t, resolveExclusive(3.05, true, scalar("!!int", "5")))
	})
}

// With no version to coerce towards, the shape of the value decides: a boolean can only be the
// 3.0 form, a number can only be the 3.1 form, and anything else is left absent rather than
// recorded as a bound of zero the author never wrote.
func TestResolveExclusive_UnknownVersion(t *testing.T) {
	scalar := func(tag, value string) *yaml.Node {
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: value}
	}

	t.Run("boolean is read as the 3.0 form", func(t *testing.T) {
		got := resolveExclusive(0, false, scalar("!!bool", "true"))
		require.NotNil(t, got)
		assert.Equal(t, 0, got.N)
		assert.True(t, got.A)
	})

	t.Run("number is read as the 3.1 form", func(t *testing.T) {
		got := resolveExclusive(0, false, scalar("!!float", "2.5"))
		require.NotNil(t, got)
		assert.Equal(t, 1, got.N)
		assert.Equal(t, 2.5, got.B)
	})

	// Tagged as a boolean but not parseable as one. The YAML 1.2 core schema will not produce
	// this, but a hand-built or foreign-tagged node can, and it must not be recorded.
	t.Run("unparseable boolean yields nothing", func(t *testing.T) {
		assert.Nil(t, resolveExclusive(0, false, scalar("!!bool", "yes")))
	})

	t.Run("non-numeric value yields nothing", func(t *testing.T) {
		assert.Nil(t, resolveExclusive(0, false, scalar("!!str", "not-a-number")))
	})
}
