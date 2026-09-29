// Copyright 2022-2025 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package low

import (
	"hash/maphash"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/testify/assert"
)

func TestHashBool_True(t *testing.T) {
	result := WithHasher(func(h *maphash.Hash) uint64 {
		HashBool(h, true)
		return h.Sum64()
	})
	assert.NotZero(t, result)
}

func TestHashBool_False(t *testing.T) {
	result := WithHasher(func(h *maphash.Hash) uint64 {
		HashBool(h, false)
		return h.Sum64()
	})
	assert.NotZero(t, result)
}

func TestHashBool_DifferentValues(t *testing.T) {
	trueHash := WithHasher(func(h *maphash.Hash) uint64 {
		HashBool(h, true)
		return h.Sum64()
	})
	falseHash := WithHasher(func(h *maphash.Hash) uint64 {
		HashBool(h, false)
		return h.Sum64()
	})
	// true and false should produce different hashes
	assert.NotEqual(t, trueHash, falseHash)
}

func TestHashInt64(t *testing.T) {
	result := WithHasher(func(h *maphash.Hash) uint64 {
		HashInt64(h, 12345)
		return h.Sum64()
	})
	assert.NotZero(t, result)
}

func TestHashInt64_Negative(t *testing.T) {
	result := WithHasher(func(h *maphash.Hash) uint64 {
		HashInt64(h, -99999)
		return h.Sum64()
	})
	assert.NotZero(t, result)
}

func TestHashInt64_Zero(t *testing.T) {
	result := WithHasher(func(h *maphash.Hash) uint64 {
		HashInt64(h, 0)
		return h.Sum64()
	})
	assert.NotZero(t, result)
}

func TestHashUint64(t *testing.T) {
	result := WithHasher(func(h *maphash.Hash) uint64 {
		HashUint64(h, 987654321)
		return h.Sum64()
	})
	assert.NotZero(t, result)
}

func TestHashUint64_Zero(t *testing.T) {
	result := WithHasher(func(h *maphash.Hash) uint64 {
		HashUint64(h, 0)
		return h.Sum64()
	})
	assert.NotZero(t, result)
}

func TestHashUint64_MaxValue(t *testing.T) {
	result := WithHasher(func(h *maphash.Hash) uint64 {
		HashUint64(h, ^uint64(0)) // max uint64
		return h.Sum64()
	})
	assert.NotZero(t, result)
}

func TestPutVisitedMap_OversizedDiscard(t *testing.T) {
	// Build a map with >1024 entries so putVisitedMap discards it.
	m := make(map[*yaml.Node]bool, 1100)
	for i := 0; i < 1025; i++ {
		m[&yaml.Node{Value: "n"}] = true
	}
	putVisitedMap(m)

	// The next getVisitedMap should return a fresh/empty map, not the oversized one.
	fresh := getVisitedMap()
	assert.Empty(t, fresh)
	putVisitedMap(fresh)
}

func TestGetPutVisitedMap_Reuse(t *testing.T) {
	m := getVisitedMap()
	assert.Empty(t, m)

	// Populate and return it.
	m[&yaml.Node{Value: "x"}] = true
	putVisitedMap(m)

	// Next retrieval should be cleared.
	m2 := getVisitedMap()
	assert.Empty(t, m2)
	putVisitedMap(m2)
}

func TestClearNodePools(t *testing.T) {
	initial := getVisitedMap()
	initial[&yaml.Node{Value: "old"}] = true
	putVisitedMap(initial)

	// a deprecated no-op: maps are already cleared before they go back to the pool.
	ClearNodePools()

	fresh := getVisitedMap()
	assert.NotNil(t, fresh)
	assert.Empty(t, fresh)
	putVisitedMap(fresh)
}

func hashWith(fn func(h *maphash.Hash)) uint64 {
	return WithHasher(func(h *maphash.Hash) uint64 {
		fn(h)
		return h.Sum64()
	})
}

func TestHashLabel_SeparatesFieldsWithTheSameValue(t *testing.T) {
	minimum := hashWith(func(h *maphash.Hash) {
		HashLabel(h, "minimum")
		HashInt64(h, 5)
	})
	maximum := hashWith(func(h *maphash.Hash) {
		HashLabel(h, "maximum")
		HashInt64(h, 5)
	})
	assert.NotEqual(t, minimum, maximum)
}

func TestHashString(t *testing.T) {
	title := hashWith(func(h *maphash.Hash) { HashString(h, "title", "same") })
	description := hashWith(func(h *maphash.Hash) { HashString(h, "description", "same") })
	assert.NotEqual(t, title, description)
	assert.Equal(t, title, hashWith(func(h *maphash.Hash) { HashString(h, "title", "same") }))

	// the length prefix stops a value holding the separator from passing for two fields.
	joined := hashWith(func(h *maphash.Hash) { HashString(h, "title", "a|description:b") })
	split := hashWith(func(h *maphash.Hash) {
		HashString(h, "title", "a")
		HashString(h, "description", "b")
	})
	assert.NotEqual(t, joined, split)
}

func TestHashMap(t *testing.T) {
	build := func(entries ...string) *orderedmap.Map[KeyReference[string], ValueReference[string]] {
		m := orderedmap.New[KeyReference[string], ValueReference[string]]()
		for i := 0; i < len(entries); i += 2 {
			m.Set(KeyReference[string]{Value: entries[i]}, ValueReference[string]{Value: entries[i+1]})
		}
		return m
	}
	hashMap := func(label string, m *orderedmap.Map[KeyReference[string], ValueReference[string]]) uint64 {
		return hashWith(func(h *maphash.Hash) { HashMap(h, label, m) })
	}

	empty := hashWith(func(h *maphash.Hash) {})
	assert.Equal(t, empty, hashMap("mapping", nil))
	assert.Equal(t, empty, hashMap("mapping", build()))

	ab := hashMap("mapping", build("a", "1", "b", "2"))
	assert.Equal(t, ab, hashMap("mapping", build("b", "2", "a", "1")), "declaration order should not count")
	assert.NotEqual(t, ab, hashMap("mapping", build("a", "1", "c", "2")), "renaming a key should count")
	assert.NotEqual(t, ab, hashMap("mapping", build("a", "2", "b", "1")), "moving a value to another key should count")
	assert.NotEqual(t, ab, hashMap("scopes", build("a", "1", "b", "2")), "the label should count")
}
