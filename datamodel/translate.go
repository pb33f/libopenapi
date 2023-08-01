package datamodel

import (
	"context"
	"errors"
	"io"
	"runtime"
	"sync"
)

type ActionFunc[T any] func(T) error
type TranslateFunc[IN any, OUT any] func(IN) (OUT, error)
type TranslateSliceFunc[IN any, OUT any] func(int, IN) (OUT, error)
type TranslateMapFunc[K any, V any, OUT any] func(K, V) (OUT, error)

type continueError struct {
	error
}

var Continue = &continueError{error: errors.New("Continue")}

type jobStatus[OUT any] struct {
	done   chan struct{}
	cont   bool
	result OUT
}

type pipelineJobStatus[IN any, OUT any] struct {
	done   chan struct{}
	cont   bool
	eof    bool
	input  IN
	result OUT
}

// TranslateSliceParallel iterates a slice in parallel and calls translate()
// asynchronously.
// translate() may return `datamodel.Continue` to continue iteration.
// translate() or result() may return `io.EOF` to break iteration.
// Results are provided sequentially to result() in stable order from slice.
func TranslateSliceParallel[IN any, OUT any](in []IN, translate TranslateSliceFunc[IN, OUT], result ActionFunc[OUT]) error {
	if in == nil {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	concurrency := runtime.NumCPU()
	jobChan := make(chan *jobStatus[OUT], concurrency)
	var reterr error
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(1) // input goroutine.

	// Fan out translate jobs.
	go func() {
		defer func() {
			close(jobChan)
			wg.Done()
		}()
		for idx, valueIn := range in {
			j := &jobStatus[OUT]{
				done: make(chan struct{}),
			}
			select {
			case jobChan <- j:
			case <-ctx.Done():
				return
			}

			wg.Add(1)
			go func(idx int, valueIn IN) {
				valueOut, err := translate(idx, valueIn)
				if err == Continue {
					j.cont = true
				} else if err != nil {
					mu.Lock()
					if reterr == nil {
						reterr = err
					}
					mu.Unlock()
					cancel()
					wg.Done()
					return
				}
				j.result = valueOut
				close(j.done)
				wg.Done()
			}(idx, valueIn)
		}
	}()

	// Iterate jobChan as jobs complete.
JOBLOOP:
	for j := range jobChan {
		select {
		case <-j.done:
			if j.cont || result == nil {
				break
			}
			err := result(j.result)
			if err != nil {
				cancel()
				wg.Wait()
				if err == io.EOF {
					return nil
				}
				return err
			}
		case <-ctx.Done():
			break JOBLOOP
		}
	}

	wg.Wait()
	if reterr == io.EOF {
		return nil
	}
	return reterr
}

// TranslateMapParallel iterates a map in parallel and calls translate()
// asynchronously.
// translate() or result() may return `io.EOF` to break iteration.
// Results are provided sequentially to result().  Result order is
// nondeterministic.
func TranslateMapParallel[K comparable, V any, OUT any](m map[K]V, translate TranslateMapFunc[K, V, OUT], result ActionFunc[OUT]) error {
	if len(m) == 0 {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	concurrency := runtime.NumCPU()
	resultChan := make(chan OUT, concurrency)
	var reterr error
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Fan out input translation.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for k, v := range m {
			if ctx.Err() != nil {
				return
			}
			wg.Add(1)
			go func(k K, v V) {
				defer wg.Done()
				value, err := translate(k, v)
				if err == Continue {
					return
				}
				if err != nil {
					mu.Lock()
					if reterr == nil {
						reterr = err
					}
					mu.Unlock()
					cancel()
					return
				}
				select {
				case resultChan <- value:
				case <-ctx.Done():
				}
			}(k, v)
		}
	}()

	go func() {
		// Indicate EOF after all translate goroutines finish.
		wg.Wait()
		close(resultChan)
	}()

	// Iterate results.
	for value := range resultChan {
		err := result(value)
		if err != nil {
			cancel()
			wg.Wait()
			reterr = err
			break
		}
	}

	if reterr == io.EOF {
		return nil
	}
	return reterr
}

// TranslatePipeline processes input sequentially through predicate(), sends to
// translate() in parallel, then outputs in stable order.
// translate() may return `datamodel.Continue` to continue iteration.
// Caller must close `in` channel to indicate EOF.
// TranslatePipeline closes `out` channel to indicate EOF.
func TranslatePipeline[IN any, OUT any](in <-chan IN, out chan<- OUT, translate TranslateFunc[IN, OUT]) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	concurrency := runtime.NumCPU()
	workChan := make(chan *pipelineJobStatus[IN, OUT])
	resultChan := make(chan *pipelineJobStatus[IN, OUT])
	var reterr error
	var mu sync.Mutex
	var wg sync.WaitGroup
	defer wg.Wait()
	wg.Add(1) // input goroutine.

	// Launch worker pool.
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case j, ok := <-workChan:
					if !ok {
						return
					}
					result, err := translate(j.input)
					if err == Continue {
						j.cont = true
						close(j.done)
						continue
					}
					if err != nil {
						mu.Lock()
						defer mu.Unlock()
						if reterr == nil {
							reterr = err
						}
						cancel()
						return
					}
					j.result = result
					close(j.done)
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	// Iterate input, send to workers.
	go func() {
		defer func() {
			close(workChan)
			close(resultChan)
			wg.Done()
		}()
		for {
			select {
			case value, ok := <-in:
				if !ok {
					return
				}
				j := &pipelineJobStatus[IN, OUT]{
					done:  make(chan struct{}),
					input: value,
				}
				select {
				case workChan <- j:
				case <-ctx.Done():
					return
				}
				select {
				case resultChan <- j:
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Collect results in stable order, send to output channel.
	defer close(out)
	for j := range resultChan {
		select {
		case <-j.done:
			if j.cont {
				continue
			}
			out <- j.result
		case <-ctx.Done():
			return reterr
		}
	}

	return reterr
}
