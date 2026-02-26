// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"context"
	"errors"
	"testing"

	"github.com/pb33f/libopenapi/arazzo/expression"
	high "github.com/pb33f/libopenapi/datamodel/high/arazzo"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

type recordingExecutor struct {
	operationIDs []string
}

func (r *recordingExecutor) Execute(_ context.Context, req *ExecutionRequest) (*ExecutionResponse, error) {
	r.operationIDs = append(r.operationIDs, req.OperationID)
	return &ExecutionResponse{StatusCode: 200}, nil
}

type failingExecutor struct {
	err error
}

func (f *failingExecutor) Execute(_ context.Context, _ *ExecutionRequest) (*ExecutionResponse, error) {
	return nil, f.err
}

type captureExecutor struct {
	lastRequest *ExecutionRequest
	response    *ExecutionResponse
}

func (c *captureExecutor) Execute(_ context.Context, req *ExecutionRequest) (*ExecutionResponse, error) {
	c.lastRequest = req
	if c.response != nil {
		return c.response, nil
	}
	return &ExecutionResponse{StatusCode: 200}, nil
}

type statusRecordingExecutor struct {
	operationIDs      []string
	statusByOperation map[string]int
}

func (s *statusRecordingExecutor) Execute(_ context.Context, req *ExecutionRequest) (*ExecutionResponse, error) {
	s.operationIDs = append(s.operationIDs, req.OperationID)
	status := 200
	if s.statusByOperation != nil {
		if customStatus, ok := s.statusByOperation[req.OperationID]; ok {
			status = customStatus
		}
	}
	return &ExecutionResponse{StatusCode: status}, nil
}

type sequenceExecutor struct {
	operationIDs []string
	statuses     map[string][]int
	index        map[string]int
	response     *ExecutionResponse
}

func (s *sequenceExecutor) Execute(_ context.Context, req *ExecutionRequest) (*ExecutionResponse, error) {
	s.operationIDs = append(s.operationIDs, req.OperationID)
	if s.response != nil {
		return s.response, nil
	}
	if s.index == nil {
		s.index = make(map[string]int)
	}
	series := s.statuses[req.OperationID]
	if len(series) == 0 {
		return &ExecutionResponse{StatusCode: 200}, nil
	}
	pos := s.index[req.OperationID]
	if pos >= len(series) {
		pos = len(series) - 1
	}
	status := series[pos]
	s.index[req.OperationID]++
	return &ExecutionResponse{StatusCode: status}, nil
}

func TestEngine_RunAll_RespectsWorkflowDependencies(t *testing.T) {
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{
				WorkflowId: "wf1",
				Steps: []*high.Step{
					{StepId: "s1", OperationId: "op1"},
				},
			},
			{
				WorkflowId: "wf2",
				DependsOn:  []string{"wf1"},
				Steps: []*high.Step{
					{StepId: "s2", OperationId: "op2"},
				},
			},
		},
	}
	executor := &recordingExecutor{}
	engine := NewEngine(doc, executor, nil)

	result, err := engine.RunAll(context.Background(), nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, []string{"op1", "op2"}, executor.operationIDs)
}

func TestEngine_RunAll_MissingDependencyIsNotExecutedAndDependentFails(t *testing.T) {
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{
				WorkflowId: "wf1",
				Steps: []*high.Step{
					{StepId: "s1", OperationId: "op1"},
				},
			},
			{
				WorkflowId: "wf2",
				DependsOn:  []string{"missing"},
				Steps: []*high.Step{
					{StepId: "s2", OperationId: "op2"},
				},
			},
		},
	}
	executor := &recordingExecutor{}
	engine := NewEngine(doc, executor, nil)

	result, err := engine.RunAll(context.Background(), nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success)

	byID := make(map[string]*WorkflowResult, len(result.Workflows))
	for _, wf := range result.Workflows {
		byID[wf.WorkflowId] = wf
	}

	assert.NotContains(t, byID, "missing")
	require.Contains(t, byID, "wf2")
	assert.False(t, byID["wf2"].Success)
	require.Error(t, byID["wf2"].Error)
	assert.ErrorIs(t, byID["wf2"].Error, ErrUnresolvedWorkflowRef)
	assert.Contains(t, byID["wf2"].Error.Error(), "missing")
	assert.Equal(t, []string{"op1"}, executor.operationIDs)
}

