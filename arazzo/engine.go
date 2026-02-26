// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/pb33f/libopenapi/arazzo/expression"
	high "github.com/pb33f/libopenapi/datamodel/high/arazzo"
)

const maxWorkflowDepth = 32
const maxStepTransitions = 1024

// Executor defines the interface for executing API calls.
type Executor interface {
	Execute(ctx context.Context, req *ExecutionRequest) (*ExecutionResponse, error)
}

// ExecutionRequest represents a request to execute an API operation.
type ExecutionRequest struct {
	Source        *ResolvedSource
	OperationID   string
	OperationPath string
	Method        string
	Parameters    map[string]any
	RequestBody   any
	ContentType   string
}

// ExecutionResponse represents the response from an API operation execution.
type ExecutionResponse struct {
	StatusCode int
	Headers    map[string][]string
	Body       any
	URL        string // Actual request URL (populated by Executor)
	Method     string // HTTP method used (populated by Executor)
}

// EngineConfig configures engine behavior.
type EngineConfig struct {
	RetainResponseBodies bool // If false, nil out response bodies after extracting outputs
}

// Engine orchestrates the execution of Arazzo workflows.
// An Engine is NOT safe for concurrent use from multiple goroutines.
type Engine struct {
	document         *high.Arazzo
	executor         Executor
	sources          map[string]*ResolvedSource
	defaultSource    *ResolvedSource // cached for single-source fast path
	sourceOrder      []string        // deterministic source ordering from document
	workflows        map[string]*high.Workflow
	config           *EngineConfig
	exprCache        map[string]expression.Expression
	criterionCaches  *criterionCaches
	cachedComponents *expression.ComponentsContext // immutable component maps, built once
}

// NewEngine creates a new Engine for executing Arazzo workflows.
func NewEngine(doc *high.Arazzo, executor Executor, sources []*ResolvedSource) *Engine {
	sourceMap := make(map[string]*ResolvedSource, len(sources))
	for _, s := range sources {
		sourceMap[s.Name] = s
	}

	// Cache a default source for the single-source fast path to avoid map iteration per step.
	var defaultSource *ResolvedSource
	if len(sourceMap) == 1 {
		for _, s := range sourceMap {
			defaultSource = s
		}
	}

	// Build deterministic source ordering from the document's ordered SourceDescriptions list.
	var sourceOrder []string
	if doc != nil {
		sourceOrder = make([]string, 0, len(doc.SourceDescriptions))
		for _, sd := range doc.SourceDescriptions {
			if sd != nil {
				sourceOrder = append(sourceOrder, sd.Name)
			}
		}
	}

	var workflowMap map[string]*high.Workflow
	if doc != nil {
		workflowMap = make(map[string]*high.Workflow, len(doc.Workflows))
		for _, wf := range doc.Workflows {
			if wf == nil {
				continue
			}
			workflowMap[wf.WorkflowId] = wf
		}
	} else {
		workflowMap = make(map[string]*high.Workflow)
	}
	e := &Engine{
		document:        doc,
		executor:        executor,
		sources:         sourceMap,
		defaultSource:   defaultSource,
		sourceOrder:     sourceOrder,
		workflows:       workflowMap,
		config:          &EngineConfig{},
		exprCache:       make(map[string]expression.Expression),
		criterionCaches: newCriterionCaches(),
	}
	e.criterionCaches.parseExpr = e.parseExpression
	e.cachedComponents = e.buildCachedComponents()
	return e
}

// NewEngineWithConfig creates a new Engine with custom configuration.
func NewEngineWithConfig(doc *high.Arazzo, executor Executor, sources []*ResolvedSource, config *EngineConfig) *Engine {
	e := NewEngine(doc, executor, sources)
	if config != nil {
		e.config = config
	}
	return e
}

// ClearCaches resets all per-engine caches (expressions, regex, JSONPath).
func (e *Engine) ClearCaches() {
	e.exprCache = make(map[string]expression.Expression)
	e.criterionCaches = newCriterionCaches()
	e.criterionCaches.parseExpr = e.parseExpression
}

