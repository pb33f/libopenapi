package errorutils

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnwrapErrors(t *testing.T) {
	err := Join(errors.New("foo"), errors.New("bar"))

	errs := Unwrap(err)
	require.Len(t, errs, 2)
}

func TestUnwrapError(t *testing.T) {
	err := fmt.Errorf("foo: %w", errors.New("bar"))

	errs := Unwrap(err)
	require.Len(t, errs, 2)
}

func TestUnwrapHierarchyError(t *testing.T) {
	err1 := fmt.Errorf("foo: %w", errors.New("bar"))
	err2 := Join(nil, fmt.Errorf("barr: %w", errors.New("fo")))
	err := Join(err1, err2)

	errs := Unwrap(err)
	require.Len(t, errs, 4)
}

func TestJoinNils(t *testing.T) {
	err := Join(nil, nil)
	require.Nil(t, err)
}
