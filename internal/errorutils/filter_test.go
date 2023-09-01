package errorutils

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type customErr2 struct {
	FieldValue string
}

func (e *customErr2) Error() string {
	return e.FieldValue
}

type customErr struct {
	FieldValue string
}

func (e *customErr) Error() string {
	return e.FieldValue
}

func shallowFilter(err error) bool {
	_, ok := err.(*customErr)
	return ok
}

func custom2Filter(err error) bool {
	_, ok := err.(*customErr2)
	return ok
}

func deepFilter(err error) bool {
	var cerr *customErr
	return errors.As(err, &cerr)
}

func removeAllFilter(err error) bool {
	return false
}

func TestFilter(t *testing.T) {

	errs := []error{
		&customErr{FieldValue: "foo"},
		fmt.Errorf("foo: %w", &customErr{FieldValue: "foo"}),
		errors.New("bar"),
		nil,
	}

	filtered := Filter(errs, shallowFilter)

	require.Len(t, filtered, 1)

	for _, err := range filtered {
		require.NotNil(t, err)
		cerr, ok := err.(*customErr)
		require.True(t, ok, "cannot type assert to customErr")

		require.NotEmpty(t, cerr.FieldValue)
	}

	filtered = Filter(errs, deepFilter)

	require.Len(t, filtered, 2)
}

func TestFilteredNormal(t *testing.T) {
	errs := []error{
		&customErr{FieldValue: "foo"},
		fmt.Errorf("foo: %w", &customErr{FieldValue: "foo"}),
		errors.New("bar"),
		&customErr2{"custom2 error"},
		nil,
	}
	err := Join(errs...)

	filtered := Filtered(err, shallowFilter)

	require.Len(t, ShallowUnwrap(filtered), 1)

	filtered = Filtered(err, deepFilter)
	require.Len(t, ShallowUnwrap(filtered), 2)

	// we define multiple filters that specialize on their own type of error
	// to is one of those filters deems the error as one that we want to keep,
	// the error is then kept in the resulting multi error list.
	filtered = Filtered(err, deepFilter, custom2Filter)
	require.Len(t, ShallowUnwrap(filtered), 3)
}

func TestFilteredNil(t *testing.T) {

	filtered := Filtered(nil, shallowFilter)
	require.Nil(t, filtered)
}

func TestFilteredEmpty(t *testing.T) {
	errs := []error{
		&customErr{FieldValue: "foo"},
		fmt.Errorf("foo: %w", &customErr{FieldValue: "foo"}),
		errors.New("bar"),
		nil,
	}
	err := Join(errs...)

	filtered := Filtered(err, removeAllFilter)
	require.Len(t, ShallowUnwrap(filtered), 0)
}
