// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/pb33f/libopenapi/arazzo/expression"
	high "github.com/pb33f/libopenapi/datamodel/high/arazzo"
	"github.com/pb33f/libopenapi/orderedmap"
)

// actionTypeRequest groups the parameters for processActionTypeResult,
// normalizing both success and failure actions into a common structure.
type actionTypeRequest struct {
	actionType     string
	workflowId     string
	stepId         string
	retryAfterSec  float64
	retryLimit     int64
	currentRetries int
}

type stepActionResult struct {
	endWorkflow   bool
	retryCurrent  bool
	retryAfter    time.Duration
	jumpToStepIdx int
}

func (e *Engine) processSuccessActions(
	ctx context.Context,
	step *high.Step,
	wf *high.Workflow,
	exprCtx *expression.Context,
	state *executionState,
	stepIndexByID map[string]int,
) (*stepActionResult, error) {
	action, err := e.selectSuccessAction(step.OnSuccess, wf.SuccessActions, exprCtx)
	if err != nil {
		return nil, err
	}
	if action == nil {
		return &stepActionResult{jumpToStepIdx: -1}, nil
	}
	return e.processActionTypeResult(ctx, &actionTypeRequest{
		actionType: action.Type,
		workflowId: action.WorkflowId,
		stepId:     action.StepId,
	}, exprCtx, state, stepIndexByID)
}

func (e *Engine) processFailureActions(
	ctx context.Context,
	step *high.Step,
	wf *high.Workflow,
	exprCtx *expression.Context,
	state *executionState,
	stepIndexByID map[string]int,
	currentRetries int,
) (*stepActionResult, error) {
	action, err := e.selectFailureAction(step.OnFailure, wf.FailureActions, exprCtx)
	if err != nil {
		return nil, err
	}
	if action == nil {
		return &stepActionResult{jumpToStepIdx: -1}, nil
	}
	var retryAfterSec float64
	if action.RetryAfter != nil {
		retryAfterSec = *action.RetryAfter
	}
	var retryLimit int64
	if action.RetryLimit != nil {
		retryLimit = *action.RetryLimit
	}
	return e.processActionTypeResult(ctx, &actionTypeRequest{
		actionType:     action.Type,
		workflowId:     action.WorkflowId,
		stepId:         action.StepId,
		retryAfterSec:  retryAfterSec,
		retryLimit:     retryLimit,
		currentRetries: currentRetries,
	}, exprCtx, state, stepIndexByID)
}

func (e *Engine) processActionTypeResult(
	ctx context.Context,
	req *actionTypeRequest,
	exprCtx *expression.Context,
	state *executionState,
	stepIndexByID map[string]int,
) (*stepActionResult, error) {
	result := &stepActionResult{jumpToStepIdx: -1}
	switch req.actionType {
	case "end":
		result.endWorkflow = true
	case "goto":
		if req.workflowId != "" {
			wfResult, runErr := e.runWorkflow(ctx, req.workflowId, nil, state)
			if runErr != nil {
				return nil, runErr
			}
			exprCtx.Workflows = buildWorkflowContexts(state.workflowResults)
			if wfResult != nil && !wfResult.Success {
				if wfResult.Error != nil {
					return nil, wfResult.Error
				}
				return nil, fmt.Errorf("workflow %q failed", req.workflowId)
			}
			result.endWorkflow = true
			return result, nil
		}
		if req.stepId != "" {
			idx, ok := stepIndexByID[req.stepId]
			if !ok {
				return nil, fmt.Errorf("%w: %q", ErrStepIdNotInWorkflow, req.stepId)
			}
			result.jumpToStepIdx = idx
		}
	case "retry":
		limit := req.retryLimit
		if limit <= 0 {
			limit = 1
		}
		if int64(req.currentRetries) >= limit {
			return &stepActionResult{jumpToStepIdx: -1}, nil
		}
		result.retryCurrent = true
		if req.retryAfterSec > 0 {
			retryAfter := time.Duration(math.Round(req.retryAfterSec * float64(time.Second)))
			if retryAfter > 0 {
				result.retryAfter = retryAfter
			}
		}
	}
	return result, nil
}

