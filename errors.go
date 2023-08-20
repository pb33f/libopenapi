package libopenapi

import (
	"errors"

	"github.com/pb33f/libopenapi/resolver"
)

var (
	ErrNoConfiguration   = errors.New("no configuration available")
	ErrVersionMismatch   = errors.New("document version mismatch")
	ErrOpenAPI3Operation = errors.New("this operation is only supported for OpenAPI 3 documents")
)

func defaultErrorFilter(err error) bool {
	return true
}

func isCircularErr(err error) bool {
	if err == nil {
		return false
	}

	var resolvErr *resolver.ResolvingError
	if errors.As(err, &resolvErr) {
		return resolvErr.CircularReference != nil
	}

	return false
}

// returns a filter function that checks if a given error is a circular reference error
// and in case that circular references are allowed or not, it returns false
// in order to skip the error or true in order to keep the error in the wrapped error list.
func circularReferenceErrorFilter(refAllowed bool) func(error) (keep bool) {
	return func(err error) bool {
		if err == nil {
			return false
		}

		if isCircularErr(err) {
			if refAllowed {
				return false
			} else {
				return true
			}
		}

		// keep unknown error
		return true
	}
}
