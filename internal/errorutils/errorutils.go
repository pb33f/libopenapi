package errorutils

import (
	"fmt"
	"strings"
)

// MultiError is a collection of errors.
// It never contains nil values.
type MultiError struct {
	errs []error
}

func (e *MultiError) Error() string {
	var b strings.Builder
	b.Grow(len(e.errs) * 16)
	for i, err := range e.errs {
		b.WriteString(fmt.Sprintf("[%d] %v\n", i, err))
	}
	return b.String()
}

func (e *MultiError) Unwrap() []error {
	return e.errs
}

// Join should be used at the end of top level functions to join all errors.
func Join(errs ...error) error {
	var result MultiError

	size := 0
	for _, err := range errs {
		if err != nil {
			size++
		}
	}
	if size == 0 {
		return nil
	}

	result.errs = make([]error, 0, size)
	for _, err := range errs {
		if err == nil {
			continue
		}
		// try to keep MultiError flat
		result.errs = append(result.errs, deepUnwrapMultiError(err)...)
	}
	return &result
}

func ShallowUnwrap(err error) []error {
	if err == nil {
		return nil
	}
	unwrap, ok := err.(interface{ Unwrap() []error })
	if !ok {
		return []error{err}
	}

	return unwrap.Unwrap()
}

func deepUnwrapMultiError(err error) []error {
	if err == nil {
		return nil
	}
	var result []error

	if multi, ok := err.(*MultiError); ok {
		for _, e := range multi.Unwrap() {
			result = append(result, deepUnwrapMultiError(e)...)
		}
	} else {
		result = append(result, err)
	}
	return result
}