// RunWorkflow executes a single workflow by its ID.
func (e *Engine) RunWorkflow(ctx context.Context, workflowId string, inputs map[string]any) (*WorkflowResult, error) {
	state := &executionState{
		workflowResults:  make(map[string]*WorkflowResult),
		workflowContexts: make(map[string]*expression.WorkflowContext),
		activeWorkflows:  make(map[string]struct{}),
		depth:            0,
	}

	return e.runWorkflow(ctx, workflowId, inputs, state)
}

// RunAll executes all workflows in dependency order.
func (e *Engine) RunAll(ctx context.Context, inputs map[string]map[string]any) (*RunResult, error) {
	start := time.Now()
	result := &RunResult{
		Success: true,
	}

	state := &executionState{
		workflowResults:  make(map[string]*WorkflowResult),
		workflowContexts: make(map[string]*expression.WorkflowContext),
		activeWorkflows:  make(map[string]struct{}),
		depth:            0,
	}

	// Topological sort on dependsOn
	order, err := e.topologicalSort()
	if err != nil {
		return nil, err
	}
	for _, wfId := range order {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		wf := e.workflows[wfId]
		if wf != nil {
			if depErr := dependencyExecutionError(wf, state.workflowResults); depErr != nil {
				result.Success = false
				wfResult := &WorkflowResult{
					WorkflowId: wfId,
					Success:    false,
					Error:      depErr,
				}
				state.workflowResults[wfId] = wfResult
				result.Workflows = append(result.Workflows, wfResult)
				continue
			}
		}

		wfInputs := inputs[wfId]
		wfResult, execErr := e.runWorkflow(ctx, wfId, wfInputs, state)
		if execErr != nil {
			result.Success = false
			failedResult := &WorkflowResult{
				WorkflowId: wfId,
				Success:    false,
				Inputs:     wfInputs,
				Error:      execErr,
			}
			state.workflowResults[wfId] = failedResult
			result.Workflows = append(result.Workflows, failedResult)
			continue
		}
		result.Workflows = append(result.Workflows, wfResult)
		if !wfResult.Success {
			result.Success = false
		}
	}

	result.Duration = time.Since(start)
	return result, nil
}

type executionState struct {
	workflowResults  map[string]*WorkflowResult
	workflowContexts map[string]*expression.WorkflowContext
	activeWorkflows  map[string]struct{}
	depth            int
}

