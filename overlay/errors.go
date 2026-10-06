// Copyright 2022-2025 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package overlay

import (
	"errors"
	"fmt"

	highoverlay "github.com/pb33f/libopenapi/datamodel/high/overlay"
)

// Warning represents a non-fatal issue encountered during overlay application.
type Warning struct {
	Action  *highoverlay.Action
	Target  string
	Message string
}

func (w *Warning) String() string {
	return fmt.Sprintf("overlay warning: target '%s': %s", w.Target, w.Message)
}

// OverlayError represents an error that occurred during an overlay application.
type OverlayError struct {
	Action *highoverlay.Action
	Cause  error
}

func (e *OverlayError) Error() string {
	if e.Action != nil {
		return fmt.Sprintf("overlay error at target '%s': %v", e.Action.Target, e.Cause)
	}
	return fmt.Sprintf("overlay error: %v", e.Cause)
}

func (e *OverlayError) Unwrap() error {
	return e.Cause
}

// Sentinel errors for overlay operations.
var (
	// Parsing errors
	ErrInvalidOverlay      = errors.New("invalid overlay document")
	ErrMissingOverlayField = errors.New("missing required 'overlay' field")
	ErrMissingInfo         = errors.New("missing required 'info' field")
	ErrMissingActions      = errors.New("missing required 'actions' field")
	ErrEmptyActions        = errors.New("actions array must contain at least one action")

	ErrUnsupportedVersion     = errors.New("unsupported overlay version; expected 1.<minor> or 1.<minor>.<patch>")
	ErrInvalidInfo            = errors.New("overlay info requires title and version")
	ErrMissingTarget          = errors.New("action requires a target")
	ErrInvalidActionReference = errors.New("invalid reusable action reference")
	ErrInvalidReusableAction  = errors.New("reusable action fields must not contain target or $ref")
	ErrIncompatibleUpdate     = errors.New("incompatible overlay update and target types")

	// JSONPath errors
	ErrInvalidJSONPath = errors.New("invalid JSONPath expression")
	// Deprecated: primitive targets are supported. Incompatible updates return ErrIncompatibleUpdate.
	ErrPrimitiveTarget = errors.New("JSONPath target resolved to primitive/null; must be object or array")

	// Application errors
	ErrNoTargetDocument = errors.New("no target document provided")

	// Copy action errors
	ErrCopySourceNotFound = errors.New("copy source JSONPath matched zero nodes")
	ErrCopySourceMultiple = errors.New("copy source JSONPath must match exactly one node")
	ErrCopyTypeMismatch   = errors.New("copy source is incompatible with target")
)