func (e *Engine) selectSuccessAction(stepActions, workflowActions []*high.SuccessAction, exprCtx *expression.Context) (*high.SuccessAction, error) {
	if action, err := e.findMatchingSuccessAction(stepActions, exprCtx); err != nil || action != nil {
		return action, err
	}
	return e.findMatchingSuccessAction(workflowActions, exprCtx)
}

func (e *Engine) selectFailureAction(stepActions, workflowActions []*high.FailureAction, exprCtx *expression.Context) (*high.FailureAction, error) {
	if action, err := e.findMatchingFailureAction(stepActions, exprCtx); err != nil || action != nil {
		return action, err
	}
	return e.findMatchingFailureAction(workflowActions, exprCtx)
}

func (e *Engine) findMatchingSuccessAction(actions []*high.SuccessAction, exprCtx *expression.Context) (*high.SuccessAction, error) {
	return findMatchingAction(actions, e.resolveSuccessAction,
		func(a *high.SuccessAction) []*high.Criterion { return a.Criteria },
		e.evaluateActionCriteria, exprCtx)
}

func (e *Engine) findMatchingFailureAction(actions []*high.FailureAction, exprCtx *expression.Context) (*high.FailureAction, error) {
	return findMatchingAction(actions, e.resolveFailureAction,
		func(a *high.FailureAction) []*high.Criterion { return a.Criteria },
		e.evaluateActionCriteria, exprCtx)
}

// findMatchingAction iterates actions, resolves component references, evaluates criteria,
// and returns the first action whose criteria all pass.
func findMatchingAction[T any](
	actions []T,
	resolve func(T) (T, error),
	getCriteria func(T) []*high.Criterion,
	evalCriteria func([]*high.Criterion, *expression.Context) (bool, error),
	exprCtx *expression.Context,
) (T, error) {
	var zero T
	for _, action := range actions {
		resolved, err := resolve(action)
		if err != nil {
			return zero, err
		}
		matches, err := evalCriteria(getCriteria(resolved), exprCtx)
		if err != nil {
			return zero, err
		}
		if matches {
			return resolved, nil
		}
	}
	return zero, nil
}

func (e *Engine) resolveSuccessAction(action *high.SuccessAction) (*high.SuccessAction, error) {
	if action == nil {
		return nil, nil
	}
	if !action.IsReusable() {
		return action, nil
	}
	if e.document == nil || e.document.Components == nil {
		return nil, fmt.Errorf("%w: %q", ErrUnresolvedComponent, action.Reference)
	}
	return lookupComponent(action.Reference, "$components.successActions.",
		e.document.Components.SuccessActions)
}

func (e *Engine) resolveFailureAction(action *high.FailureAction) (*high.FailureAction, error) {
	if action == nil {
		return nil, nil
	}
	if !action.IsReusable() {
		return action, nil
	}
	if e.document == nil || e.document.Components == nil {
		return nil, fmt.Errorf("%w: %q", ErrUnresolvedComponent, action.Reference)
	}
	return lookupComponent(action.Reference, "$components.failureActions.",
		e.document.Components.FailureActions)
}

// lookupComponent resolves a $components reference against an ordered map.
func lookupComponent[T any](ref, prefix string, componentMap *orderedmap.Map[string, T]) (T, error) {
	var zero T
	if !strings.HasPrefix(ref, prefix) {
		return zero, fmt.Errorf("%w: %q", ErrUnresolvedComponent, ref)
	}
	if componentMap == nil {
		return zero, fmt.Errorf("%w: %q", ErrUnresolvedComponent, ref)
	}
	name := strings.TrimPrefix(ref, prefix)
	resolved, ok := componentMap.Get(name)
	if !ok {
		return zero, fmt.Errorf("%w: %q", ErrUnresolvedComponent, ref)
	}
	return resolved, nil
}

// evaluateActionCriteria evaluates all criteria for an action, using per-engine caches.
func (e *Engine) evaluateActionCriteria(criteria []*high.Criterion, exprCtx *expression.Context) (bool, error) {
	if len(criteria) == 0 {
		return true, nil
	}
	for i, criterion := range criteria {
		ok, err := evaluateCriterionImpl(criterion, exprCtx, e.criterionCaches)
		if err != nil {
			return false, fmt.Errorf("failed to evaluate action criteria[%d]: %w", i, err)
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}