func TestEngine_RunWorkflow_PropagatesFailedStepErrorToWorkflow(t *testing.T) {
	execErr := errors.New("executor failed")
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{
				WorkflowId: "wf1",
				Steps: []*high.Step{
					{StepId: "s1", OperationId: "op1"},
				},
			},
		},
	}
	engine := NewEngine(doc, &failingExecutor{err: execErr}, nil)

	result, err := engine.RunWorkflow(context.Background(), "wf1", nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success)
	require.Len(t, result.Steps, 1)
	require.Error(t, result.Steps[0].Error)
	assert.ErrorIs(t, result.Steps[0].Error, execErr)
	require.Error(t, result.Error)
	assert.ErrorIs(t, result.Error, execErr)
}

func TestEngine_RunWorkflow_PopulatesExecutionRequestFromStepInputs(t *testing.T) {
	var payloadNode yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("name: fluffy\nage: 2\n"), &payloadNode))

	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{
				WorkflowId: "wf1",
				Steps: []*high.Step{
					{
						StepId:      "s1",
						OperationId: "createPet",
						Parameters: []*high.Parameter{
							{Name: "api_key", In: "header", Value: &yaml.Node{Kind: yaml.ScalarNode, Value: "abc123"}},
							{Name: "limit", In: "query", Value: &yaml.Node{Kind: yaml.ScalarNode, Value: "10"}},
						},
						RequestBody: &high.RequestBody{
							ContentType: "application/json",
							Payload:     payloadNode.Content[0],
						},
					},
				},
			},
		},
	}

	executor := &captureExecutor{}
	engine := NewEngine(doc, executor, nil)

	result, err := engine.RunWorkflow(context.Background(), "wf1", nil)
	require.NoError(t, err)
	require.True(t, result.Success)
	require.NotNil(t, executor.lastRequest)
	assert.Equal(t, "createPet", executor.lastRequest.OperationID)
	assert.Equal(t, "abc123", executor.lastRequest.Parameters["api_key"])
	assert.Equal(t, 10, executor.lastRequest.Parameters["limit"])
	assert.Equal(t, "application/json", executor.lastRequest.ContentType)

	requestBody, ok := executor.lastRequest.RequestBody.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "fluffy", requestBody["name"])
	assert.Equal(t, 2, requestBody["age"])
}

func TestEngine_RunWorkflow_PassesStepParametersToNestedWorkflowInputs(t *testing.T) {
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{
				WorkflowId: "main",
				Steps: []*high.Step{
					{
						StepId:     "callSub",
						WorkflowId: "sub",
						Parameters: []*high.Parameter{
							{Name: "token", Value: &yaml.Node{Kind: yaml.ScalarNode, Value: "$inputs.token"}},
						},
					},
				},
			},
			{
				WorkflowId: "sub",
				Steps: []*high.Step{
					{
						StepId:      "useInput",
						OperationId: "op-sub",
						Parameters: []*high.Parameter{
							{Name: "auth", In: "header", Value: &yaml.Node{Kind: yaml.ScalarNode, Value: "$inputs.token"}},
						},
					},
				},
			},
		},
	}

	executor := &captureExecutor{}
	engine := NewEngine(doc, executor, nil)

	result, err := engine.RunWorkflow(context.Background(), "main", map[string]any{"token": "secret"})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	require.NotNil(t, executor.lastRequest)
	assert.Equal(t, "op-sub", executor.lastRequest.OperationID)
	assert.Equal(t, "secret", executor.lastRequest.Parameters["auth"])
}

func TestEngine_RunWorkflow_EvaluatesStepAndWorkflowOutputs(t *testing.T) {
	stepOutputs := orderedmap.New[string, string]()
	stepOutputs.Set("petId", "$response.body#/id")
	workflowOutputs := orderedmap.New[string, string]()
	workflowOutputs.Set("createdPetId", "$steps.s1.outputs.petId")

	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{
				WorkflowId: "wf1",
				Steps: []*high.Step{
					{
						StepId:      "s1",
						OperationId: "createPet",
						Outputs:     stepOutputs,
					},
				},
				Outputs: workflowOutputs,
			},
		},
	}

	executor := &captureExecutor{
		response: &ExecutionResponse{
			StatusCode: 201,
			Body:       map[string]any{"id": "pet-42"},
		},
	}
	engine := NewEngine(doc, executor, nil)

	result, err := engine.RunWorkflow(context.Background(), "wf1", nil)
	require.NoError(t, err)
	require.True(t, result.Success)
	require.Len(t, result.Steps, 1)
	assert.Equal(t, "pet-42", result.Steps[0].Outputs["petId"])
	assert.Equal(t, "pet-42", result.Outputs["createdPetId"])
}

