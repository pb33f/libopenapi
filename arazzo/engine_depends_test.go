// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
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
	"go.yaml.in/yaml/v4"
)

func TestApplyPayloadReplacements_EvaluatesExpressionTarget(t *testing.T) {
	engine := NewEngine(&high.Arazzo{}, nil, nil)
	root := map[string]any{
		"metadata": map[string]any{
			"assessment": map[string]any{
				"alerts": map[string]any{},
			},
		},
	}
	exprCtx := &expression.Context{
		Workflows: map[string]*expression.WorkflowContext{
			"registerAlertName": {
				Outputs: map[string]any{"nameSlug": "run-1-pressure"},
			},
		},
	}

	result, err := engine.applyPayloadReplacements(root, []*high.PayloadReplacement{{
		Target: "/metadata/assessment/alerts/{$workflows.registerAlertName.outputs.nameSlug}",
		Value:  mustYAMLMapping(t, "severity: 1\n"),
	}}, exprCtx, "uploadPressureEvidence")
	require.NoError(t, err)

	body, ok := result.(map[string]any)
	require.True(t, ok)
	alerts := body["metadata"].(map[string]any)["assessment"].(map[string]any)["alerts"].(map[string]any)
	got, ok := alerts["run-1-pressure"].(map[string]any)
	require.True(t, ok)
	assert.EqualValues(t, 1, got["severity"])
}

func TestApplyPayloadReplacements_TargetMustEvaluateToString(t *testing.T) {
	engine := NewEngine(&high.Arazzo{}, nil, nil)
	_, err := engine.applyPayloadReplacements(map[string]any{}, []*high.PayloadReplacement{{
		Target: "$inputs.n",
		Value:  makeValueNode("x"),
	}}, &expression.Context{Inputs: map[string]any{"n": 7}}, "s1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `replacement target "$inputs.n" in step "s1" evaluated to int, want string`)
}

func TestEngine_RunWorkflow_RunsDependsOnAndResolvesReplacementTarget(t *testing.T) {
	stepOutputs := orderedmap.New[string, string]()
	stepOutputs.Set("slug", "$response.body#/slug")
	workflowOutputs := orderedmap.New[string, string]()
	workflowOutputs.Set("nameSlug", "$steps.create.outputs.slug")

	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{
				WorkflowId: "registerAlertName",
				Steps: []*high.Step{{
					StepId:      "create",
					OperationId: "register",
					Outputs:     stepOutputs,
				}},
				Outputs: workflowOutputs,
			},
			{
				WorkflowId: "raiseAlertingEvidence",
				DependsOn:  []string{"registerAlertName"},
				Steps: []*high.Step{{
					StepId:      "upload",
					OperationId: "upload",
					RequestBody: &high.RequestBody{
						ContentType: "application/json",
						Payload:     mustYAMLMapping(t, "metadata:\n  assessment:\n    alerts: {}\n"),
						Replacements: []*high.PayloadReplacement{{
							Target: "/metadata/assessment/alerts/{$workflows.registerAlertName.outputs.nameSlug}",
							Value:  mustYAMLMapping(t, "severity: 1\n"),
						}},
					},
				}},
			},
		},
	}

	executor := &bodyByOperationExecutor{
		bodies: map[string]any{
			"register": map[string]any{"slug": "run-1728-pressure"},
			"upload":   map[string]any{"id": "ev-1"},
		},
	}
	engine := NewEngine(doc, executor, nil)

	result, err := engine.RunWorkflow(context.Background(), "raiseAlertingEvidence", map[string]any{"runId": "1728"})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	require.Equal(t, []string{"register", "upload"}, executor.operationIDs)
	require.Len(t, executor.requests, 2)

	body, ok := executor.requests[1].RequestBody.(map[string]any)
	require.True(t, ok)
	alerts := body["metadata"].(map[string]any)["assessment"].(map[string]any)["alerts"].(map[string]any)
	got, ok := alerts["run-1728-pressure"].(map[string]any)
	require.True(t, ok, "alerts=%v", alerts)
	assert.EqualValues(t, 1, got["severity"])
}

func TestEngine_RunWorkflow_AbortsWhenDependencyFails(t *testing.T) {
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{
				WorkflowId: "registerAlertName",
				Steps: []*high.Step{{
					StepId:      "create",
					OperationId: "register",
					SuccessCriteria: []*high.Criterion{{
						Condition: "$statusCode == 201",
					}},
				}},
			},
			{
				WorkflowId: "raiseAlertingEvidence",
				DependsOn:  []string{"registerAlertName"},
				Steps: []*high.Step{{
					StepId:      "upload",
					OperationId: "upload",
				}},
			},
		},
	}
	executor := &statusRecordingExecutor{
		statusByOperation: map[string]int{"register": 500, "upload": 201},
	}
	engine := NewEngine(doc, executor, nil)

	result, err := engine.RunWorkflow(context.Background(), "raiseAlertingEvidence", nil)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorContains(t, err, `dependency "registerAlertName" failed`)
	assert.Equal(t, []string{"register"}, executor.operationIDs)
}

func TestEngine_RunWorkflow_DependsOnCycle(t *testing.T) {
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{
				WorkflowId: "wf1",
				DependsOn:  []string{"wf2"},
				Steps:      []*high.Step{{StepId: "s1", OperationId: "op1"}},
			},
			{
				WorkflowId: "wf2",
				DependsOn:  []string{"wf1"},
				Steps:      []*high.Step{{StepId: "s2", OperationId: "op2"}},
			},
		},
	}
	executor := &recordingExecutor{}
	engine := NewEngine(doc, executor, nil)

	result, err := engine.RunWorkflow(context.Background(), "wf1", nil)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrCircularDependency)
	assert.Empty(t, executor.operationIDs)
}

type bodyByOperationExecutor struct {
	bodies       map[string]any
	operationIDs []string
	requests     []*ExecutionRequest
}

func (b *bodyByOperationExecutor) Execute(_ context.Context, req *ExecutionRequest) (*ExecutionResponse, error) {
	b.operationIDs = append(b.operationIDs, req.OperationID)
	b.requests = append(b.requests, req)
	return &ExecutionResponse{StatusCode: 201, Body: b.bodies[req.OperationID]}, nil
}

func mustYAMLMapping(t *testing.T, src string) *yaml.Node {
	t.Helper()
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(src), &node))
	require.NotEmpty(t, node.Content)
	return node.Content[0]
}
