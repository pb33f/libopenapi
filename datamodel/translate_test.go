package datamodel_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTranslateSliceParallel(t *testing.T) {
	testCases := []struct {
		MapSize int
	}{
		{MapSize: 1},
		{MapSize: 10},
		{MapSize: 100},
		{MapSize: 100_000},
	}

	for _, testCase := range testCases {
		mapSize := testCase.MapSize
		t.Run(fmt.Sprintf("Size %d", mapSize), func(t *testing.T) {
			t.Run("Happy path", func(t *testing.T) {
				var sl []int
				for i := 0; i < mapSize; i++ {
					sl = append(sl, i)
				}

				var translateCounter int64
				translateFunc := func(_, value int) (string, error) {
					result := fmt.Sprintf("foobar %d", value)
					atomic.AddInt64(&translateCounter, 1)
					return result, nil
				}
				var resultCounter int
				resultFunc := func(value string) error {
					assert.Equal(t, fmt.Sprintf("foobar %d", resultCounter), value)
					resultCounter++
					return nil
				}
				err := datamodel.TranslateSliceParallel[int, string](sl, translateFunc, resultFunc)
				require.NoError(t, err)
				assert.Equal(t, int64(mapSize), translateCounter)
				assert.Equal(t, mapSize, resultCounter)
			})

			t.Run("nil", func(t *testing.T) {
				var sl []int
				var translateCounter int64
				translateFunc := func(_, value int) (string, error) {
					atomic.AddInt64(&translateCounter, 1)
					return "", nil
				}
				var resultCounter int
				resultFunc := func(value string) error {
					resultCounter++
					return nil
				}
				err := datamodel.TranslateSliceParallel[int, string](sl, translateFunc, resultFunc)
				require.NoError(t, err)
				assert.Zero(t, translateCounter)
				assert.Zero(t, resultCounter)
			})

			t.Run("Error in translate", func(t *testing.T) {
				var sl []int
				for i := 0; i < mapSize; i++ {
					sl = append(sl, i)
				}

				var translateCounter int64
				translateFunc := func(_, _ int) (string, error) {
					atomic.AddInt64(&translateCounter, 1)
					return "", errors.New("Foobar")
				}
				var resultCounter int
				resultFunc := func(_ string) error {
					resultCounter++
					return nil
				}
				err := datamodel.TranslateSliceParallel[int, string](sl, translateFunc, resultFunc)
				require.ErrorContains(t, err, "Foobar")
				assert.Zero(t, resultCounter)
			})

			t.Run("Error in result", func(t *testing.T) {
				var sl []int
				for i := 0; i < mapSize; i++ {
					sl = append(sl, i)
				}

				translateFunc := func(_, value int) (string, error) {
					return "foobar", nil
				}
				var resultCounter int
				resultFunc := func(_ string) error {
					resultCounter++
					return errors.New("Foobar")
				}
				err := datamodel.TranslateSliceParallel[int, string](sl, translateFunc, resultFunc)
				require.ErrorContains(t, err, "Foobar")
			})

			t.Run("EOF in translate", func(t *testing.T) {
				var sl []int
				for i := 0; i < mapSize; i++ {
					sl = append(sl, i)
				}

				var translateCounter int64
				translateFunc := func(_, _ int) (string, error) {
					atomic.AddInt64(&translateCounter, 1)
					return "", io.EOF
				}
				var resultCounter int
				resultFunc := func(_ string) error {
					resultCounter++
					return nil
				}
				err := datamodel.TranslateSliceParallel[int, string](sl, translateFunc, resultFunc)
				require.NoError(t, err)
				assert.Zero(t, resultCounter)
			})

			t.Run("EOF in result", func(t *testing.T) {
				var sl []int
				for i := 0; i < mapSize; i++ {
					sl = append(sl, i)
				}

				translateFunc := func(_, value int) (string, error) {
					return "foobar", nil
				}
				var resultCounter int
				resultFunc := func(_ string) error {
					resultCounter++
					return io.EOF
				}
				err := datamodel.TranslateSliceParallel[int, string](sl, translateFunc, resultFunc)
				require.NoError(t, err)
			})

			t.Run("Continue in translate", func(t *testing.T) {
				var sl []int
				for i := 0; i < mapSize; i++ {
					sl = append(sl, i)
				}

				var translateCounter int64
				translateFunc := func(_, _ int) (string, error) {
					atomic.AddInt64(&translateCounter, 1)
					return "", datamodel.Continue
				}
				var resultCounter int
				resultFunc := func(_ string) error {
					resultCounter++
					return nil
				}
				err := datamodel.TranslateSliceParallel[int, string](sl, translateFunc, resultFunc)
				require.NoError(t, err)
				assert.Equal(t, int64(mapSize), translateCounter)
				assert.Zero(t, resultCounter)
			})
		})
	}
}

