// Copyright 2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/arazzo/expression"
	high "github.com/pb33f/libopenapi/datamodel/high/arazzo"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestEngine_RunWorkflow_PreservesParameterLocations(t *testing.T) {
	source := &ResolvedSource{
		Name: "pets", Type: "openapi",
		OpenAPIDocument: buildOpenAPISourceDoc("", "/pets/{id}", "get", "getPet"),
	}
	// The operation does not declare X-Trace. Its location must come from the workflow.
	require.Empty(t, source.OpenAPIDocument.Paths.PathItems.GetOrZero("/pets/{id}").Get.Parameters)
	doc := &high.Arazzo{
		Arazzo: "1.0.1",
		Workflows: []*high.Workflow{{
			WorkflowId: "get-pet",
			Steps: []*high.Step{{
				StepId: "fetch", OperationId: "getPet",
				Parameters: []*high.Parameter{
					{Name: "id", In: "path", Value: makeValueNode("$inputs.petId")},
					{Name: "id", In: "query", Value: makeValueNode("456")},
					{Name: "X-Trace", In: "header", Value: makeValueNode("$inputs.trace")},
					{Name: "session", In: "cookie", Value: makeValueNode("session-123")},
					{Name: "tags", In: "query", Value: makeValueNode("$inputs.tags")},
				},
				Outputs: orderedmap.FromPairs(
					orderedmap.NewPair("pathId", high.NewExpressionOutputValue("$request.path.id")),
					orderedmap.NewPair("queryId", high.NewExpressionOutputValue("$request.query.id")),
					orderedmap.NewPair("trace", high.NewExpressionOutputValue("$request.header.X-Trace")),
				),
			}},
		}},
	}
	executor := &captureExecutor{}
	engine := NewEngine(doc, executor, []*ResolvedSource{source})
	result, err := engine.RunWorkflow(context.Background(), "get-pet", map[string]any{
		"petId": 123, "trace": "trace-abc", "tags": []any{"cat", "available"},
	})
	require.NoError(t, err)
	require.True(t, result.Success, "%v", result.Error)
	req := executor.lastRequest
	require.NotNil(t, req)
	assert.Same(t, source, req.Source)
	assert.Equal(t, []ResolvedParameter{
		{Name: "id", In: "path", Value: 123},
		{Name: "id", In: "query", Value: 456},
		{Name: "X-Trace", In: "header", Value: "trace-abc"},
		{Name: "session", In: "cookie", Value: "session-123"},
		{Name: "tags", In: "query", Value: []any{"cat", "available"}},
	}, req.ResolvedParameters)
	// Existing executors keep the same name-keyed values and last-write behavior.
	assert.Equal(t, map[string]any{
		"id": 456, "X-Trace": "trace-abc", "session": "session-123",
		"tags": []any{"cat", "available"},
	}, req.Parameters)
	require.Len(t, result.Steps, 1)
	assert.Equal(t, map[string]any{"pathId": "123", "queryId": "456", "trace": "trace-abc"}, result.Steps[0].Outputs)
}

func TestBuildExecutionRequest_ResolvedReusableParameter(t *testing.T) {
	params := orderedmap.New[string, *high.Parameter]()
	params.Set("trace", &high.Parameter{Name: "X-Trace", In: "header", Value: makeValueNode("default-trace")})
	engine := NewEngine(&high.Arazzo{
		Arazzo: "1.0.1", Components: &high.Components{Parameters: params},
	}, nil, nil)
	for _, name := range []string{"component value", "step override"} {
		t.Run(name, func(t *testing.T) {
			param := &high.Parameter{Reference: "$components.parameters.trace"}
			want := "default-trace"
			if name == "step override" {
				param.Value = makeValueNode("$inputs.trace")
				want = "override-trace"
			}
			req, err := engine.buildExecutionRequest(nil, &high.Step{Parameters: []*high.Parameter{param}},
				&expression.Context{Inputs: map[string]any{"trace": "override-trace"}})
			require.NoError(t, err)
			assert.Equal(t, []ResolvedParameter{{Name: "X-Trace", In: "header", Value: want}}, req.ResolvedParameters)
			assert.Equal(t, want, req.Parameters["X-Trace"])
		})
	}
}

func TestBuildExecutionRequest_ResolvedParametersEmptyAndUnspecifiedLocation(t *testing.T) {
	engine := NewEngine(&high.Arazzo{}, nil, nil)
	req, err := engine.buildExecutionRequest(nil, &high.Step{}, &expression.Context{})
	require.NoError(t, err)
	assert.Nil(t, req.ResolvedParameters)
	assert.NotNil(t, req.Parameters)
	assert.Empty(t, req.Parameters)

	req, err = engine.buildExecutionRequest(nil, &high.Step{Parameters: []*high.Parameter{{Name: "optional"}}}, &expression.Context{})
	require.NoError(t, err)
	assert.Equal(t, []ResolvedParameter{{Name: "optional", In: "", Value: nil}}, req.ResolvedParameters)
	value, exists := req.Parameters["optional"]
	assert.True(t, exists)
	assert.Nil(t, value)
}