func (e *Engine) runWorkflow(ctx context.Context, workflowId string, inputs map[string]any, state *executionState) (*WorkflowResult, error) {
	if _, active := state.activeWorkflows[workflowId]; active {
		return nil, fmt.Errorf("%w: %s", ErrCircularDependency, workflowId)
	}

	if state.depth >= maxWorkflowDepth {
		return nil, fmt.Errorf("maximum workflow depth %d exceeded", maxWorkflowDepth)
	}

	wf := e.workflows[workflowId]
	if wf == nil {
		return nil, fmt.Errorf("%w: %s", ErrUnresolvedWorkflowRef, workflowId)
	}

	state.activeWorkflows[workflowId] = struct{}{}
	state.depth++
	defer func() {
		delete(state.activeWorkflows, workflowId)
		state.depth--
	}()

	start := time.Now()
	result := &WorkflowResult{
		WorkflowId: workflowId,
		Success:    true,
		Inputs:     inputs,
		Outputs:    make(map[string]any),
	}

	exprCtx, _ := e.newExpressionContext(inputs, state)
	// Error is non-fatal: unresolvable component input expressions fall back to raw YAML nodes.

	stepIdx := 0
	stepTransitions := 0
	stepIndexByID := make(map[string]int, len(wf.Steps))
	retryCounts := make(map[string]int, len(wf.Steps))
	for i, step := range wf.Steps {
		stepIndexByID[step.StepId] = i
	}

	for stepIdx < len(wf.Steps) {
		if err := ctx.Err(); err != nil {
			result.Success = false
			result.Error = err
			break
		}

		stepTransitions++
		if stepTransitions > maxStepTransitions {
			result.Success = false
			result.Error = fmt.Errorf("%w: exceeded max step transitions for workflow %q", ErrCircularDependency, wf.WorkflowId)
			break
		}

		step := wf.Steps[stepIdx]
		stepResult := e.executeStep(ctx, step, wf, exprCtx, state)
		stepResult.Retries = retryCounts[step.StepId]
		result.Steps = append(result.Steps, stepResult)

		nextStepIdx := stepIdx + 1
		if stepResult.Success {
			retryCounts[step.StepId] = 0
			actionResult, actionErr := e.processSuccessActions(ctx, step, wf, exprCtx, state, stepIndexByID)
			if actionErr != nil {
				result.Success = false
				result.Error = actionErr
				break
			}
			if actionResult.endWorkflow {
				break
			}
			if actionResult.jumpToStepIdx >= 0 {
				nextStepIdx = actionResult.jumpToStepIdx
			}
			stepIdx = nextStepIdx
			continue
		}

		actionResult, actionErr := e.processFailureActions(ctx, step, wf, exprCtx, state, stepIndexByID, retryCounts[step.StepId])
		if actionErr != nil {
			result.Success = false
			result.Error = actionErr
			break
		}
		if actionResult.retryCurrent {
			retryCounts[step.StepId]++
			if err := sleepWithContext(ctx, actionResult.retryAfter); err != nil {
				result.Success = false
				result.Error = err
				break
			}
			continue
		}
		if actionResult.endWorkflow {
			result.Success = false
			result.Error = stepResult.Error
			if result.Error == nil {
				result.Error = &StepFailureError{StepId: step.StepId, CriterionIndex: -1}
			}
			break
		}
		if actionResult.jumpToStepIdx >= 0 {
			stepIdx = actionResult.jumpToStepIdx
			continue
		}

		result.Success = false
		result.Error = stepResult.Error
		if result.Error == nil {
			result.Error = &StepFailureError{StepId: step.StepId, CriterionIndex: -1}
		}
		break
	}
	if result.Success {
		if err := e.populateWorkflowOutputs(wf, result, exprCtx); err != nil {
			result.Success = false
			result.Error = err
		}
	}

	result.Duration = time.Since(start)
	state.workflowResults[workflowId] = result
	state.workflowContexts[workflowId] = &expression.WorkflowContext{
		Inputs:  result.Inputs,
		Outputs: result.Outputs,
	}
	return result, nil
}

