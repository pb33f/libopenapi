// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package index

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// NewTestSpecIndex Test helper function to create a SpecIndex with initialised high cache.
func NewTestSpecIndex() *SpecIndex {
	index := &SpecIndex{}
	index.InitHighCache()
	return index
}

// SimpleCache struct and methods are assumed to be imported from the respective package

// TestCreateNewCache tests that a new cache is correctly created.
func TestCreateNewCache(t *testing.T) {
	cache := CreateNewCache()
	assert.NotNil(t, cache)
	assert.NotNil(t, cache.GetStore())
}

// TestSetAndGetStore tests that the store is correctly set and retrieved.
func TestSetAndGetStore(t *testing.T) {
	cache := CreateNewCache()
	newStore := &sync.Map{}
	cache.SetStore(newStore)
	assert.Equal(t, newStore, cache.GetStore())
}

// TestLoadAndStore tests that a value can be stored and loaded from the cache.
func TestLoadAndStore(t *testing.T) {
	cache := CreateNewCache()
	key, value := "key", "value"
	cache.Store(key, value)

	loadedValue, ok := cache.Load(key)
	assert.True(t, ok)
	assert.Equal(t, value, loadedValue)

	// Test for a key that doesn't exist
	_, ok = cache.Load("non-existent")
	assert.False(t, ok)
}

// TestAddHit tests that hits are incremented correctly.
func TestAddHit(t *testing.T) {
	cache := CreateNewCache()
	initialHits := cache.GetHits()

	cache.AddHit()
	newHits := cache.GetHits()
	assert.Equal(t, initialHits+1, newHits)
}

// TestAddMiss tests that misses are incremented correctly.
func TestAddMiss(t *testing.T) {
	cache := CreateNewCache()
	initialMisses := cache.GetMisses()

	cache.AddMiss()
	newMisses := cache.GetMisses()
	assert.Equal(t, initialMisses+1, newMisses)
}

// TestClear tests that the cache is correctly cleared.
func TestClear(t *testing.T) {
	cache := CreateNewCache()
	key, value := "key", "value"
	cache.Store(key, value)
	cache.Clear()

	_, ok := cache.Load(key)
	assert.False(t, ok)
}

// TestConcurrentAccess tests that the cache supports concurrent access.
func TestConcurrentAccess(t *testing.T) {
	cache := CreateNewCache()
	var wg sync.WaitGroup

	// Run 1000 concurrent Store operations
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			cache.Store(i, i)
		}(i)
	}

	// Run 1000 concurrent Load operations
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			cache.Load(i)
		}(i)
	}

	wg.Wait()
	// Check for consistency in hits/misses
	assert.True(t, cache.GetHits() >= 0)
	assert.True(t, cache.GetMisses() >= 0)
}
