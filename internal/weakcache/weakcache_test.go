// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package weakcache

import (
	"runtime"
	"testing"
	"time"
	"unsafe"
	"weak"

	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

type key struct {
	name string
}

func TestCache_LoadStore(t *testing.T) {
	var c Cache[key, string]
	k := &key{name: "a"}

	_, ok := c.Load(k)
	assert.False(t, ok)

	c.Store(k, "one")
	v, ok := c.Load(k)
	assert.True(t, ok)
	assert.Equal(t, "one", v)

	// a different object with equal contents is a different key.
	_, ok = c.Load(&key{name: "a"})
	assert.False(t, ok)

	c.Store(k, "two")
	v, _ = c.Load(k)
	assert.Equal(t, "two", v)
	assert.Equal(t, 1, c.Len())
	runtime.KeepAlive(k)
}

func TestCache_Clear(t *testing.T) {
	var c Cache[key, string]
	k := &key{name: "a"}
	c.Store(k, "one")

	c.Clear()
	_, ok := c.Load(k)
	assert.False(t, ok)

	// storing a live key again replaces the invalidated entry instead of adding one.
	c.Store(k, "two")
	v, ok := c.Load(k)
	assert.True(t, ok)
	assert.Equal(t, "two", v)
	assert.Equal(t, 1, c.Len())
	runtime.KeepAlive(k)
}

func TestCache_DoesNotRetainKeys(t *testing.T) {
	var c Cache[key, string]
	collected := make(chan struct{})

	func() {
		k := &key{name: "short-lived"}
		runtime.AddCleanup(k, func(ch chan struct{}) { close(ch) }, collected)
		c.Store(k, "value")
		require.Equal(t, 1, c.Len())
	}()

	// the cache must not keep the key reachable, and must drop the entry once the key is reclaimed.
	require.Eventually(t, func() bool {
		runtime.GC()
		select {
		case <-collected:
			return c.Len() == 0
		default:
			return false
		}
	}, 5*time.Second, 10*time.Millisecond)
}

// An entry whose weak key no longer points at the object living at its address was left behind by a reclaimed
// object: the new object must miss, and storing replaces the entry.
func TestCache_ReplacesEntryOfReclaimedObject(t *testing.T) {
	var c Cache[key, string]
	k := &key{name: "new"}
	previous := &key{name: "previous"}
	stale := &entry[key, string]{key: weak.Make(previous)}
	stale.value.Store(&stamped[string]{generation: c.generation.Load(), value: "previous"})
	c.entries.Store(uintptr(unsafe.Pointer(k)), stale)

	_, ok := c.Load(k)
	assert.False(t, ok)

	c.Store(k, "new")
	v, ok := c.Load(k)
	assert.True(t, ok)
	assert.Equal(t, "new", v)
	assert.Equal(t, 1, c.Len())
	runtime.KeepAlive(previous)
}