func TestEngine_RunWorkflow_FailsWhenSuccessCriteriaNotMet(t *testing.T) {
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{
				WorkflowId: "wf1",
				Steps: []*high.Step{
					{
						StepId:      "s1",
						OperationId: "op1",
						SuccessCriteria: []*high.Criterion{
							{Condition: "$statusCode == 200"},
						},
					},
					{
						StepId:      "s2",
						OperationId: "op2",
					},
				},
			},
		},
	}

	executor := &statusRecordingExecutor{
		statusByOperation: map[string]int{
			"op1": 500,
			"op2": 200,
		},
	}
	engine := NewEngine(doc, executor, nil)

	result, err := engine.RunWorkflow(context.Background(), "wf1", nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success)
	require.Len(t, result.Steps, 1)
	assert.False(t, result.Steps[0].Success)
	require.Error(t, result.Steps[0].Error)
	assert.Contains(t, result.Steps[0].Error.Error(), "successCriteria[0]")
	assert.Equal(t, []string{"op1"}, executor.operationIDs)
}

func TestEngine_RunAll_DeterministicOrderForIndependentWorkflows(t *testing.T) {
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{
				WorkflowId: "wf3",
				Steps: []*high.Step{
					{StepId: "s3", OperationId: "op3"},
				},
			},
			{
				WorkflowId: "wf1",
				Steps: []*high.Step{
					{StepId: "s1", OperationId: "op1"},
				},
			},
			{
				WorkflowId: "wf2",
				Steps: []*high.Step{
					{StepId: "s2", OperationId: "op2"},
				},
			},
		},
	}

	for i := 0; i < 25; i++ {
		executor := &recordingExecutor{}
		engine := NewEngine(doc, executor, nil)

		result, err := engine.RunAll(context.Background(), nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Workflows, 3)

		assert.Equal(t, []string{"op3", "op1", "op2"}, executor.operationIDs)
		assert.Equal(t, "wf3", result.Workflows[0].WorkflowId)
		assert.Equal(t, "wf1", result.Workflows[1].WorkflowId)
		assert.Equal(t, "wf2", result.Workflows[2].WorkflowId)
	}
}

func TestEngine_RunWorkflow_OnFailureRetry_ReusesComponentAction(t *testing.T) {
	failureActions := orderedmap.New[string, *high.FailureAction]()
	failureActions.Set("retryOnce", &high.FailureAction{Name: "retryOnce", Type: "retry", RetryLimit: ptrInt64(1)})
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{
				WorkflowId: "wf1",
				Steps: []*high.Step{
					{
						StepId:      "s1",
						OperationId: "op1",
						SuccessCriteria: []*high.Criterion{
							{Condition: "$statusCode == 200"},
						},
						OnFailure: []*high.FailureAction{
							{Reference: "$components.failureActions.retryOnce"},
						},
					},
					{
						StepId:      "s2",
						OperationId: "op2",
					},
				},
			},
		},
		Components: &high.Components{
			FailureActions: failureActions,
		},
	}

	executor := &sequenceExecutor{
		statuses: map[string][]int{
			"op1": {500, 200},
			"op2": {200},
		},
	}
	engine := NewEngine(doc, executor, nil)

	result, err := engine.RunWorkflow(context.Background(), "wf1", nil)
	require.NoError(t, err)
	require.True(t, result.Success)
	require.Len(t, result.Steps, 3)
	assert.False(t, result.Steps[0].Success)
	assert.Equal(t, 0, result.Steps[0].Retries)
	assert.True(t, result.Steps[1].Success)
	assert.Equal(t, 1, result.Steps[1].Retries)
	assert.Equal(t, []string{"op1", "op1", "op2"}, executor.operationIDs)
}