func (e *Engine) topologicalSort() ([]string, error) {
	if e.document == nil || len(e.document.Workflows) == 0 {
		return nil, nil
	}

	adj := make(map[string][]string)
	inDegree := make(map[string]int)
	workflowIds := make(map[string]struct{}, len(e.document.Workflows))

	for _, wf := range e.document.Workflows {
		if wf == nil {
			continue
		}
		id := wf.WorkflowId
		workflowIds[id] = struct{}{}
		if _, ok := inDegree[id]; !ok {
			inDegree[id] = 0
		}
	}
	for _, wf := range e.document.Workflows {
		if wf == nil {
			continue
		}
		id := wf.WorkflowId
		for _, dep := range wf.DependsOn {
			if _, ok := workflowIds[dep]; !ok {
				continue
			}
			adj[dep] = append(adj[dep], id)
			inDegree[id]++
		}
	}

	var queue []string
	for _, wf := range e.document.Workflows {
		if wf == nil {
			continue
		}
		id := wf.WorkflowId
		if inDegree[id] == 0 {
			queue = append(queue, id)
		}
	}

	var order []string
	for head := 0; head < len(queue); head++ {
		id := queue[head]
		order = append(order, id)

		for _, dependent := range adj[id] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	if len(order) != len(inDegree) {
		return nil, fmt.Errorf("%w in workflow dependencies", ErrCircularDependency)
	}

	return order, nil
}

func dependencyExecutionError(wf *high.Workflow, workflowResults map[string]*WorkflowResult) error {
	for _, depId := range wf.DependsOn {
		depResult, ok := workflowResults[depId]
		if !ok {
			return fmt.Errorf("%w: %s", ErrUnresolvedWorkflowRef, depId)
		}
		if !depResult.Success {
			if depResult.Error != nil {
				return fmt.Errorf("dependency %q failed: %w", depId, depResult.Error)
			}
			return fmt.Errorf("dependency %q failed", depId)
		}
	}
	return nil
}

// parseExpression parses and caches an expression.
func (e *Engine) parseExpression(input string) (expression.Expression, error) {
	if cached, ok := e.exprCache[input]; ok {
		return cached, nil
	}
	expr, err := expression.Parse(input)
	if err != nil {
		return expression.Expression{}, err
	}
	e.exprCache[input] = expr
	return expr, nil
}

// buildCachedComponents builds the immutable portion of the components context once.
// Parameters, SuccessActions, and FailureActions are read-only and shared across workflow runs.
// Inputs are resolved per-run because they may contain runtime expressions.
func (e *Engine) buildCachedComponents() *expression.ComponentsContext {
	if e.document == nil || e.document.Components == nil {
		return nil
	}
	components := &expression.ComponentsContext{}
	if e.document.Components.Parameters != nil {
		components.Parameters = make(map[string]any, e.document.Components.Parameters.Len())
		for name, parameter := range e.document.Components.Parameters.FromOldest() {
			components.Parameters[name] = parameter
		}
	}
	if e.document.Components.SuccessActions != nil {
		components.SuccessActions = make(map[string]any, e.document.Components.SuccessActions.Len())
		for name, action := range e.document.Components.SuccessActions.FromOldest() {
			components.SuccessActions[name] = action
		}
	}
	if e.document.Components.FailureActions != nil {
		components.FailureActions = make(map[string]any, e.document.Components.FailureActions.Len())
		for name, action := range e.document.Components.FailureActions.FromOldest() {
			components.FailureActions[name] = action
		}
	}
	return components
}

func (e *Engine) newExpressionContext(inputs map[string]any, state *executionState) (*expression.Context, error) {
	ctx := &expression.Context{
		Inputs:      inputs,
		Outputs:     make(map[string]any),
		Steps:       make(map[string]*expression.StepContext),
		Workflows:   copyWorkflowContexts(state.workflowContexts),
		SourceDescs: make(map[string]*expression.SourceDescContext),
	}
	for name, source := range e.sources {
		ctx.SourceDescs[name] = &expression.SourceDescContext{URL: source.URL}
	}
	if e.cachedComponents != nil {
		components := &expression.ComponentsContext{
			Parameters:     e.cachedComponents.Parameters,
			SuccessActions: e.cachedComponents.SuccessActions,
			FailureActions: e.cachedComponents.FailureActions,
		}
		var inputErrors []error
		if e.document.Components.Inputs != nil {
			components.Inputs = make(map[string]any, e.document.Components.Inputs.Len())
			for name, input := range e.document.Components.Inputs.FromOldest() {
				decoded, err := e.resolveYAMLNodeValue(input, ctx)
				if err != nil {
					inputErrors = append(inputErrors, fmt.Errorf("component input %q: %w", name, err))
					components.Inputs[name] = input
					continue
				}
				components.Inputs[name] = decoded
			}
		}
		ctx.Components = components
		if len(inputErrors) > 0 {
			return ctx, fmt.Errorf("failed to resolve component inputs: %w", errors.Join(inputErrors...))
		}
	}
	return ctx, nil
}

func copyWorkflowContexts(src map[string]*expression.WorkflowContext) map[string]*expression.WorkflowContext {
	if len(src) == 0 {
		return make(map[string]*expression.WorkflowContext)
	}
	dst := make(map[string]*expression.WorkflowContext, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

