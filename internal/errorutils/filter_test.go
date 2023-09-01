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

func custom1Filter(err error) (keep bool) {
	if _, ok := err.(*customErr); ok {
		return false
	}
	// do not touch unknown errors
	return true
}

func custom2Filter(err error) bool {
	if _, ok := err.(*customErr2); ok {
		return false
	}
	return true
}

func deepCustom1Filter(err error) bool {
	var cerr *customErr
	return !errors.As(err, &cerr)
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

	filtered := Filter(errs, custom1Filter)
	require.Len(t, filtered, 2)

	for _, err := range filtered {
		require.NotNil(t, err)
		_, ok := err.(*customErr)
		require.False(t, ok, "cannot type assert to customErr")
	}

	// additional filter that removes errors
	filtered = Filter(errs, deepCustom1Filter)

	require.Len(t, filtered, 1)
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

	filtered := Filtered(err, custom1Filter)

	require.Len(t, ShallowUnwrap(filtered), 3)

	filtered = Filtered(err, deepCustom1Filter)
	require.Len(t, ShallowUnwrap(filtered), 2)

	filtered = Filtered(err, deepCustom1Filter, custom2Filter)
	require.Len(t, ShallowUnwrap(filtered), 1)
}

func TestFilteredNil(t *testing.T) {

	filtered := Filtered(nil, custom1Filter)
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
