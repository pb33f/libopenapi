// Copyright 2023 Princess B33f Heavy Industries
// SPDX-License-Identifier: MIT

package libopenapi

import (
	"errors"
	"fmt"
	"strings"
)

func errorMsg(msg string) *MultiError {
	return &MultiError{errs: []error{errors.New(msg)}}
}

func errorMsgf(msg string, a ...any) *MultiError {
	return &MultiError{errs: []error{fmt.Errorf(msg, a...)}}
}

func wrapErr(err error) error {
	if err == nil {
		return nil
	}
	return &MultiError{errs: []error{err}}
}

func wrapErrs(err []error) error {
	if len(err) == 0 {
		return nil
	}
	return &MultiError{err}
}

type MultiError struct {
	errs []error
}

func (e *MultiError) Append(err error) {
	if err == nil {
		return
	}

	var m *MultiError
	if errors.As(err, &m) {
		e.errs = append(e.errs, m.errs...)
		return
	}
	e.errs = append(e.errs, err)
}

func (e *MultiError) Count() int {
	return len(e.errs)
}

func (e *MultiError) Error() string {
	var b strings.Builder
	for i, err := range e.errs {
		if err == nil {
			b.WriteString(fmt.Sprintf("[%d] nil\n", i))
			continue
		}
		b.WriteString(fmt.Sprintf("[%d] %s\n", i, err.Error()))
	}
	return b.String()
}

func (e *MultiError) Unwrap() []error {
	return e.errs
}

// OrNil returns this instance of *MultiError or nil if there are no errors
// This is useful because returning a &MultiError{} even if it's empty is
// still considered an error.
func (e *MultiError) OrNil() error {
	if len(e.errs) == 0 {
		return nil
	}
	return e
}

func (e *MultiError) Print() {
	for i, err := range e.errs {
		fmt.Printf("[%d] %s\n", i, err.Error())
	}
}
