// Ordered map container
// Works like the Golang `map` built-in, but preserves order that key/value
// pairs were added when iterating.

package orderedmap

import (
	"context"
	"io"
	"runtime"
	"sync"

	wk8orderedmap "github.com/wk8/go-ordered-map/v2"
)

type Map[K comparable, V any] interface {
	Lengthiness
	Get(K) (V, bool)
	GetOrZero(K) V
	Set(K, V) (V, bool)
	Delete(K) (V, bool)
	First() Pair[K, V]
}

type Lengthiness interface {
	Len() int
}

type Pair[K comparable, V any] interface {
	Key() K
	KeyPtr() *K
	Value() V
	ValuePtr() *V
	Next() Pair[K, V]
}

type wrapOrderedMap[K comparable, V any] struct {
	*wk8orderedmap.OrderedMap[K, V]
}

type wrapPair[K comparable, V any] struct {
	*wk8orderedmap.Pair[K, V]
}

type ActionFunc[K comparable, V any] func(Pair[K, V]) error
type TranslateFunc[IN any, OUT any] func(IN) (OUT, error)
type ResultFunc[V any] func(V) error

// New creates an ordered map generic object.
func New[K comparable, V any]() Map[K, V] {
	return &wrapOrderedMap[K, V]{
		OrderedMap: wk8orderedmap.New[K, V](),
	}
}

func (o *wrapOrderedMap[K, V]) GetOrZero(k K) V {
	v, ok := o.OrderedMap.Get(k)
	if !ok {
		var zero V
		return zero
	}
	return v
}

func (o *wrapOrderedMap[K, V]) First() Pair[K, V] {
	pair := o.OrderedMap.Oldest()
	if pair == nil {
		return nil
	}
	return &wrapPair[K, V]{
		Pair: pair,
	}
}

// NewPair instantiates a `Pair` object for use with `FromPairs()`.
func NewPair[K comparable, V any](key K, value V) Pair[K, V] {
	return &wrapPair[K, V]{
		Pair: &wk8orderedmap.Pair[K, V]{
			Key: key,
			Value: value,
		},
	}
}

// FromPairs creates an `OrderedMap` from an array of pairs.
// Use `NewPair()` to generate input parameters.
func FromPairs[K comparable, V any](pairs ...Pair[K, V]) Map[K, V] {
	om := New[K, V]()
	for _, pair := range pairs {
		om.Set(pair.Key(), pair.Value())
	}
	return om
}

// IsZero is required to support `omitempty` tag for YAML/JSON marshaling.
func (o *wrapOrderedMap[K, V]) IsZero() bool {
	return o.Len() == 0
}

func (p *wrapPair[K, V]) Next() Pair[K, V] {
	next := p.Pair.Next()
	if next == nil {
		return nil
	}
	return &wrapPair[K, V]{
		Pair: next,
	}
}

func (p *wrapPair[K, V]) Key() K {
	return p.Pair.Key
}

func (p *wrapPair[K, V]) KeyPtr() *K {
	return &p.Pair.Key
}

func (p *wrapPair[K, V]) Value() V {
	return p.Pair.Value
}

func (p *wrapPair[K, V]) ValuePtr() *V {
	return &p.Pair.Value
}

// Len returns the length of a container implementing a `Len()` method.
// Safely returns zero on nil pointer.
func Len(l Lengthiness) int {
	if l == nil {
		return 0
	}
	return l.Len()
}

// ToOrderedMap converts map built-in to OrderedMap.
// Iterate the map in order.
// Safely handles nil pointer.
// Be sure to iterate to end or cancel the context when done to release
// resources.
func Iterate[K comparable, V any](ctx context.Context, m Map[K, V]) <-chan Pair[K, V] {
	c := make(chan Pair[K, V])
	if Len(m) == 0 {
		close(c)
		return c
	}
	go func() {
		defer close(c)
		for pair := m.First(); pair != nil; pair = pair.Next() {
			select {
			case c <- pair:
			case <-ctx.Done():
				return
			}
		}
	}()
	return c
}

// ToOrderedMap converts a `map` to `OrderedMap`.
func ToOrderedMap[K comparable, V any](m map[K]V) Map[K, V] {
	om := New[K, V]()
	for k, v := range m {
		om.Set(k, v)
	}
	return om
}

// First returns map's first pair for iteration.
// Safely handles nil pointer.
func First[K comparable, V any](m Map[K, V]) Pair[K, V] {
	if m == nil {
		return nil
	}
	return m.First()
}

type jobStatus[T any] struct {
	done   chan struct{}
	result T
}

// TranslateMapParallel iterates a `Map` in parallel and calls translate()
// asynchronously.
// translate() or result() may return `io.EOF` to break iteration.
// Safely handles nil pointer.
// Results are provided sequentially to result() in stable order from `Map`.
func TranslateMapParallel[K comparable, V any, RV any](m Map[K, V], translate TranslateFunc[Pair[K, V], RV], result ResultFunc[RV]) error {
	if m == nil {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	concurrency := runtime.NumCPU()
	c := Iterate(ctx, m)
	jobChan := make(chan *jobStatus[RV], concurrency)
	var reterr error
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Fan out translate jobs.
	wg.Add(1)
	go func() {
		defer func() {
			close(jobChan)
			wg.Done()
		}()
		for pair := range c {
			j := &jobStatus[RV]{
				done: make(chan struct{}),
			}
			select {
			case jobChan <- j:
			case <-ctx.Done():
				return
			}

			wg.Add(1)
			go func(pair Pair[K, V]) {
				value, err := translate(pair)
				if err != nil {
					mu.Lock()
					defer func() {
						mu.Unlock()
						wg.Done()
						cancel()
					}()
					if reterr == nil {
						reterr = err
					}
					return
				}
				j.result = value
				close(j.done)
				wg.Done()
			}(pair)
		}
	}()

	// Iterate jobChan as jobs complete.
	defer wg.Wait()
JOBLOOP:
	for j := range jobChan {
		select {
		case <-j.done:
			err := result(j.result)
			if err != nil {
				cancel()
				if err == io.EOF {
					return nil
				}
				return err
			}
		case <-ctx.Done():
			break JOBLOOP
		}
	}

	if reterr == io.EOF {
		return nil
	}
	return reterr
}
