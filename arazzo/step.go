// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pb33f/libopenapi/arazzo/expression"
	high "github.com/pb33f/libopenapi/datamodel/high/arazzo"
	"go.yaml.in/yaml/v4"
)

func (e *Engine) executeStep(ctx context.Context, step *high.Step, wf *high.Workflow, exprCtx *expression.Context, state *executionState) *StepResult {
	_ = wf // retained for future per-workflow step configuration
	start := time.Now()
	result := &StepResult{
		StepId:  step.StepId,
		Success: true,
		Outputs: make(map[string]any),
	}
	exprCtx.StatusCode = 0
	exprCtx.RequestHeaders = nil
	exprCtx.RequestQuery = nil
	exprCtx.RequestPath = nil
	exprCtx.RequestBody = nil
	exprCtx.ResponseHeaders = nil
	exprCtx.ResponseBody = nil
	var stepInputs map[string]any

	if step.WorkflowId != "" {
		if len(step.Parameters) > 0 {
			stepInputs = make(map[string]any, len(step.Parameters))
			for _, param := range step.Parameters {
				resolvedParam, err := e.resolveParameter(param)
				if err != nil {
					result.Success = false
					result.Error = err
					break
				}
				value, err := e.resolveYAMLNodeValue(resolvedParam.Value, exprCtx)
				if err != nil {
					result.Success = false
					result.Error = fmt.Errorf("failed to evaluate parameter %q for step %q: %w", resolvedParam.Name, step.StepId, err)
					break
				}
				stepInputs[resolvedParam.Name] = value
			}
		}
		if result.Success {
			wfResult, err := e.runWorkflow(ctx, step.WorkflowId, stepInputs, state)
			if err != nil {
				result.Success = false
				result.Error = err
			} else if !wfResult.Success {
				result.Success = false
				result.Error = wfResult.Error
			}
			exprCtx.Workflows = copyWorkflowContexts(state.workflowContexts)
		}
	} else {
		req, err := e.buildExecutionRequest(step, exprCtx)
		if err != nil {
			result.Success = false
			result.Error = err
		} else {
			stepInputs = req.Parameters

			if e.executor == nil {
				result.Success = false
				result.Error = ErrExecutorNotConfigured
			} else {
				resp, execErr := e.executor.Execute(ctx, req)
				if execErr != nil {
					result.Success = false
					result.Error = execErr
				} else {
					result.StatusCode = resp.StatusCode

					exprCtx.StatusCode = resp.StatusCode
					exprCtx.URL = resp.URL
					exprCtx.Method = resp.Method
					exprCtx.ResponseHeaders = firstHeaderValues(resp.Headers)
					exprCtx.ResponseBody, execErr = toYAMLNode(resp.Body)
					if execErr != nil {
						result.Success = false
						result.Error = execErr
					} else if !e.config.RetainResponseBodies {
						resp.Body = nil
					}
				}
			}
		}
	}
	if result.Success {
		if err := e.evaluateStepSuccessCriteria(step, exprCtx); err != nil {
			result.Success = false
			result.Error = err
		}
	}
	if result.Success {
		if err := e.populateStepOutputs(step, result, exprCtx); err != nil {
			result.Success = false
			result.Error = err
		}
	}

	exprCtx.Steps[step.StepId] = &expression.StepContext{
		Inputs:  stepInputs,
		Outputs: result.Outputs,
	}

	result.Duration = time.Since(start)
	return result
}

func (e *Engine) evaluateStepSuccessCriteria(step *high.Step, exprCtx *expression.Context) error {
	if len(step.SuccessCriteria) == 0 {
		return nil
	}

	for i, criterion := range step.SuccessCriteria {
		ok, err := evaluateCriterionImpl(criterion, exprCtx, e.criterionCaches)
		if err != nil {
			return &StepFailureError{StepId: step.StepId, CriterionIndex: i, Cause: err}
		}
		if !ok {
			return &StepFailureError{StepId: step.StepId, CriterionIndex: i, Message: "not satisfied"}
		}
	}
	return nil
}