func TestTranslateMapParallel(t *testing.T) {
	const mapSize = 1000

	t.Run("Happy path", func(t *testing.T) {
		var expectedResults []string
		m := orderedmap.New[string, int]()
		for i := 0; i < mapSize; i++ {
			m.Set(fmt.Sprintf("key%d", i), i+1000)
			expectedResults = append(expectedResults, fmt.Sprintf("foobar %d", i+1000))
		}

		var translateCounter int64
		translateFunc := func(pair orderedmap.Pair[string, int]) (string, error) {
			result := fmt.Sprintf("foobar %d", pair.Value())
			atomic.AddInt64(&translateCounter, 1)
			return result, nil
		}
		var results []string
		resultFunc := func(value string) error {
			results = append(results, value)
			return nil
		}
		err := datamodel.TranslateMapParallel[string, int, string](m, translateFunc, resultFunc)
		require.NoError(t, err)
		assert.Equal(t, int64(mapSize), translateCounter)
		assert.Equal(t, mapSize, len(results))
		sort.Strings(results)
		assert.Equal(t, expectedResults, results)
	})

	t.Run("nil", func(t *testing.T) {
		var m *orderedmap.Map[string, int]
		var translateCounter int64
		translateFunc := func(pair orderedmap.Pair[string, int]) (string, error) {
			atomic.AddInt64(&translateCounter, 1)
			return "", nil
		}
		var resultCounter int
		resultFunc := func(value string) error {
			resultCounter++
			return nil
		}
		err := datamodel.TranslateMapParallel[string, int, string](m, translateFunc, resultFunc)
		require.NoError(t, err)
		assert.Zero(t, translateCounter)
		assert.Zero(t, resultCounter)
	})

	t.Run("Error in translate", func(t *testing.T) {
		m := orderedmap.New[string, int]()
		for i := 0; i < mapSize; i++ {
			m.Set(fmt.Sprintf("key%d", i), i+1000)
		}

		var translateCounter int64
		translateFunc := func(_ orderedmap.Pair[string, int]) (string, error) {
			atomic.AddInt64(&translateCounter, 1)
			return "", errors.New("Foobar")
		}
		resultFunc := func(_ string) error {
			t.Fatal("Expected no call to resultFunc()")
			return nil
		}
		err := datamodel.TranslateMapParallel[string, int, string](m, translateFunc, resultFunc)
		require.ErrorContains(t, err, "Foobar")
	})

	t.Run("Error in result", func(t *testing.T) {
		m := orderedmap.New[string, int]()
		for i := 0; i < mapSize; i++ {
			m.Set(fmt.Sprintf("key%d", i), i+1000)
		}

		translateFunc := func(_ orderedmap.Pair[string, int]) (string, error) {
			return "", nil
		}
		var resultCounter int
		resultFunc := func(_ string) error {
			resultCounter++
			return errors.New("Foobar")
		}
		err := datamodel.TranslateMapParallel[string, int, string](m, translateFunc, resultFunc)
		require.ErrorContains(t, err, "Foobar")
		assert.Less(t, resultCounter, mapSize)
	})

	t.Run("EOF in translate", func(t *testing.T) {
		m := orderedmap.New[string, int]()
		for i := 0; i < mapSize; i++ {
			m.Set(fmt.Sprintf("key%d", i), i+1000)
		}

		var translateCounter int64
		translateFunc := func(_ orderedmap.Pair[string, int]) (string, error) {
			atomic.AddInt64(&translateCounter, 1)
			return "", io.EOF
		}
		resultFunc := func(_ string) error {
			t.Fatal("Expected no call to resultFunc()")
			return nil
		}
		err := datamodel.TranslateMapParallel[string, int, string](m, translateFunc, resultFunc)
		require.NoError(t, err)
	})

	t.Run("EOF in result", func(t *testing.T) {
		m := orderedmap.New[string, int]()
		for i := 0; i < mapSize; i++ {
			m.Set(fmt.Sprintf("key%d", i), i+1000)
		}

		translateFunc := func(_ orderedmap.Pair[string, int]) (string, error) {
			return "", nil
		}
		var resultCounter int
		resultFunc := func(_ string) error {
			resultCounter++
			return io.EOF
		}
		err := datamodel.TranslateMapParallel[string, int, string](m, translateFunc, resultFunc)
		require.NoError(t, err)
		assert.Less(t, resultCounter, mapSize)
	})
}

