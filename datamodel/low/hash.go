// Copyright 2022-2025 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package low

import (
	"encoding/binary"
	"hash/maphash"
	"math"
	"sync"
)

// globalHashSeed ensures consistent hashes across all pooled instances.
// Set once at init, deterministic within a process run.
var globalHashSeed maphash.Seed

func init() {
	globalHashSeed = maphash.MakeSeed()
}

// hasherPool pools maphash.Hash instances for reuse
var hasherPool = sync.Pool{
	New: func() any {
		h := &maphash.Hash{}
		h.SetSeed(globalHashSeed)
		return h
	},
}

// WithHasher provides a pooled hasher for the duration of fn.
// The hasher is automatically returned to the pool after fn completes.
// This pattern eliminates forgotten PutHasher() bugs.
func WithHasher(fn func(h *maphash.Hash) uint64) uint64 {
	hasher := hasherPool.Get().(*maphash.Hash)
	hasher.Reset()
	result := fn(hasher)
	hasherPool.Put(hasher)
	return result
}

// HashString writes a string to the hasher (zero allocation).
func HashString(h *maphash.Hash, s string) {
	h.WriteString(s)
}

// HashByte writes a single byte (typically a separator).
func HashByte(h *maphash.Hash, b byte) {
	h.WriteByte(b)
}

// HashBool writes a boolean as a single byte.
func HashBool(h *maphash.Hash, b bool) {
	if b {
		h.WriteByte(1)
	} else {
		h.WriteByte(0)
	}
}

// HashInt64 writes an int64 without allocation using binary encoding.
func HashInt64(h *maphash.Hash, n int64) {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(n))
	h.Write(buf[:])
}

// HashFloat64 writes a float64 using its IEEE 754 bit pattern (zero allocation).
func HashFloat64(h *maphash.Hash, f float64) {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], math.Float64bits(f))
	h.Write(buf[:])
}

// HashUint64 writes another hash value (for composition of nested Hashable objects).
func HashUint64(h *maphash.Hash, v uint64) {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], v)
	h.Write(buf[:])
}

// HASH_PIPE is the separator byte used between hash fields. :)
const HASH_PIPE = '|'
