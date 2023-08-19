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

// returns a filter function that checks if a given error is a circular reference error
// and in case that circular references are allowed or not, it returns false
// in order to skip the error or true in order to keep the error in the wrapped error list.
func circularReferenceErrorFilter(refAllowed bool) func(error) (keep bool) {
	return func(err error) bool {
		if refErr, ok := err.(*resolver.ResolvingError); ok {
			if refAllowed && refErr.CircularReference != nil {
				// allowed & is ref -> skip
				return false
			} else if !refAllowed && refErr.CircularReference != nil {
				// not allowed + is ref -> keep
				return true
			}
			// some other error -> keep
			return true
		}
		return true
	}
}
