// SPDX-FileCopyrightText: Copyright 2026 HNO3Miracle
// SPDX-License-Identifier: MIT

package index

import (
	"os"
	"testing"
)

func requireNetworkTests(t *testing.T) {
	t.Helper()
	if os.Getenv("LIBOPENAPI_RUN_NETWORK_TESTS") == "" {
		t.Skip("set LIBOPENAPI_RUN_NETWORK_TESTS to run tests requiring remote fixtures")
	}
}
