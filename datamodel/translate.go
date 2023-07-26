package datamodel

import (
	"context"
	"io"
	"runtime"
	"sync"
)

type ActionFunc[T any] func(T) error
type TranslateFunc[IN any, OUT any] func(IN) (OUT, error)
type TranslateSliceFunc[IN any, OUT any] func(int, IN) (OUT, error)
type ResultFunc[V any] func(V) error

type continueError struct {
	error
}

var Continue = &continueError{}

// TranslateSliceParallel iterates a slice in parallel and calls translate()
// asynchronously.
// translate() may return `datamodel.Continue` to continue iteration.
// translate() or result() may return `io.EOF` to break iteration.
// Results are provided sequentially to result() in stable order from slice.
func TranslateSliceParallel[IN any, OUT any](in []IN, translate TranslateSliceFunc[IN, OUT], result ActionFunc[OUT]) error {
	if in == nil {
		return nil
	}

	type jobStatus struct {
		done   chan struct{}
		cont   bool
		result OUT
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	concurrency := runtime.NumCPU()
	jobChan := make(chan *jobStatus, concurrency)
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
			j := &jobStatus{
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

// TranslatePipeline processes input sequentially through predicate(), sends to
// translate() in parallel, then outputs in stable order.
// translate() may return `datamodel.Continue` to continue iteration.
// Caller must close `in` channel to indicate EOF.
// TranslatePipeline closes `out` channel to indicate EOF.
func TranslatePipeline[IN any, OUT any](in <-chan IN, out chan<- OUT, translate TranslateFunc[IN, OUT]) error {
	type jobStatus struct {
		done   chan struct{}
		cont   bool
		eof    bool
		input  IN
		result OUT
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	concurrency := runtime.NumCPU()
	workChan := make(chan *jobStatus)
	resultChan := make(chan *jobStatus)
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
				j := &jobStatus{
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
