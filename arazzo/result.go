// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"time"
)

// WorkflowResult represents the result of executing a single workflow.
type WorkflowResult struct {
	WorkflowId string
	Success    bool
	Outputs    map[string]any
	Steps      []*StepResult
	Error      error
	Duration   time.Duration
}

// StepResult represents the result of executing a single step.
type StepResult struct {
	StepId     string
	Success    bool
	StatusCode int
	Outputs    map[string]any
	Error      error
	Duration   time.Duration
	Retries    int
}

// RunResult represents the result of executing all workflows.
type RunResult struct {
	Workflows []*WorkflowResult
	Success   bool
	Duration  time.Duration
}
