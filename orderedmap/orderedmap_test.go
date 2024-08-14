package orderedmap_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pb33f/libopenapi/datamodel"
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
			var m *orderedmap.Map[string, int]
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
			err := datamodel.TranslateMapParallel[string, int, string](m, translateFunc, resultFunc)
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
			err := datamodel.TranslateMapParallel[string, int, string](m, translateFunc, resultFunc)
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
			err := datamodel.TranslateMapParallel[string, int, string](m, translateFunc, resultFunc)
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
			err := datamodel.TranslateMapParallel[string, int, string](m, translateFunc, resultFunc)
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
			err := datamodel.TranslateMapParallel[string, int, string](m, translateFunc, resultFunc)
			require.NoError(t, err)
			assert.Equal(t, 1, resultCounter)
		})
	})
}

func TestFirst(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		pair := orderedmap.First[string, int](nil)
		require.Nil(t, pair)
	})

	t.Run("Single item", func(t *testing.T) {
		m := orderedmap.New[string, int]()
		m.Set("key", 1)

		var count int
		for pair := orderedmap.First(m); pair != nil; pair = pair.Next() {
			count++
		}
		assert.Equal(t, 1, count)
	})

	t.Run("Many items", func(t *testing.T) {
		const mapSize = 100
		m := orderedmap.New[string, int]()
		for i := 0; i < mapSize; i++ {
			m.Set(fmt.Sprintf("key%d", i), i+1000)
		}

		var count int
		for pair := orderedmap.First(m); pair != nil; pair = pair.Next() {
			count++
		}
		assert.Equal(t, mapSize, count)
	})
}

func TestLen(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		m := (*orderedmap.Map[string, int])(nil)
		require.Zero(t, orderedmap.Len(m))
	})

	t.Run("Single item", func(t *testing.T) {
		m := orderedmap.New[string, int]()
		m.Set("key", 1)

		assert.Equal(t, 1, orderedmap.Len(m))
	})

	t.Run("Many items", func(t *testing.T) {
		const mapSize = 100
		m := orderedmap.New[string, int]()
		for i := 0; i < mapSize; i++ {
			m.Set(fmt.Sprintf("key%d", i), i+1000)
		}

		assert.Equal(t, mapSize, orderedmap.Len(m))
	})
}

func TestFromPairs(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		m := orderedmap.FromPairs[string, int]()
		require.NotNil(t, m)
		assert.Zero(t, m.Len())
	})

	t.Run("Single item", func(t *testing.T) {
		m := orderedmap.FromPairs(
			orderedmap.NewPair[string, int]("key", 1),
		)
		require.NotNil(t, m)
		assert.Equal(t, 1, m.Len())
		pair := m.First()
		assert.Equal(t, "key", pair.Key())
		assert.Equal(t, 1, pair.Value())
		assert.Nil(t, pair.Next())
	})

	t.Run("Many items", func(t *testing.T) {
		const mapSize = 100
		var pairs []orderedmap.Pair[string, int]
		for i := 0; i < mapSize; i++ {
			key := fmt.Sprintf("key%d", i)
			pairs = append(pairs, orderedmap.NewPair[string, int](key, i+1000))
		}

		m := orderedmap.FromPairs(pairs...)
		require.NotNil(t, m)
		assert.Equal(t, mapSize, m.Len())

		var count int
		for pair := m.First(); pair != nil; pair = pair.Next() {
			expectedKey := fmt.Sprintf("key%d", count)
			assert.Equal(t, expectedKey, pair.Key())
			assert.Equal(t, count+1000, pair.Value())
			count++
			require.LessOrEqual(t, count, mapSize)
		}
		assert.Equal(t, mapSize, count)
	})
}

func TestIterators(t *testing.T) {
	om := orderedmap.New[int, any]()
	om.Set(1, "bar")
	om.Set(2, 28)
	om.Set(3, 100)
	om.Set(4, "baz")
	om.Set(5, "28")
	om.Set(6, "100")
	om.Set(7, "baz")
	om.Set(8, "baz")

	expectedKeys := []int{1, 2, 3, 4, 5, 6, 7, 8}
	expectedKeysFromNewest := []int{8, 7, 6, 5, 4, 3, 2, 1}
	expectedValues := []any{"bar", 28, 100, "baz", "28", "100", "baz", "baz"}
	expectedValuesFromNewest := []any{"baz", "baz", "100", "28", "baz", 100, 28, "bar"}

	var keys []int
	var values []any

	for k, v := range om.FromOldest() {
		keys = append(keys, k)
		values = append(values, v)
	}

	assert.Equal(t, expectedKeys, keys)
	assert.Equal(t, expectedValues, values)

	keys, values = []int{}, []any{}

	for k, v := range om.FromNewest() {
		keys = append(keys, k)
		values = append(values, v)
	}

	assert.Equal(t, expectedKeysFromNewest, keys)
	assert.Equal(t, expectedValuesFromNewest, values)

	keys = []int{}

	for k := range om.KeysFromOldest() {
		keys = append(keys, k)
	}

	assert.Equal(t, expectedKeys, keys)

	keys = []int{}

	for k := range om.KeysFromNewest() {
		keys = append(keys, k)
	}

	assert.Equal(t, expectedKeysFromNewest, keys)

	values = []any{}

	for v := range om.ValuesFromOldest() {
		values = append(values, v)
	}

	assert.Equal(t, expectedValues, values)

	values = []any{}

	for v := range om.ValuesFromNewest() {
		values = append(values, v)
	}

	assert.Equal(t, expectedValuesFromNewest, values)
}

func TestIteratorsFrom(t *testing.T) {
	om := orderedmap.New[int, any]()
	om.Set(1, "bar")
	om.Set(2, 28)
	om.Set(3, 100)
	om.Set(4, "baz")
	om.Set(5, "28")
	om.Set(6, "100")
	om.Set(7, "baz")
	om.Set(8, "baz")

	om2 := orderedmap.From(om.FromOldest())

	expectedKeys := []int{1, 2, 3, 4, 5, 6, 7, 8}
	expectedValues := []any{"bar", 28, 100, "baz", "28", "100", "baz", "baz"}

	var keys []int
	var values []any

	for k, v := range om2.FromOldest() {
		keys = append(keys, k)
		values = append(values, v)
	}

	assert.Equal(t, expectedKeys, keys)
	assert.Equal(t, expectedValues, values)

	expectedKeysFromNewest := []int{8, 7, 6, 5, 4, 3, 2, 1}
	expectedValuesFromNewest := []any{"baz", "baz", "100", "28", "baz", 100, 28, "bar"}

	om2 = orderedmap.From(om.FromNewest())

	keys = []int{}
	values = []any{}

	for k, v := range om2.FromOldest() {
		keys = append(keys, k)
		values = append(values, v)
	}

	assert.Equal(t, expectedKeysFromNewest, keys)
	assert.Equal(t, expectedValuesFromNewest, values)
}

func requireClosed[K comparable, V any](t *testing.T, c <-chan orderedmap.Pair[K, V]) {
	select {
	case pair := <-c:
		require.Nil(t, pair, "Expected channel to be closed")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout reading channel; expected channel to be closed")
	}
}
