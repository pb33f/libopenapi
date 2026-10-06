// Copyright 2023-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package orderedmap_test

import (
	"reflect"
	"testing"

	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

// GetKeyType and GetValueType report the reflection types the map was instantiated with,
// which is how reflection-driven callers discover what a Map holds.
func TestMap_GetKeyAndValueType(t *testing.T) {
	m := orderedmap.New[string, int]()

	assert.Equal(t, reflect.TypeOf(new(string)), m.GetKeyType())
	assert.Equal(t, reflect.TypeOf(new(int)), m.GetValueType())

	pointers := orderedmap.New[string, *testValue]()
	assert.Equal(t, reflect.TypeOf(new(*testValue)), pointers.GetValueType())
}

type testValue struct{ name string }

// GetOrZero substitutes the value type's zero value for a missing key, so callers can read
// without a comma-ok dance.
func TestMap_GetOrZero(t *testing.T) {
	m := orderedmap.New[string, int]()
	m.Set("present", 42)

	assert.Equal(t, 42, m.GetOrZero("present"))
	assert.Equal(t, 0, m.GetOrZero("absent"), "a missing key yields the zero value")

	pointers := orderedmap.New[string, *testValue]()
	assert.Nil(t, pointers.GetOrZero("absent"), "the zero value of a pointer type is nil")

	strings := orderedmap.New[string, string]()
	assert.Equal(t, "", strings.GetOrZero("absent"))
}

// IsZero backs the omitempty tag: an empty or nil map must not be rendered.
func TestMap_IsZero(t *testing.T) {
	empty := orderedmap.New[string, int]()
	assert.True(t, empty.IsZero())

	populated := orderedmap.New[string, int]()
	populated.Set("k", 1)
	assert.False(t, populated.IsZero())

	var nilMap *orderedmap.Map[string, int]
	assert.True(t, nilMap.IsZero(), "a nil map is empty for rendering purposes")
}

// Cast recovers a typed Map from an any, returning nil rather than panicking when the value
// is absent or is some other type.
func TestCast(t *testing.T) {
	m := orderedmap.New[string, int]()
	m.Set("k", 1)

	got := orderedmap.Cast[string, int](m)
	require.NotNil(t, got)
	assert.Equal(t, 1, got.GetOrZero("k"))

	assert.Nil(t, orderedmap.Cast[string, int](nil), "nil input yields nil")
	assert.Nil(t, orderedmap.Cast[string, int]("not a map"), "a mismatched type yields nil")
	assert.Nil(t, orderedmap.Cast[string, int](orderedmap.New[string, string]()),
		"a map with different type parameters yields nil")
}

// SortAlpha returns a new map ordered by key, and tolerates a nil input.
func TestSortAlpha(t *testing.T) {
	assert.Nil(t, orderedmap.SortAlpha[string, int](nil))

	m := orderedmap.New[string, int]()
	m.Set("charlie", 3)
	m.Set("alpha", 1)
	m.Set("bravo", 2)

	sorted := orderedmap.SortAlpha(m)
	require.NotNil(t, sorted)

	var keys []string
	for pair := sorted.First(); pair != nil; pair = pair.Next() {
		keys = append(keys, pair.Key())
	}
	assert.Equal(t, []string{"alpha", "bravo", "charlie"}, keys)

	// The original ordering is left alone.
	assert.Equal(t, "charlie", m.First().Key())
}

// First is nil-safe at both the method and package-function level.
func TestFirst_NilSafe(t *testing.T) {
	var nilMap *orderedmap.Map[string, int]
	assert.Nil(t, nilMap.First())
	assert.Nil(t, orderedmap.First(nilMap))

	m := orderedmap.New[string, int]()
	assert.Nil(t, m.First(), "an empty map has no first pair")

	m.Set("k", 1)
	require.NotNil(t, orderedmap.First(m))
	assert.Equal(t, "k", orderedmap.First(m).Key())
}