func TestEngine_RunWorkflow_OnSuccessGotoStep(t *testing.T) {
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{
				WorkflowId: "wf1",
				Steps: []*high.Step{
					{
						StepId:      "s1",
						OperationId: "op1",
						OnSuccess: []*high.SuccessAction{
							{Name: "jump", Type: "goto", StepId: "s3"},
						},
					},
					{StepId: "s2", OperationId: "op2"},
					{StepId: "s3", OperationId: "op3"},
				},
			},
		},
	}
	executor := &sequenceExecutor{}
	engine := NewEngine(doc, executor, nil)

	result, err := engine.RunWorkflow(context.Background(), "wf1", nil)
	require.NoError(t, err)
	require.True(t, result.Success)
	assert.Equal(t, []string{"op1", "op3"}, executor.operationIDs)
}

func TestEngine_RunWorkflow_OnSuccessEnd(t *testing.T) {
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{
				WorkflowId: "wf1",
				Steps: []*high.Step{
					{
						StepId:      "s1",
						OperationId: "op1",
						OnSuccess: []*high.SuccessAction{
							{Name: "done", Type: "end"},
						},
					},
					{StepId: "s2", OperationId: "op2"},
				},
			},
		},
	}
	executor := &sequenceExecutor{}
	engine := NewEngine(doc, executor, nil)

	result, err := engine.RunWorkflow(context.Background(), "wf1", nil)
	require.NoError(t, err)
	require.True(t, result.Success)
	assert.Equal(t, []string{"op1"}, executor.operationIDs)
}

func TestEngine_RunWorkflow_OnFailureGotoStep(t *testing.T) {
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{
				WorkflowId: "wf1",
				Steps: []*high.Step{
					{
						StepId:      "s1",
						OperationId: "op1",
						SuccessCriteria: []*high.Criterion{
							{Condition: "$statusCode == 200"},
						},
						OnFailure: []*high.FailureAction{
							{Name: "recover", Type: "goto", StepId: "s3"},
						},
					},
					{StepId: "s2", OperationId: "op2"},
					{StepId: "s3", OperationId: "op3"},
				},
			},
		},
	}
	executor := &sequenceExecutor{
		statuses: map[string][]int{
			"op1": {500},
		},
	}
	engine := NewEngine(doc, executor, nil)

	result, err := engine.RunWorkflow(context.Background(), "wf1", nil)
	require.NoError(t, err)
	require.True(t, result.Success)
	assert.Equal(t, []string{"op1", "op3"}, executor.operationIDs)
}

func TestEngine_BuildExecutionRequest_PopulatesSource(t *testing.T) {
	doc := &high.Arazzo{Arazzo: "1.0.1"}
	sources := []*ResolvedSource{
		{Name: "fallback", URL: "https://example.com/fallback.yaml"},
		{Name: "api", URL: "https://example.com/openapi.yaml"},
	}
	engine := NewEngine(doc, nil, sources)
	step := &high.Step{
		StepId:        "s1",
		OperationPath: "{$sourceDescriptions.api.url}#/paths/~1pets/get",
	}
	exprCtx := &expression.Context{
		Inputs:  make(map[string]any),
		Steps:   make(map[string]*expression.StepContext),
		Outputs: make(map[string]any),
	}

	req, err := engine.buildExecutionRequest(step, exprCtx)
	require.NoError(t, err)
	require.NotNil(t, req.Source)
	assert.Equal(t, "api", req.Source.Name)
}

func TestEngine_RunWorkflow_RetainResponseBodiesHonorsConfig(t *testing.T) {
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{
				WorkflowId: "wf1",
				Steps: []*high.Step{
					{StepId: "s1", OperationId: "op1"},
				},
			},
		},
	}

	t.Run("disabled", func(t *testing.T) {
		exec := &sequenceExecutor{
			response: &ExecutionResponse{
				StatusCode: 200,
				Body:       map[string]any{"id": 123},
			},
		}
		engine := NewEngineWithConfig(doc, exec, nil, &EngineConfig{RetainResponseBodies: false})
		result, err := engine.RunWorkflow(context.Background(), "wf1", nil)
		require.NoError(t, err)
		require.True(t, result.Success)
		assert.Nil(t, exec.response.Body)
	})

	t.Run("enabled", func(t *testing.T) {
		exec := &sequenceExecutor{
			response: &ExecutionResponse{
				StatusCode: 200,
				Body:       map[string]any{"id": 123},
			},
		}
		engine := NewEngineWithConfig(doc, exec, nil, &EngineConfig{RetainResponseBodies: true})
		result, err := engine.RunWorkflow(context.Background(), "wf1", nil)
		require.NoError(t, err)
		require.True(t, result.Success)
		assert.NotNil(t, exec.response.Body)
	})
}
