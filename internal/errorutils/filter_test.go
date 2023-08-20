package errorutils

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

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
		nil,
	}
	err := Join(errs...)

	filtered := Filtered(err, shallowFilter)

	require.Len(t, ShallowUnwrap(filtered), 1)

	filtered = Filtered(err, deepFilter)
	require.Len(t, ShallowUnwrap(filtered), 2)
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
