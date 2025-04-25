package v2

import (
	"testing"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	// This will report any leaked goroutines throughout the test suite
	goleak.VerifyTestMain(m)
}
