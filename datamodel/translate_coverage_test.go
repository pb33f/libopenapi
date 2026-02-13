package datamodel_test

import (
	"errors"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTranslateMapParallel_EmptyMap(t *testing.T) {
	m := orderedmap.New[string, int]()
	var translateCalled atomic.Bool
	var resultCalled atomic.Bool

	err := datamodel.TranslateMapParallel[string, int, string](
		m,
		func(pair orderedmap.Pair[string, int]) (string, error) {
			translateCalled.Store(true)
			return "", nil
		},
		func(value string) error {
			resultCalled.Store(true)
			return nil
		},
	)

	require.NoError(t, err)
	assert.False(t, translateCalled.Load())
	assert.False(t, resultCalled.Load())
}

// TestTranslatePipeline_CancelWhileOutputBlocked targets cancellation branches
// that only trigger when result delivery blocks and workers are back-pressured.
func TestTranslatePipeline_CancelWhileOutputBlocked(t *testing.T) {
	workers := runtime.NumCPU()
	if workers < 2 {
		workers = 2
	}

	for iteration := 0; iteration < 8; iteration++ {
		in := make(chan int, workers*64)
		for i := 0; i < cap(in); i++ {
			in <- i
		}
		close(in)

		// Intentionally unconsumed so the collector blocks on `out <-`.
		out := make(chan string)

		errChan := make(chan error, 1)
		go func() {
			errChan <- datamodel.TranslatePipeline[int, string](in, out, func(value int) (string, error) {
				switch value {
				case 0:
					// The first sequence value must be available so the collector reaches out-send.
					return "first", nil
				case 1:
					// Delay cancellation long enough for workers to saturate resultChan.
					time.Sleep(25 * time.Millisecond)
					return "", errors.New("forced cancellation while blocked")
				default:
					return "filler", nil
				}
			})
		}()

		select {
		case err := <-errChan:
			require.ErrorContains(t, err, "forced cancellation while blocked")
		case <-time.After(3 * time.Second):
			t.Fatalf("iteration %d: timed out waiting for TranslatePipeline to return", iteration)
		}
	}
}

// TestTranslateMapParallel_ContextCancellation specifically targets lines 158-159
// in translate.go which handle context cancellation during job dispatch.
// This test ensures 100% coverage even on single-CPU systems like GitHub runners.
//
// The flaky coverage issue occurs because the select statement at lines 156-160:
//
//	select {
//	case jobChan <- j:
//	case <-ctx.Done():
//	  return
//	}
//
// The ctx.Done() branch (lines 158-159) is only hit when the context is cancelled
// while the goroutine is blocked trying to send to jobChan. This is a race condition
// that doesn't always occur, especially on single-CPU systems.
//
// This test forces the condition by:
// 1. Setting GOMAXPROCS to 1 to limit concurrency
// 2. Creating enough work items to fill the job channel
// 3. Having the first job return an error to trigger context cancellation
// 4. Running multiple iterations to ensure we hit the race condition
func TestTranslateMapParallel_ContextCancellation(t *testing.T) {
	// Force single CPU to make the race condition more predictable
	oldMaxProcs := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(oldMaxProcs)

	// Run the test multiple times to ensure we consistently hit the code path
	// This is necessary because even with our setup, the race condition might
	// not occur on the first try.
	for iteration := 0; iteration < 10; iteration++ {
		m := orderedmap.New[string, int]()
		const itemCount = 100
		for i := 0; i < itemCount; i++ {
			m.Set(string(rune('a'+i)), i)
		}

		var translateStarted atomic.Bool
		var jobsBlocked atomic.Int32

		translateFunc := func(pair orderedmap.Pair[string, int]) (string, error) {
			if translateStarted.CompareAndSwap(false, true) {
				// First job: wait briefly then return error to trigger cancel()
				// This causes context cancellation while other jobs are queuing
				time.Sleep(10 * time.Millisecond)
				return "", errors.New("trigger cancellation")
			}

			// Other jobs: count how many get started
			jobsBlocked.Add(1)
			time.Sleep(100 * time.Millisecond)
			return "should not get here", nil
		}

		resultFunc := func(value string) error {
			// Should not be called because translate returns error immediately
			return nil
		}

		err := datamodel.TranslateMapParallel[string, int, string](m, translateFunc, resultFunc)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "trigger cancellation")

		// Wait for goroutines to clean up
		time.Sleep(20 * time.Millisecond)

		// Verify context cancellation prevented all jobs from running
		// If lines 158-159 are hit, some jobs will be skipped
		assert.Less(t, int(jobsBlocked.Load()), itemCount-1,
			"Iteration %d: Context cancellation should prevent some jobs", iteration)
	}
}
