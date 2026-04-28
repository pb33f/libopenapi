// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"errors"
	"fmt"
)

var (
	ErrNilSchema          = errors.New("nil schema")
	ErrNilType            = errors.New("nil type")
	ErrUnsupportedType    = errors.New("unsupported type")
	ErrUnsupportedMapKey  = errors.New("unsupported map key")
	ErrInvalidPackageName = errors.New("invalid package name")
	// Deprecated: name collisions are reported through DiagnosticCode values.
	ErrNameCollision = errors.New("name collision")
)

func wrapPath(err error, path string) error {
	if path == "" {
		return fmt.Errorf("generator/golang: %w", err)
	}
	return fmt.Errorf("generator/golang: %w at %s", err, path)
}
