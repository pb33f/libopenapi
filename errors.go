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

func isCircularErr(err error) bool {
	var resolvErr *resolver.ResolvingError
	if errors.As(err, &resolvErr) {
		return resolvErr.CircularReference != nil
	}

	return false
}

// returns a filter function that checks if a given error is a circular reference error
// and in case that circular references are allowed or not, it returns false
// in order to skip the error or true in order to keep the error in the wrapped error list.
func circularReferenceErrorFilter(forbidden bool) func(error) (keep bool) {
	return func(err error) bool {
		// no nil check needed, as errorutils.Filter already removes nil errors

		if isCircularErr(err) {
			// if forbidded -> keep the error and pass it to the user
			return forbidden
		}

		// keep unknown error
		return true
	}
}
