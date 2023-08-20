package errorutils

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnwrapErrors(t *testing.T) {
	err := Join(errors.New("foo"), errors.New("bar"))
	require.NotEmpty(t, err.Error())

	errs := ShallowUnwrap(err)
	require.Len(t, errs, 2)
}

func TestUnwrapError(t *testing.T) {
	err := fmt.Errorf("foo: %w", errors.New("bar"))

	errs := ShallowUnwrap(err)
	require.Len(t, errs, 1)
}

func TestUnwrapHierarchyError(t *testing.T) {
	err1 := errors.New("bar")
	err2 := fmt.Errorf("foo: %w", err1)

	err3 := errors.New("fo")
	err4 := fmt.Errorf("barr: %w", err3)

	err := Join(Join(nil, err2), Join(nil, err4, nil))

	errs := ShallowUnwrap(err)
	require.Len(t, errs, 2)
}

func TestJoinNils(t *testing.T) {
	err := Join(nil, nil)
	require.Nil(t, err)
}

func TestDeepMultiErrorUnwrapNil(t *testing.T) {
	require.Nil(t, deepUnwrapMultiError(nil))
}
