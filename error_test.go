// Copyright 2023 Princess B33f Heavy Industries
// SPDX-License-Identifier: MIT

package libopenapi

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMultiError(t *testing.T) {
	err := &MultiError{}
	err.Append(errors.New("error 1"))
	err.Append(errors.New("error 2"))
	err.Append(wrapErr(errors.New("error 3")))
	assert.Equal(t, "[0] error 1\n[1] error 2\n[2] error 3\n", err.Error())
}

func TestMultiError_OrNil(t *testing.T) {
	err := &MultiError{}
	err.Append(errors.New("error 1"))
	err.Append(errors.New("error 2"))

	// Append does not add nil errors
	nilErr := &MultiError{}
	err.Append(wrapErr(nilErr.OrNil()))

	assert.Equal(t, "[0] error 1\n[1] error 2\n", err.Error())
}

func TestMultiError_NilError(t *testing.T) {
	// When nil error added to the list.
	err := &MultiError{errs: []error{
		errors.New("error 1"),
		nil,
		errors.New("error 2"),
	}}

	// Should output as 'nil'
	assert.Equal(t, "[0] error 1\n[1] nil\n[2] error 2\n", err.Error())
}

func ExampleMultiError_Print() {
	err := &MultiError{}
	err.Append(errors.New("error 1"))
	err.Append(errors.New("error 2"))
	err.Append(errors.New("error 3"))

	err.Print()

	// Output:
	// [0] error 1
	// [1] error 2
	// [2] error 3
}