func TestTranslatePipeline(t *testing.T) {
	testCases := []struct {
		ItemCount int
	}{
		{ItemCount: 1},
		{ItemCount: 10},
		{ItemCount: 100},
		{ItemCount: 100_000},
	}

	for _, testCase := range testCases {
		itemCount := testCase.ItemCount
		t.Run(fmt.Sprintf("Size %d", itemCount), func(t *testing.T) {
			t.Run("Happy path", func(t *testing.T) {
				var inputErr error
				in := make(chan int)
				out := make(chan string)
				done := make(chan struct{})
				var wg sync.WaitGroup
				wg.Add(2) // input and output goroutines.

				// Send input.
				go func() {
					defer func() {
						close(in)
						wg.Done()
					}()
					for i := 0; i < itemCount; i++ {
						select {
						case in <- i:
						case <-done:
							inputErr = errors.New("exited unexpectedly")
							return
						}
					}
				}()

				// Collect output.
				var resultCounter int
				go func() {
					for {
						result, ok := <-out
						if !ok {
							break
						}
						assert.Equal(t, strconv.Itoa(resultCounter), result)
						resultCounter++
					}
					close(done)
					wg.Done()
				}()

				err := datamodel.TranslatePipeline[int, string](in, out,
					func(value int) (string, error) {
						return strconv.Itoa(value), nil
					},
				)
				wg.Wait()
				require.NoError(t, err)
				require.NoError(t, inputErr)
				assert.Equal(t, itemCount, resultCounter)
			})

			t.Run("Error in translate", func(t *testing.T) {
				in := make(chan int)
				out := make(chan string)
				done := make(chan struct{})
				var wg sync.WaitGroup
				wg.Add(2) // input and output goroutines.

				// Send input.
				go func() {
					for i := 0; i < itemCount; i++ {
						select {
						case in <- i:
						case <-done:
							// Expected to exit after the first translate.
						}
					}
					close(in)
					wg.Done()
				}()

				// Collect output.
				var resultCounter int
				go func() {
					defer func() {
						close(done)
						wg.Done()
					}()
					for {
						_, ok := <-out
						if !ok {
							return
						}
						resultCounter++
					}
				}()

				err := datamodel.TranslatePipeline[int, string](in, out,
					func(value int) (string, error) {
						return "", errors.New("Foobar")
					},
				)
				wg.Wait()
				require.ErrorContains(t, err, "Foobar")
				assert.Zero(t, resultCounter)
			})

			t.Run("Continue in translate", func(t *testing.T) {
				var inputErr error
				in := make(chan int)
				out := make(chan string)
				done := make(chan struct{})
				var wg sync.WaitGroup
				wg.Add(2) // input and output goroutines.

				// Send input.
				go func() {
					defer wg.Done()
					for i := 0; i < itemCount; i++ {
						select {
						case in <- i:
						case <-done:
							inputErr = errors.New("Exited unexpectedly")
						}
					}
					close(in)
				}()

				// Collect output.
				var resultCounter int
				go func() {
					for {
						_, ok := <-out
						if !ok {
							break
						}
						resultCounter++
					}
					close(done)
					wg.Done()
				}()

				err := datamodel.TranslatePipeline[int, string](in, out,
					func(value int) (string, error) {
						return "", datamodel.Continue
					},
				)
				wg.Wait()
				require.NoError(t, err)
				require.NoError(t, inputErr)
				assert.Zero(t, resultCounter)
			})

			// Target error handler that catches when internal context cancels
			// while waiting on input.
			t.Run("Error while waiting on input", func(t *testing.T) {
				in := make(chan int)
				out := make(chan string)
				var wg sync.WaitGroup
				wg.Add(1) // input goroutine

				// Send input.
				go func() {
					in <- 1
					wg.Done()
				}()

				// No need to capture output channel.

				err := datamodel.TranslatePipeline[int, string](in, out,
					func(value int) (string, error) {
						// Returning an error causes TranslatePipline to cancel its internal context.
						return "", errors.New("Foobar")
					},
				)
				wg.Wait()
				require.Error(t, err)
			})

			// Target error handler that catches when internal context cancels
			// while sending a pipelineJobStatus to worker pool channel.
			// This happens when one item returns an error, triggering a
			// context cancel.  Then the second item is aborted by this error
			// handler.
			t.Run("Error while waiting on worker", func(t *testing.T) {
				// this test gets stuck sometimes, so it needs a hard limit.

				ctx, c := context.WithTimeout(context.Background(), 5*time.Second)
				defer c()
				doneChan := make(chan struct{})

				go func(completedChan chan struct{}) {
					const concurrency = 2
					in := make(chan int)
					out := make(chan string)
					done := make(chan struct{})
					var wg sync.WaitGroup
					wg.Add(1) // input goroutine

					// Send input.
					go func() {
						// Fill up worker pool with items.
						for i := 0; i < concurrency; i++ {
							select {
							case in <- i:
							case <-done:
							}
						}
						wg.Done()
					}()

					// No need to capture output channel.

					var itemCount atomic.Int64
					err := datamodel.TranslatePipeline[int, string](in, out,
						func(value int) (string, error) {
							counter := itemCount.Add(1)
							// Cause error on first call.
							if counter == 1 {
								return "", errors.New("Foobar")
							}
							return "", nil
						},
					)
					close(done)
					wg.Wait()
					require.Error(t, err)
					doneChan <- struct{}{}
				}(doneChan)

				select {
				case <-ctx.Done():
					t.Log("error waiting on worker test timed out")
				case <-doneChan:
					// test passed
				}
				time.Sleep(1 * time.Second)
			})
		})
	}
}
