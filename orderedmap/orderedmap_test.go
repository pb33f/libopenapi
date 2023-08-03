package orderedmap_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderedMap(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		m := orderedmap.New[string, int]()
		assert.Equal(t, m.Len(), 0)
		assert.Nil(t, m.First())
	})

	t.Run("First()", func(t *testing.T) {
		const mapSize = 1000
		m := orderedmap.New[string, int]()
		for i := 0; i < mapSize; i++ {
			m.Set(fmt.Sprintf("foobar_%d", i), i)
		}
		assert.Equal(t, m.Len(), mapSize)

		for i := 0; i < mapSize; i++ {
			assert.Equal(t, i, m.GetOrZero(fmt.Sprintf("foobar_%d", i)))
		}

		var i int
		for pair := m.First(); pair != nil; pair = pair.Next() {
			assert.Equal(t, fmt.Sprintf("foobar_%d", i), pair.Key())
			assert.Equal(t, fmt.Sprintf("foobar_%d", i), *pair.KeyPtr())
			assert.Equal(t, i, pair.Value())
			assert.Equal(t, i, *pair.ValuePtr())
			i++
			require.LessOrEqual(t, i, mapSize)
		}
		assert.Equal(t, mapSize, i)
	})

	t.Run("Get()", func(t *testing.T) {
		const mapSize = 1000
		m := orderedmap.New[string, int]()
		for i := 0; i < mapSize; i++ {
			m.Set(fmt.Sprintf("key%d", i), 1000+i)
		}

		for i := 0; i < mapSize; i++ {
			actual, ok := m.Get(fmt.Sprintf("key%d", i))
			assert.True(t, ok)
			assert.Equal(t, 1000+i, actual)
		}

		_, ok := m.Get("bogus")
		assert.False(t, ok)
	})

	t.Run("GetOrZero()", func(t *testing.T) {
		const mapSize = 1000
		m := orderedmap.New[string, int]()
		for i := 0; i < mapSize; i++ {
			m.Set(fmt.Sprintf("key%d", i), 1000+i)
		}

		for i := 0; i < mapSize; i++ {
			actual := m.GetOrZero(fmt.Sprintf("key%d", i))
			assert.Equal(t, 1000+i, actual)
		}

		assert.Equal(t, 0, m.GetOrZero("bogus"))
	})
}

