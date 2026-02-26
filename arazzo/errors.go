// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"errors"
	"fmt"
	"strings"
)

// Document errors
var (
	ErrInvalidArazzo             = errors.New("invalid arazzo document")
	ErrMissingArazzoField        = errors.New("missing required 'arazzo' field")
	ErrMissingInfo               = errors.New("missing required 'info' field")
	ErrMissingSourceDescriptions = errors.New("missing required 'sourceDescriptions' field")
	ErrEmptySourceDescriptions   = errors.New("sourceDescriptions must have at least one entry")
	ErrMissingWorkflows          = errors.New("missing required 'workflows' field")
	ErrEmptyWorkflows            = errors.New("workflows must have at least one entry")
)

// Workflow errors
var (
	ErrMissingWorkflowId   = errors.New("missing required 'workflowId'")
	ErrMissingSteps        = errors.New("missing required 'steps'")
	ErrEmptySteps          = errors.New("steps must have at least one entry")
	ErrDuplicateWorkflowId = errors.New("duplicate workflowId")
)

// Step errors
var (
	ErrMissingStepId         = errors.New("missing required 'stepId'")
	ErrDuplicateStepId       = errors.New("duplicate stepId within workflow")
	ErrStepMutualExclusion   = errors.New("step must have exactly one of operationId, operationPath, or workflowId")
	ErrExecutorNotConfigured = errors.New("executor is not configured")
)

// Parameter errors
var (
	ErrMissingParameterName  = errors.New("missing required 'name'")
	ErrMissingParameterIn    = errors.New("missing required 'in' for operation parameter")
	ErrInvalidParameterIn    = errors.New("'in' must be path, query, header, or cookie")
	ErrMissingParameterValue = errors.New("missing required 'value'")
)

// Action errors
var (
	ErrMissingActionName     = errors.New("missing required 'name'")
	ErrMissingActionType     = errors.New("missing required 'type'")
	ErrInvalidSuccessType    = errors.New("success action type must be 'end' or 'goto'")
	ErrInvalidFailureType    = errors.New("failure action type must be 'end', 'retry', or 'goto'")
	ErrActionMutualExclusion = errors.New("action cannot have both workflowId and stepId")
	ErrGotoRequiresTarget    = errors.New("goto action requires workflowId or stepId")
	ErrStepIdNotInWorkflow   = errors.New("stepId must reference a step in the current workflow")
)

// Criterion errors
var (
	ErrMissingCondition = errors.New("missing required 'condition'")
)

// Expression errors
var (
	ErrInvalidExpression       = errors.New("invalid runtime expression")
	ErrUnknownExpressionPrefix = errors.New("unknown expression prefix")
)

// Reference errors
var (
	ErrUnresolvedWorkflowRef  = errors.New("workflowId references unknown workflow")
	ErrUnresolvedSourceDesc   = errors.New("sourceDescription reference not found")
	ErrUnresolvedOperationRef = errors.New("operation reference not found")
	ErrOperationSourceMapping = errors.New("operation source mapping failed")
	ErrUnresolvedComponent    = errors.New("component reference not found")
	ErrCircularDependency     = errors.New("circular workflow dependency detected")
)

// Source description errors
var (
	ErrSourceDescLoadFailed = errors.New("failed to load source description")
)

// ValidationError represents a structured validation error with source location.
type ValidationError struct {
	Path   string // e.g. "workflows[0].steps[2].parameters[1]"
	Line   int
	Column int
	Cause  error
}

func (e *ValidationError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("%s (line %d, col %d): %s", e.Path, e.Line, e.Column, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Path, e.Cause)
}

func (e *ValidationError) Unwrap() error {
	return e.Cause
}

// StepFailureError represents a step execution failure with structured context.
type StepFailureError struct {
	StepId         string
	CriterionIndex int // -1 if not criterion-related
	Message        string
	Cause          error
}

func (e *StepFailureError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("step %q failed: %s", e.StepId, e.Cause)
	}
	if e.CriterionIndex >= 0 {
		return fmt.Sprintf("step %q: successCriteria[%d] %s", e.StepId, e.CriterionIndex, e.Message)
	}
	return fmt.Sprintf("step %q failed", e.StepId)
}

func (e *StepFailureError) Unwrap() error {
	return e.Cause
}

// Warning represents a non-fatal validation issue.
type Warning struct {
	Path    string
	Line    int
	Column  int
	Message string
}

func (w *Warning) String() string {
	if w.Line > 0 {
		return fmt.Sprintf("%s (line %d, col %d): %s", w.Path, w.Line, w.Column, w.Message)
	}
	return fmt.Sprintf("%s: %s", w.Path, w.Message)
}

// ValidationResult holds all validation errors and warnings.
type ValidationResult struct {
	Errors   []*ValidationError
	Warnings []*Warning
}

// HasErrors returns true if there are any validation errors.
func (r *ValidationResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// HasWarnings returns true if there are any validation warnings.
func (r *ValidationResult) HasWarnings() bool {
	return len(r.Warnings) > 0
}

// Error implements the error interface, returning all errors as a combined string.
func (r *ValidationResult) Error() string {
	if !r.HasErrors() {
		return ""
	}
	msgs := make([]string, 0, len(r.Errors))
	for _, e := range r.Errors {
		msgs = append(msgs, e.Error())
	}
	return strings.Join(msgs, "; ")
}

// Unwrap returns the individual validation errors for use with errors.Is/As (Go 1.20+).
func (r *ValidationResult) Unwrap() []error {
	if len(r.Errors) == 0 {
		return nil
	}
	errs := make([]error, len(r.Errors))
	for i, ve := range r.Errors {
		errs[i] = ve
	}
	return errs
}
