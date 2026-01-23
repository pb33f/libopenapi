// Copyright 2022-2025 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package low

import (
	"hash/maphash"
	"testing"

	"github.com/stretchr/testify/assert"
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
