// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"hash/maphash"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

// hashOf returns the hash of whatever write puts into a hasher, so helper tests can compare two writes.
func hashOf(write func(h *maphash.Hash)) uint64 {
	return low.WithHasher(func(h *maphash.Hash) uint64 {
		write(h)
		return h.Sum64()
	})
}

func TestHashSchemaDependentRequired(t *testing.T) {
	build := func(entries ...any) *orderedmap.Map[low.KeyReference[string], low.ValueReference[[]string]] {
		m := orderedmap.New[low.KeyReference[string], low.ValueReference[[]string]]()
		for i := 0; i < len(entries); i += 2 {
			m.Set(low.KeyReference[string]{Value: entries[i].(string)},
				low.ValueReference[[]string]{Value: entries[i+1].([]string)})
		}
		return m
	}
	hashDependentRequired := func(m *orderedmap.Map[low.KeyReference[string], low.ValueReference[[]string]]) uint64 {
		return hashOf(func(h *maphash.Hash) { hashSchemaDependentRequired(h, "dependentRequired", m) })
	}

	nothing := hashOf(func(h *maphash.Hash) {})
	assert.Equal(t, nothing, hashDependentRequired(nil))
	assert.Equal(t, nothing, hashDependentRequired(build()))

	values := hashDependentRequired(build("omega", []string{"z", "a"}, "alpha", []string{"x"}))
	assert.Equal(t, values, hashDependentRequired(build("alpha", []string{"x"}, "omega", []string{"z", "a"})),
		"declaring the keys in another order should not count")
	assert.NotEqual(t, values, hashDependentRequired(build("omega", []string{"z"}, "alpha", []string{"x", "a"})),
		"moving a value to another key should count")
	assert.NotEqual(t, values, hashDependentRequired(build("omega", []string{"za"}, "alpha", []string{"x"})),
		"joining two values should count")
}

func TestHashSchemaList(t *testing.T) {
	list := func(label string, sorted bool, values ...string) uint64 {
		return hashOf(func(h *maphash.Hash) { hashSchemaList(h, label, values, sorted) })
	}

	assert.Equal(t, hashOf(func(h *maphash.Hash) {}), list("type", true))

	assert.Equal(t, list("type", true, "zeta", "alpha"), list("type", true, "alpha", "zeta"))
	assert.NotEqual(t, list("prefixItems", false, "zeta", "alpha"), list("prefixItems", false, "alpha", "zeta"))
	assert.NotEqual(t, list("oneOf", true, "a", "b"), list("allOf", true, "a", "b"))
	assert.NotEqual(t, list("required", true, "a|b"), list("required", true, "a", "b"))
	assert.NotEqual(t, list("required", true, "ab"), list("required", true, "a", "b"))
}

func TestHashSchemaScalars(t *testing.T) {
	hashes := []uint64{
		hashOf(func(h *maphash.Hash) { hashSchemaFloat(h, "minimum", 5) }),
		hashOf(func(h *maphash.Hash) { hashSchemaFloat(h, "maximum", 5) }),
		hashOf(func(h *maphash.Hash) { hashSchemaFloat(h, "maximum", 5.5) }),
		hashOf(func(h *maphash.Hash) { hashSchemaInt(h, "maxLength", 5) }),
		hashOf(func(h *maphash.Hash) { hashSchemaFloat(h, "exclusiveMaximum", 1) }),
		hashOf(func(h *maphash.Hash) { hashSchemaBool(h, "exclusiveMaximum", true) }),
		hashOf(func(h *maphash.Hash) { hashSchemaBool(h, "readOnly", true) }),
		hashOf(func(h *maphash.Hash) { hashSchemaBool(h, "writeOnly", true) }),
	}
	for i := range hashes {
		for j := i + 1; j < len(hashes); j++ {
			assert.NotEqual(t, hashes[i], hashes[j], "hashes %d and %d collide", i, j)
		}
	}

	// formatted values are written exactly as low.HashString writes a string.
	assert.Equal(t, hashOf(func(h *maphash.Hash) { low.HashString(h, "minimum", "5.5") }),
		hashOf(func(h *maphash.Hash) { hashSchemaFloat(h, "minimum", 5.5) }))
	assert.Equal(t, hashOf(func(h *maphash.Hash) { low.HashString(h, "maxLength", "5") }),
		hashOf(func(h *maphash.Hash) { hashSchemaInt(h, "maxLength", 5) }))
}

func TestSchemaHashIncludesDefs(t *testing.T) {
	build := func(source string) *Schema {
		t.Helper()
		var root yaml.Node
		require.NoError(t, yaml.Unmarshal([]byte(source), &root))

		var schema Schema
		require.NoError(t, schema.Build(context.Background(), root.Content[0], nil))
		return &schema
	}

	a := build(`type: object
$defs:
  shared:
    type: string`)
	b := build(`type: object
$defs:
  shared:
    type: integer`)

	assert.NotEqual(t, a.Hash(), b.Hash())
}