func (e *Engine) buildExecutionRequest(step *high.Step, exprCtx *expression.Context) (*ExecutionRequest, error) {
	req := &ExecutionRequest{
		OperationID:   step.OperationId,
		OperationPath: step.OperationPath,
		Parameters:    make(map[string]any),
	}
	req.Source = e.resolveStepSource(step)
	requestHeaders := make(map[string]string)
	requestQuery := make(map[string]string)
	requestPath := make(map[string]string)

	for _, param := range step.Parameters {
		resolvedParam, err := e.resolveParameter(param)
		if err != nil {
			return nil, err
		}
		value, err := e.resolveYAMLNodeValue(resolvedParam.Value, exprCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate parameter %q for step %q: %w", resolvedParam.Name, step.StepId, err)
		}
		req.Parameters[resolvedParam.Name] = value

		switch resolvedParam.In {
		case "header":
			requestHeaders[resolvedParam.Name] = fmt.Sprint(value)
		case "query":
			requestQuery[resolvedParam.Name] = fmt.Sprint(value)
		case "path":
			requestPath[resolvedParam.Name] = fmt.Sprint(value)
		}
	}

	if step.RequestBody != nil {
		requestBody, err := e.resolveYAMLNodeValue(step.RequestBody.Payload, exprCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate requestBody for step %q: %w", step.StepId, err)
		}
		if len(step.RequestBody.Replacements) > 0 {
			requestBody, err = e.applyPayloadReplacements(requestBody, step.RequestBody.Replacements, exprCtx, step.StepId)
			if err != nil {
				return nil, err
			}
		}
		req.RequestBody = requestBody
		req.ContentType = step.RequestBody.ContentType

		exprCtx.RequestBody, err = toYAMLNode(requestBody)
		if err != nil {
			return nil, fmt.Errorf("failed to parse requestBody for step %q: %w", step.StepId, err)
		}
	}

	if len(requestHeaders) > 0 {
		exprCtx.RequestHeaders = requestHeaders
	}
	if len(requestQuery) > 0 {
		exprCtx.RequestQuery = requestQuery
	}
	if len(requestPath) > 0 {
		exprCtx.RequestPath = requestPath
	}

	return req, nil
}

func (e *Engine) resolveStepSource(step *high.Step) *ResolvedSource {
	if len(e.sources) == 0 || step == nil {
		return nil
	}
	if e.defaultSource != nil {
		return e.defaultSource
	}
	if name, found := extractSourceNameFromOperationPath(step.OperationPath); found {
		if source, ok := e.sources[name]; ok {
			return source
		}
	}
	// Deterministic fallback: use the first source from the document's ordered list.
	for _, name := range e.sourceOrder {
		if source, ok := e.sources[name]; ok {
			return source
		}
	}
	return nil
}

func (e *Engine) resolveParameter(param *high.Parameter) (*high.Parameter, error) {
	if param == nil {
		return nil, fmt.Errorf("nil step parameter")
	}
	if !param.IsReusable() {
		return param, nil
	}
	const prefix = "$components.parameters."
	if !strings.HasPrefix(param.Reference, prefix) {
		return nil, fmt.Errorf("%w: %q", ErrUnresolvedComponent, param.Reference)
	}
	if e.document == nil || e.document.Components == nil || e.document.Components.Parameters == nil {
		return nil, fmt.Errorf("%w: %q", ErrUnresolvedComponent, param.Reference)
	}
	componentName := strings.TrimPrefix(param.Reference, prefix)
	componentParameter, ok := e.document.Components.Parameters.Get(componentName)
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnresolvedComponent, param.Reference)
	}
	resolved := &high.Parameter{
		Name:  componentParameter.Name,
		In:    componentParameter.In,
		Value: componentParameter.Value,
	}
	if param.Value != nil {
		resolved.Value = param.Value
	}
	return resolved, nil
}

func (e *Engine) resolveYAMLNodeValue(node *yaml.Node, exprCtx *expression.Context) (any, error) {
	if node == nil {
		return nil, nil
	}
	var decoded any
	if err := node.Decode(&decoded); err != nil {
		return nil, err
	}
	return e.resolveExpressionValues(decoded, exprCtx)
}

func (e *Engine) resolveExpressionValues(value any, exprCtx *expression.Context) (any, error) {
	switch typed := value.(type) {
	case string:
		return e.evaluateStringValue(typed, exprCtx)
	case []any:
		if !sliceNeedsResolution(typed) {
			return typed, nil
		}
		items := make([]any, len(typed))
		for i := range typed {
			resolved, err := e.resolveExpressionValues(typed[i], exprCtx)
			if err != nil {
				return nil, err
			}
			items[i] = resolved
		}
		return items, nil
	case map[string]any:
		if !mapNeedsResolution(typed) {
			return typed, nil
		}
		items := make(map[string]any, len(typed))
		for k, v := range typed {
			resolved, err := e.resolveExpressionValues(v, exprCtx)
			if err != nil {
				return nil, err
			}
			items[k] = resolved
		}
		return items, nil
	case map[any]any:
		items := make(map[string]any, len(typed))
		resolve := mapAnyNeedsResolution(typed)
		for k, v := range typed {
			ks := sprintMapKey(k)
			if !resolve {
				items[ks] = v
				continue
			}
			resolved, err := e.resolveExpressionValues(v, exprCtx)
			if err != nil {
				return nil, err
			}
			items[ks] = resolved
		}
		return items, nil
	default:
		return typed, nil
	}
}

