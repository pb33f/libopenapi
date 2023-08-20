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
		if err != nil {
			result.errs = append(result.errs, err)
		}
	}
	return &result
}

// Unwrap recursively unwraps errors and flattens them into a slice of errors.
func Unwrap(err error) []error {
	if err == nil {
		return nil
	}
	var result []error
	if unwrap, ok := err.(interface{ Unwrap() []error }); ok {
		// ignore wrapping error - no hierarchy
		for _, e := range unwrap.Unwrap() {
			result = append(result, Unwrap(e)...)
		}
		return result
	} else if unwrap, ok := err.(interface{ Unwrap() error }); ok {
		// add parent error to result
		result = append(result, err)
		result = append(result, Unwrap(unwrap.Unwrap())...)
		return result
	}
	// no unwrapping needed, as it's not wrapped
	result = append(result, err)
	return result
}
