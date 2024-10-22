// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package index

import "sync"

func (index *SpecIndex) SetCache(sync *sync.Map) {
	index.cache = sync
}

// HighCacheHit increments the counter of high cache hits by one, and returns the current value of hits.
func (index *SpecIndex) HighCacheHit() uint64 {
	index.highModelCacheHits.Add(1)
	return index.highModelCacheHits.Load()
}

// HighCacheMiss increments the counter of high cache misses by one, and returns the current value of misses.
func (index *SpecIndex) HighCacheMiss() uint64 {
	index.highModelCacheMisses.Add(1)
	return index.highModelCacheMisses.Load()
}

// GetHighCacheHits returns the number of hits on the high model cache.
func (index *SpecIndex) GetHighCacheHits() uint64 {
	return index.highModelCacheHits.Load()
}

// GetHighCacheMisses returns the number of misses on the high model cache.
func (index *SpecIndex) GetHighCacheMisses() uint64 {
	return index.highModelCacheMisses.Load()
}

// GetHighCache returns the high model cache for this index.
func (index *SpecIndex) GetHighCache() *sync.Map {
	if index.highModelCache == nil {
		index.highModelCache = &sync.Map{}
	}
	return index.highModelCache
}
