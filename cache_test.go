// Copyright 2022-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package libopenapi

import (
	"testing"
)

func TestClearAllCaches(t *testing.T) {
	// ClearAllCaches should not panic when called on empty caches.
	ClearAllCaches()

	// Call twice to ensure idempotency.
	ClearAllCaches()
}