func TestEngine_RunWorkflow_AppliesWorkflowParameters(t *testing.T) {
	doc := &high.Arazzo{
		Arazzo: "1.0.1",
		Workflows: []*high.Workflow{{
			WorkflowId: "list-pets",
			Parameters: []*high.Parameter{
				{Name: "X-Trace", In: "header", Value: makeValueNode("$inputs.trace")},
				{Name: "limit", In: "query", Value: makeValueNode("10")},
				{Name: "id", In: "header", Value: makeValueNode("workflow-id")},
			},
			Steps: []*high.Step{
				{
					StepId: "override", OperationId: "getPet",
					Parameters: []*high.Parameter{
						// Same name, different location: not an override.
						{Name: "id", In: "path", Value: makeValueNode("123")},
						{Name: "limit", In: "query", Value: makeValueNode("50")},
					},
				},
				{StepId: "inherit", OperationId: "listPets"},
			},
		}},
	}
	executor := &requestLogExecutor{}
	engine := NewEngine(doc, executor, nil)
	result, err := engine.RunWorkflow(context.Background(), "list-pets", map[string]any{"trace": "trace-abc"})
	require.NoError(t, err)
	require.True(t, result.Success, "%v", result.Error)
	require.Len(t, executor.requests, 2)

	assert.Equal(t, []ResolvedParameter{
		{Name: "X-Trace", In: "header", Value: "trace-abc"},
		{Name: "id", In: "header", Value: "workflow-id"},
		{Name: "id", In: "path", Value: 123},
		{Name: "limit", In: "query", Value: 50},
	}, executor.requests[0].ResolvedParameters)
	assert.Equal(t, map[string]any{"X-Trace": "trace-abc", "id": 123, "limit": 50}, executor.requests[0].Parameters)

	assert.Equal(t, []ResolvedParameter{
		{Name: "X-Trace", In: "header", Value: "trace-abc"},
		{Name: "limit", In: "query", Value: 10},
		{Name: "id", In: "header", Value: "workflow-id"},
	}, executor.requests[1].ResolvedParameters)
}

func TestEngine_RunWorkflow_PassesWorkflowParametersToSubWorkflow(t *testing.T) {
	doc := &high.Arazzo{
		Arazzo: "1.0.1",
		Workflows: []*high.Workflow{
			{
				WorkflowId: "parent",
				Parameters: []*high.Parameter{
					{Name: "token", Value: makeValueNode("parent-token")},
					{Name: "petId", Value: makeValueNode("1")},
				},
				Steps: []*high.Step{{
					StepId: "call-child", WorkflowId: "child",
					Parameters: []*high.Parameter{{Name: "petId", Value: makeValueNode("2")}},
				}},
			},
			{
				WorkflowId: "child",
				Steps: []*high.Step{{
					StepId: "fetch", OperationId: "getPet",
					Parameters: []*high.Parameter{
						{Name: "Authorization", In: "header", Value: makeValueNode("$inputs.token")},
						{Name: "id", In: "path", Value: makeValueNode("$inputs.petId")},
					},
				}},
			},
		},
	}
	executor := &requestLogExecutor{}
	engine := NewEngine(doc, executor, nil)
	result, err := engine.RunWorkflow(context.Background(), "parent", nil)
	require.NoError(t, err)
	require.True(t, result.Success, "%v", result.Error)
	require.Len(t, executor.requests, 1)
	assert.Equal(t, []ResolvedParameter{
		{Name: "Authorization", In: "header", Value: "parent-token"},
		{Name: "id", In: "path", Value: 2},
	}, executor.requests[0].ResolvedParameters)
}

func TestEngine_RunWorkflow_UnresolvedWorkflowParameter(t *testing.T) {
	for name, step := range map[string]*high.Step{
		"operation step": {StepId: "s1", OperationId: "getPet"},
		"workflow step":  {StepId: "s1", WorkflowId: "child"},
	} {
		t.Run(name, func(t *testing.T) {
			doc := &high.Arazzo{
				Arazzo: "1.0.1",
				Workflows: []*high.Workflow{
					{
						WorkflowId: "parent",
						Parameters: []*high.Parameter{{Reference: "$components.parameters.missing"}},
						Steps:      []*high.Step{step},
					},
					{WorkflowId: "child", Steps: []*high.Step{{StepId: "noop", OperationId: "listPets"}}},
				},
			}
			executor := &requestLogExecutor{}
			result, err := NewEngine(doc, executor, nil).RunWorkflow(context.Background(), "parent", nil)
			require.NoError(t, err)
			assert.False(t, result.Success)
			require.ErrorIs(t, result.Error, ErrUnresolvedComponent)
			assert.Empty(t, executor.requests)
		})
	}
}

type requestLogExecutor struct {
	requests []*ExecutionRequest
}

func (r *requestLogExecutor) Execute(_ context.Context, req *ExecutionRequest) (*ExecutionResponse, error) {
	r.requests = append(r.requests, req)
	return &ExecutionResponse{StatusCode: 200}, nil
}