func (e *Engine) applyPayloadReplacements(payload any, replacements []*high.PayloadReplacement, exprCtx *expression.Context, stepId string) (any, error) {
	root, ok := payload.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("cannot apply payload replacements to non-object body in step %q", stepId)
	}
	for _, rep := range replacements {
		if rep == nil || rep.Target == "" {
			continue
		}
		value, err := e.resolveYAMLNodeValue(rep.Value, exprCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate replacement value for target %q in step %q: %w", rep.Target, stepId, err)
		}
		if err := setJSONPointerValue(root, rep.Target, value); err != nil {
			return nil, fmt.Errorf("failed to apply replacement at %q in step %q: %w", rep.Target, stepId, err)
		}
	}
	return root, nil
}

func setJSONPointerValue(root map[string]any, pointer string, value any) error {
	if pointer == "" {
		return fmt.Errorf("empty JSON pointer")
	}
	if pointer[0] != '/' {
		return fmt.Errorf("JSON pointer must start with /")
	}

	segments := strings.Split(pointer[1:], "/")
	for i := range segments {
		segments[i] = expression.UnescapeJSONPointer(segments[i])
	}

	current := any(root)
	for i := 0; i < len(segments)-1; i++ {
		seg := segments[i]
		switch m := current.(type) {
		case map[string]any:
			next, exists := m[seg]
			if !exists {
				child := make(map[string]any)
				m[seg] = child
				current = child
			} else {
				current = next
			}
		default:
			return fmt.Errorf("cannot traverse into %T at segment %q", current, seg)
		}
	}

	lastSeg := segments[len(segments)-1]
	switch m := current.(type) {
	case map[string]any:
		m[lastSeg] = value
		return nil
	default:
		return fmt.Errorf("cannot set value at %q: parent is %T", pointer, current)
	}
}

func valueNeedsResolution(v any) bool {
	switch s := v.(type) {
	case string:
		return strings.HasPrefix(s, "$") || strings.Contains(s, "{$")
	case []any, map[string]any, map[any]any:
		return true
	default:
		return false
	}
}

func sliceNeedsResolution(items []any) bool {
	for _, v := range items {
		if valueNeedsResolution(v) {
			return true
		}
	}
	return false
}

func mapAnyNeedsResolution(items map[any]any) bool {
	for _, v := range items {
		if valueNeedsResolution(v) {
			return true
		}
	}
	return false
}

func mapNeedsResolution(items map[string]any) bool {
	for _, v := range items {
		if valueNeedsResolution(v) {
			return true
		}
	}
	return false
}

func (e *Engine) evaluateStringValue(input string, exprCtx *expression.Context) (any, error) {
	if strings.HasPrefix(input, "$") {
		parsed, err := e.parseExpression(input)
		if err != nil {
			return nil, err
		}
		return expression.Evaluate(parsed, exprCtx)
	}
	if strings.Contains(input, "{$") {
		tokens, err := expression.ParseEmbedded(input)
		if err != nil {
			return nil, err
		}
		if len(tokens) == 1 && tokens[0].IsExpression {
			return expression.Evaluate(tokens[0].Expression, exprCtx)
		}
		var rendered strings.Builder
		for _, token := range tokens {
			if !token.IsExpression {
				rendered.WriteString(token.Literal)
				continue
			}
			val, err := expression.Evaluate(token.Expression, exprCtx)
			if err != nil {
				return nil, err
			}
			rendered.WriteString(fmt.Sprint(val))
		}
		return rendered.String(), nil
	}
	return input, nil
}

func (e *Engine) populateStepOutputs(step *high.Step, result *StepResult, exprCtx *expression.Context) error {
	if step.Outputs == nil || step.Outputs.Len() == 0 {
		return nil
	}
	for name, outputExpression := range step.Outputs.FromOldest() {
		value, err := e.evaluateStringValue(outputExpression, exprCtx)
		if err != nil {
			return fmt.Errorf("failed to evaluate output %q for step %q: %w", name, step.StepId, err)
		}
		result.Outputs[name] = value
	}
	return nil
}

func (e *Engine) populateWorkflowOutputs(wf *high.Workflow, result *WorkflowResult, exprCtx *expression.Context) error {
	if wf.Outputs == nil || wf.Outputs.Len() == 0 {
		return nil
	}
	for name, outputExpression := range wf.Outputs.FromOldest() {
		value, err := e.evaluateStringValue(outputExpression, exprCtx)
		if err != nil {
			return fmt.Errorf("failed to evaluate output %q for workflow %q: %w", name, wf.WorkflowId, err)
		}
		result.Outputs[name] = value
		exprCtx.Outputs[name] = value
	}
	return nil
}

func firstHeaderValues(headers map[string][]string) map[string]string {
	if len(headers) == 0 {
		return nil
	}
	values := make(map[string]string, len(headers))
	for name, headerValues := range headers {
		if len(headerValues) == 0 {
			continue
		}
		values[name] = headerValues[0]
	}
	return values
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
