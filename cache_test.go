// Copyright 2022-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package libopenapi

import (
	"os"
	"sync"
	"testing"

	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestClearAllCaches(t *testing.T) {
	// ClearAllCaches should not panic when called on empty caches.
	ClearAllCaches()

	// Call twice to ensure idempotency.
	ClearAllCaches()
}

// ClearAllCaches must be safe to call while other goroutines build and compare documents (run with -race).
func TestClearAllCaches_ConcurrentWithDocumentWork(t *testing.T) {
	original, err := os.ReadFile("test_specs/burgershop.openapi.yaml")
	require.NoError(t, err)
	modified, err := os.ReadFile("test_specs/burgershop.openapi-modified.yaml")
	require.NoError(t, err)

	var wg sync.WaitGroup
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 3; i++ {
				left, _ := NewDocument(original)
				right, _ := NewDocument(modified)
				changes, errs := CompareDocuments(left, right)
				assert.Empty(t, errs)
				assert.NotNil(t, changes)
			}
		}()
	}
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	for {
		select {
		case <-done:
			return
		default:
			ClearAllCaches()
		}
	}
}
