// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package utils

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUnwrapErrors(t *testing.T) {

	// create an array of errors
	errs := []error{
		errors.New("first error"),
		errors.New("second error"),
		errors.New("third error"),
	}

	// join them  up
	joined := errors.Join(errs...)
	assert.Error(t, joined)

	// unwrap them
	unwrapped := UnwrapErrors(joined)
	assert.Len(t, unwrapped, 3)
}

func TestUnwrapErrors_Empty(t *testing.T) {
	assert.Len(t, UnwrapErrors(nil), 0)
}