func TestMap(t *testing.T) {
	t.Run("Len()", func(t *testing.T) {
		const mapSize = 100
		m := orderedmap.New[string, int]()
		for i := 0; i < mapSize; i++ {
			m.Set(fmt.Sprintf("key%d", i), i+1000)
		}

		assert.Equal(t, mapSize, m.Len())
		assert.Equal(t, mapSize, orderedmap.Len(m))

		t.Run("Nil pointer", func(t *testing.T) {
			var m orderedmap.Map[string, int]
			assert.Zero(t, orderedmap.Len(m))
		})
	})

	t.Run("Iterate()", func(t *testing.T) {
		const mapSize = 10

		t.Run("Empty", func(t *testing.T) {
			m := orderedmap.New[string, int]()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			c := orderedmap.Iterate(ctx, m)
			for range c {
				t.Fatal("Expected no data")
			}
			requireClosed(t, c)
		})

		t.Run("Full iteration", func(t *testing.T) {
			m := orderedmap.New[string, int]()
			for i := 0; i < mapSize; i++ {
				m.Set(fmt.Sprintf("key%d", i), i+1000)
			}

			var i int
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			c := orderedmap.Iterate(ctx, m)
			for pair := range c {
				assert.Equal(t, fmt.Sprintf("key%d", i), pair.Key())
				assert.Equal(t, fmt.Sprintf("key%d", i), *pair.KeyPtr())
				assert.Equal(t, i+1000, pair.Value())
				assert.Equal(t, i+1000, *pair.ValuePtr())
				i++
				require.LessOrEqual(t, i, mapSize)
			}
			assert.Equal(t, mapSize, i)
			requireClosed(t, c)
		})

		t.Run("Partial iteration", func(t *testing.T) {
			m := orderedmap.New[string, int]()
			for i := 0; i < mapSize; i++ {
				m.Set(fmt.Sprintf("key%d", i), i+1000)
			}

			var i int
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			c := orderedmap.Iterate(ctx, m)
			for pair := range c {
				assert.Equal(t, fmt.Sprintf("key%d", i), pair.Key())
				assert.Equal(t, fmt.Sprintf("key%d", i), *pair.KeyPtr())
				assert.Equal(t, i+1000, pair.Value())
				assert.Equal(t, i+1000, *pair.ValuePtr())
				i++
				if i >= mapSize/2 {
					break
				}
			}

			cancel()
			time.Sleep(10 * time.Millisecond)
			requireClosed(t, c)
			assert.Equal(t, mapSize/2, i)
		})
	})

	t.Run("For()", func(t *testing.T) {
		const mapSize = 10

		t.Run("Nil pointer", func(t *testing.T) {
			var m orderedmap.Map[string, int]
			err := orderedmap.For(m, func(_ orderedmap.Pair[string, int]) error {
				return errors.New("Expected no data")
			})
			require.NoError(t, err)
		})

		t.Run("Empty", func(t *testing.T) {
			m := orderedmap.New[string, int]()
			err := orderedmap.For(m, func(_ orderedmap.Pair[string, int]) error {
				return errors.New("Expected no data")
			})
			require.NoError(t, err)
		})

		t.Run("Full iteration", func(t *testing.T) {
			m := orderedmap.New[string, int]()
			for i := 0; i < mapSize; i++ {
				m.Set(fmt.Sprintf("key%d", i), i+1000)
			}

			var i int
			err := orderedmap.For(m, func(pair orderedmap.Pair[string, int]) error {
				assert.Equal(t, fmt.Sprintf("key%d", i), pair.Key())
				assert.Equal(t, fmt.Sprintf("key%d", i), *pair.KeyPtr())
				assert.Equal(t, i+1000, pair.Value())
				assert.Equal(t, i+1000, *pair.ValuePtr())
				i++
				require.LessOrEqual(t, i, mapSize)
				return nil
			})
			require.NoError(t, err)
			assert.Equal(t, mapSize, i)
		})

		t.Run("Partial iteration", func(t *testing.T) {
			m := orderedmap.New[string, int]()
			for i := 0; i < mapSize; i++ {
				m.Set(fmt.Sprintf("key%d", i), i+1000)
			}

			var i int
			err := orderedmap.For(m, func(pair orderedmap.Pair[string, int]) error {
				assert.Equal(t, fmt.Sprintf("key%d", i), pair.Key())
				assert.Equal(t, fmt.Sprintf("key%d", i), *pair.KeyPtr())
				assert.Equal(t, i+1000, pair.Value())
				assert.Equal(t, i+1000, *pair.ValuePtr())
				i++
				if i >= mapSize/2 {
					return io.EOF
				}
				return nil
			})
			require.NoError(t, err)
			assert.Equal(t, mapSize/2, i)
		})
	})

	t.Run("TranslateMapParallel()", func(t *testing.T) {
		const mapSize = 1000

		t.Run("Happy path", func(t *testing.T) {
			m := orderedmap.New[string, int]()
			for i := 0; i < mapSize; i++ {
				m.Set(fmt.Sprintf("key%d", i), i+1000)
			}

			var translateCounter int64
			translateFunc := func(pair orderedmap.Pair[string, int]) (string, error) {
				result := fmt.Sprintf("foobar %d", pair.Value())
				atomic.AddInt64(&translateCounter, 1)
				return result, nil
			}
			var resultCounter int
			resultFunc := func(value string) error {
				assert.Equal(t, fmt.Sprintf("foobar %d", resultCounter+1000), value)
				resultCounter++
				return nil
			}
			err := orderedmap.TranslateMapParallel[string, int, string](m, translateFunc, resultFunc)
			require.NoError(t, err)
			assert.Equal(t, int64(mapSize), translateCounter)
			assert.Equal(t, mapSize, resultCounter)
		})

		t.Run("Error in translate", func(t *testing.T) {
			m := orderedmap.New[string, int]()
			for i := 0; i < mapSize; i++ {
				m.Set(fmt.Sprintf("key%d", i), i+1000)
			}

			translateFunc := func(pair orderedmap.Pair[string, int]) (string, error) {
				return "", errors.New("Foobar")
			}
			var resultCounter int
			resultFunc := func(value string) error {
				resultCounter++
				return nil
			}
			err := orderedmap.TranslateMapParallel[string, int, string](m, translateFunc, resultFunc)
			require.ErrorContains(t, err, "Foobar")
			assert.Zero(t, resultCounter)
		})

		t.Run("Error in result", func(t *testing.T) {
			m := orderedmap.New[string, int]()
			for i := 0; i < mapSize; i++ {
				m.Set(fmt.Sprintf("key%d", i), i+1000)
			}

			translateFunc := func(pair orderedmap.Pair[string, int]) (string, error) {
				return "", nil
			}
			var resultCounter int
			resultFunc := func(value string) error {
				resultCounter++
				return errors.New("Foobar")
			}
			err := orderedmap.TranslateMapParallel[string, int, string](m, translateFunc, resultFunc)
			require.ErrorContains(t, err, "Foobar")
			assert.Equal(t, 1, resultCounter)
		})

		t.Run("EOF in translate", func(t *testing.T) {
			m := orderedmap.New[string, int]()
			for i := 0; i < mapSize; i++ {
				m.Set(fmt.Sprintf("key%d", i), i+1000)
			}

			translateFunc := func(pair orderedmap.Pair[string, int]) (string, error) {
				return "", io.EOF
			}
			var resultCounter int
			resultFunc := func(value string) error {
				resultCounter++
				return nil
			}
			err := orderedmap.TranslateMapParallel[string, int, string](m, translateFunc, resultFunc)
			require.NoError(t, err)
			assert.Zero(t, resultCounter)
		})

		t.Run("EOF in result", func(t *testing.T) {
			m := orderedmap.New[string, int]()
			for i := 0; i < mapSize; i++ {
				m.Set(fmt.Sprintf("key%d", i), i+1000)
			}

			translateFunc := func(pair orderedmap.Pair[string, int]) (string, error) {
				return "", nil
			}
			var resultCounter int
			resultFunc := func(value string) error {
				resultCounter++
				return io.EOF
			}
			err := orderedmap.TranslateMapParallel[string, int, string](m, translateFunc, resultFunc)
			require.NoError(t, err)
			assert.Equal(t, 1, resultCounter)
		})
	})
}

func requireClosed[K comparable, V any](t *testing.T, c <-chan orderedmap.Pair[K, V]) {
	select {
	case pair := <-c:
		require.Nil(t, pair, "Expected channel to be closed")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout reading channel; expected channel to be closed")
	}
}
