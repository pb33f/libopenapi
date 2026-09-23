// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"context"
	"errors"
	"testing"

	"github.com/pb33f/libopenapi/arazzo/expression"
	high "github.com/pb33f/libopenapi/datamodel/high/arazzo"
	low "github.com/pb33f/libopenapi/datamodel/low/arazzo"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

func selectorOutputs(name string) *orderedmap.Map[string, *high.OutputValue] {
	outputs := orderedmap.New[string, *high.OutputValue]()
	outputs.Set(name, high.NewSelectorOutputValue(&high.Selector{
		Context:  "$response.body",
		Selector: "$.id",
		Type:     "jsonpath",
	}))
	return outputs
}

func TestEngine_RuntimeSelfUsesResolvedDocumentIdentity(t *testing.T) {
	document := high.NewArazzoWithOrigin(&low.Arazzo{}, &high.DocumentOrigin{
		AuthoredSelf:     "workflows/purchase.arazzo.yaml",
		ResolvedIdentity: "https://api.example.com/v2/workflows/purchase.arazzo.yaml",
	})
	document.Self = "workflows/purchase.arazzo.yaml"
	outputs := orderedmap.New[string, *high.OutputValue]()
	outputs.Set("identity", high.NewExpressionOutputValue("$self"))
	document.Workflows = []*high.Workflow{{WorkflowId: "purchase", Outputs: outputs}}

	result, err := NewEngine(document, nil, nil).RunWorkflow(context.Background(), "purchase", nil)
	require.NoError(t, err)
	assert.Equal(t, "https://api.example.com/v2/workflows/purchase.arazzo.yaml", result.Outputs["identity"])
}

func TestEngine_RunWorkflowPreflightsStepSelectorOutput(t *testing.T) {
	executor := &recordingExecutor{}
	document := &high.Arazzo{
		Workflows: []*high.Workflow{{
			WorkflowId: "root",
			Steps: []*high.Step{{
				StepId:      "call",
				OperationId: "get",
				Outputs:     selectorOutputs("selected"),
			}},
		}},
	}
	engine := NewEngine(document, executor, nil)
	result, err := engine.RunWorkflow(context.Background(), "root", nil)
	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnsupportedSelectorOutput)
	var typed *UnsupportedSelectorOutputError
	require.ErrorAs(t, err, &typed)
	assert.Equal(t, "root", typed.WorkflowId)
	assert.Equal(t, "call", typed.StepId)
	assert.Equal(t, "selected", typed.OutputName)
	assert.Equal(t,
		`selector outputs are not supported by the execution engine: workflow "root" step "call" output "selected"`,
		typed.Error())
	assert.Empty(t, executor.operationIDs)
}

func TestEngine_RunWorkflowPreflightsWorkflowSelectorOutput(t *testing.T) {
	executor := &recordingExecutor{}
	document := &high.Arazzo{
		Workflows: []*high.Workflow{{
			WorkflowId: "root",
			Steps:      []*high.Step{{StepId: "call", OperationId: "get"}},
			Outputs:    selectorOutputs("selected"),
		}},
	}
	engine := NewEngine(document, executor, nil)
	_, err := engine.RunWorkflow(context.Background(), "root", nil)
	require.Error(t, err)
	assert.Equal(t,
		`selector outputs are not supported by the execution engine: workflow "root" output "selected"`,
		err.Error())
	assert.Empty(t, executor.operationIDs)
}

func TestEngine_RunWorkflowPreflightsReachableNestedWorkflow(t *testing.T) {
	executor := &recordingExecutor{}
	document := &high.Arazzo{
		Workflows: []*high.Workflow{
			{
				WorkflowId: "root",
				Steps:      []*high.Step{{StepId: "nested", WorkflowId: "child"}},
			},
			{
				WorkflowId: "child",
				Steps: []*high.Step{{
					StepId:      "call",
					OperationId: "get",
					Outputs:     selectorOutputs("selected"),
				}},
			},
		},
	}
	engine := NewEngine(document, executor, nil)
	_, err := engine.RunWorkflow(context.Background(), "root", nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnsupportedSelectorOutput)
	assert.Empty(t, executor.operationIDs)
}

func TestEngine_RunWorkflowPreflightsActionTargetWorkflows(t *testing.T) {
	t.Run("direct success action", func(t *testing.T) {
		executor := &recordingExecutor{}
		document := &high.Arazzo{
			Workflows: []*high.Workflow{
				{
					WorkflowId: "root",
					Steps: []*high.Step{{
						StepId:      "call",
						OperationId: "get",
						OnSuccess: []*high.SuccessAction{{
							Name:       "child",
							Type:       "goto",
							WorkflowId: "child",
						}},
					}},
				},
				{
					WorkflowId: "child",
					Steps:      []*high.Step{{StepId: "child-call", OperationId: "get"}},
					Outputs:    selectorOutputs("selected"),
				},
			},
		}
		engine := NewEngine(document, executor, nil)
		_, err := engine.RunWorkflow(context.Background(), "root", nil)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrUnsupportedSelectorOutput)
		assert.Empty(t, executor.operationIDs)
	})

	t.Run("reusable failure action", func(t *testing.T) {
		executor := &recordingExecutor{}
		actions := orderedmap.New[string, *high.FailureAction]()
		actions.Set("child", &high.FailureAction{
			Name:       "child",
			Type:       "goto",
			WorkflowId: "child",
		})
		document := &high.Arazzo{
			Components: &high.Components{FailureActions: actions},
			Workflows: []*high.Workflow{
				{
					WorkflowId: "root",
					Steps: []*high.Step{{
						StepId:      "call",
						OperationId: "get",
						OnFailure: []*high.FailureAction{{
							Reference: "$components.failureActions.child",
						}},
					}},
				},
				{
					WorkflowId: "child",
					Steps:      []*high.Step{{StepId: "child-call", OperationId: "get"}},
					Outputs:    selectorOutputs("selected"),
				},
			},
		}
		engine := NewEngine(document, executor, nil)
		_, err := engine.RunWorkflow(context.Background(), "root", nil)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrUnsupportedSelectorOutput)
		assert.Empty(t, executor.operationIDs)
	})

	// Preflight looks for selector outputs, not broken references. An action whose reusable
	// reference cannot be resolved is skipped: it contributes no reachable workflow, and it
	// must not stop a document that would otherwise run. A dangling onFailure reference in
	// particular is never consulted at all unless the step actually fails.
	t.Run("unresolvable reusable action does not block the run", func(t *testing.T) {
		tests := []struct {
			name      string
			configure func(*high.Workflow, *high.Step)
		}{
			{
				name: "step success action",
				configure: func(_ *high.Workflow, step *high.Step) {
					step.OnSuccess = []*high.SuccessAction{{
						Reference: "$components.successActions.missing",
					}}
				},
			},
			{
				name: "step failure action",
				configure: func(_ *high.Workflow, step *high.Step) {
					step.OnFailure = []*high.FailureAction{{
						Reference: "$components.failureActions.missing",
					}}
				},
			},
			{
				name: "workflow success action",
				configure: func(workflow *high.Workflow, _ *high.Step) {
					workflow.SuccessActions = []*high.SuccessAction{{
						Reference: "$components.successActions.missing",
					}}
				},
			},
			{
				name: "workflow failure action",
				configure: func(workflow *high.Workflow, _ *high.Step) {
					workflow.FailureActions = []*high.FailureAction{{
						Reference: "$components.failureActions.missing",
					}}
				},
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				executor := &recordingExecutor{}
				step := &high.Step{StepId: "call", OperationId: "get"}
				workflow := &high.Workflow{WorkflowId: "root", Steps: []*high.Step{step}}
				test.configure(workflow, step)
				engine := NewEngine(&high.Arazzo{Workflows: []*high.Workflow{workflow}}, executor, nil)
				_, err := engine.RunWorkflow(context.Background(), "root", nil)

				// Whatever happens once execution starts, preflight must not have
				// short-circuited it: the step has to have reached the executor.
				assert.Equal(t, []string{"get"}, executor.operationIDs,
					"preflight must not reject a document over an unresolvable action reference")
				assert.NotErrorIs(t, err, ErrUnsupportedSelectorOutput,
					"there is no selector output here")
			})
		}
	})
}

func TestEngine_RunAllPreflightsAllWorkflows(t *testing.T) {
	executor := &recordingExecutor{}
	document := &high.Arazzo{
		Workflows: []*high.Workflow{
			{WorkflowId: "safe", Steps: []*high.Step{{StepId: "safe-call", OperationId: "get"}}},
			{WorkflowId: "unsupported", Steps: []*high.Step{{StepId: "bad", OperationId: "get"}}, Outputs: selectorOutputs("selected")},
		},
	}
	engine := NewEngine(document, executor, nil)
	result, err := engine.RunAll(context.Background(), nil)
	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnsupportedSelectorOutput)
	assert.Empty(t, executor.operationIDs)
}

func TestEngine_PreflightExecutionBoundaries(t *testing.T) {
	var engine *Engine
	assert.NoError(t, engine.preflightExecution())

	engine = NewEngine(nil, nil, nil)
	assert.NoError(t, engine.preflightExecution())
	assert.NoError(t, engine.preflightExecution("missing"))

	document := &high.Arazzo{
		Workflows: []*high.Workflow{nil, {
			WorkflowId: "cycle",
			Steps:      []*high.Step{nil, {StepId: "self", WorkflowId: "cycle"}},
		}},
	}
	engine = NewEngine(document, nil, nil)
	assert.NoError(t, engine.preflightExecution("cycle"))
}

func selectorValueNode() *yaml.Node {
	return &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{
		{Kind: yaml.ScalarNode, Value: "context"}, {Kind: yaml.ScalarNode, Value: "$response.body"},
		{Kind: yaml.ScalarNode, Value: "selector"}, {Kind: yaml.ScalarNode, Value: "$.id"},
		{Kind: yaml.ScalarNode, Value: "type"}, {Kind: yaml.ScalarNode, Value: "jsonpath"},
	}}
}

func TestEngine_PreflightsDeferredExecutionFeaturesBeforeExecutor(t *testing.T) {
	timeout := int64(1000)
	tests := []struct {
		name      string
		field     string
		configure func(*high.Workflow, *high.Step)
	}{
		{name: "channel step", field: "channelPath", configure: func(_ *high.Workflow, step *high.Step) {
			step.OperationId = ""
			step.ChannelPath = "/orders"
			step.Action = "send"
		}},
		{name: "action", field: "action", configure: func(_ *high.Workflow, step *high.Step) {
			step.OperationId = ""
			step.Action = "send"
		}},
		{name: "correlation id", field: "correlationId", configure: func(_ *high.Workflow, step *high.Step) {
			step.CorrelationId = "$message.header.X-Correlation-ID"
		}},
		{name: "timeout", field: "timeout", configure: func(_ *high.Workflow, step *high.Step) {
			step.Timeout = &timeout
		}},
		{name: "step dependency", field: "dependsOn", configure: func(_ *high.Workflow, step *high.Step) {
			step.DependsOn = []string{"earlier"}
		}},
		{name: "selector parameter", field: "parameters.value", configure: func(_ *high.Workflow, step *high.Step) {
			step.Parameters = []*high.Parameter{{Name: "id", In: "query", Value: selectorValueNode()}}
		}},
		{name: "selector payload", field: "requestBody.payload", configure: func(_ *high.Workflow, step *high.Step) {
			step.RequestBody = &high.RequestBody{Payload: selectorValueNode()}
		}},
		{name: "aliased selector payload", field: "requestBody.payload", configure: func(_ *high.Workflow, step *high.Step) {
			step.RequestBody = &high.RequestBody{Payload: &yaml.Node{Kind: yaml.AliasNode, Alias: selectorValueNode()}}
		}},
		{name: "selector replacement target", field: "requestBody.replacements.targetSelectorType", configure: func(_ *high.Workflow, step *high.Step) {
			step.RequestBody = &high.RequestBody{Replacements: []*high.PayloadReplacement{{TargetSelectorType: "jsonpath"}}}
		}},
		{name: "selector replacement value", field: "requestBody.replacements.value", configure: func(_ *high.Workflow, step *high.Step) {
			step.RequestBody = &high.RequestBody{Replacements: []*high.PayloadReplacement{nil, {Value: selectorValueNode()}}}
		}},
		{name: "action parameters", field: "successAction.parameters", configure: func(_ *high.Workflow, step *high.Step) {
			step.OnSuccess = []*high.SuccessAction{nil, {Name: "end", Type: "end", Parameters: []*high.Parameter{{Name: "id"}}}}
		}},
		{name: "failure action parameters", field: "failureAction.parameters", configure: func(_ *high.Workflow, step *high.Step) {
			step.OnFailure = []*high.FailureAction{nil, {Name: "end", Type: "end", Parameters: []*high.Parameter{{Name: "id"}}}}
		}},
		{name: "workflow selector parameter", field: "parameters.value", configure: func(workflow *high.Workflow, _ *high.Step) {
			workflow.Parameters = []*high.Parameter{nil, {Name: "id", Value: selectorValueNode()}}
		}},
		{name: "workflow success action parameters", field: "successAction.parameters", configure: func(workflow *high.Workflow, _ *high.Step) {
			workflow.SuccessActions = []*high.SuccessAction{nil, {Name: "end", Type: "end", Parameters: []*high.Parameter{{Name: "id"}}}}
		}},
		{name: "workflow failure action parameters", field: "failureAction.parameters", configure: func(workflow *high.Workflow, _ *high.Step) {
			workflow.FailureActions = []*high.FailureAction{nil, {Name: "end", Type: "end", Parameters: []*high.Parameter{{Name: "id"}}}}
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			executor := &recordingExecutor{}
			step := &high.Step{StepId: "call", OperationId: "get"}
			workflow := &high.Workflow{WorkflowId: "root", Steps: []*high.Step{step}}
			test.configure(workflow, step)
			engine := NewEngine(&high.Arazzo{Arazzo: "1.1.0", Workflows: []*high.Workflow{workflow}}, executor, nil)

			result, err := engine.RunWorkflow(context.Background(), "root", nil)
			assert.Nil(t, result)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrUnsupportedExecutionFeature)
			var typed *UnsupportedExecutionFeatureError
			require.ErrorAs(t, err, &typed)
			assert.Equal(t, test.field, typed.Field)
			assert.Contains(t, typed.Error(), `workflow "root"`)
			if typed.StepId == "" {
				assert.NotContains(t, typed.Error(), " step ")
			} else {
				assert.Contains(t, typed.Error(), `step "call"`)
			}
			assert.Empty(t, executor.operationIDs)
		})
	}
}

func TestEngine_PopulateOutputsRejectsInvalidUnionDefensively(t *testing.T) {
	engine := NewEngine(nil, nil, nil)
	outputs := orderedmap.New[string, *high.OutputValue]()
	outputs.Set("invalid", nil)

	stepResult := &StepResult{Outputs: make(map[string]any)}
	err := engine.populateStepOutputs(
		&high.Step{StepId: "step", Outputs: outputs},
		&high.Workflow{WorkflowId: "workflow"},
		stepResult,
		&expression.Context{},
	)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnsupportedSelectorOutput))
	assert.Empty(t, stepResult.Outputs)

	workflowResult := &WorkflowResult{Outputs: make(map[string]any)}
	err = engine.populateWorkflowOutputs(
		&high.Workflow{WorkflowId: "workflow", Outputs: outputs},
		workflowResult,
		&expression.Context{Outputs: make(map[string]any)},
	)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnsupportedSelectorOutput)
	assert.Empty(t, workflowResult.Outputs)
}

// A selector output reached through the direct populate path (rather than preflight)
// must still name the workflow it belongs to, so callers can locate the offending output.
func TestEngine_StepSelectorOutputErrorIdentifiesWorkflow(t *testing.T) {
	engine := NewEngine(nil, nil, nil)
	outputs := orderedmap.New[string, *high.OutputValue]()
	outputs.Set("selectorOutput", nil)

	err := engine.populateStepOutputs(
		&high.Step{StepId: "step-one", Outputs: outputs},
		&high.Workflow{WorkflowId: "workflow-one"},
		&StepResult{Outputs: make(map[string]any)},
		&expression.Context{},
	)
	require.Error(t, err)

	var unsupported *UnsupportedSelectorOutputError
	require.True(t, errors.As(err, &unsupported))
	assert.Equal(t, "workflow-one", unsupported.WorkflowId)
	assert.Equal(t, "step-one", unsupported.StepId)
	assert.Equal(t, "selectorOutput", unsupported.OutputName)
}

// Outputs are committed only after all of them evaluate. An expression that fails partway
// through must leave no partially populated map behind, because a step's outputs are
// published into the expression context and would otherwise be visible to later steps.
func TestEngine_StepOutputsAreNotPartiallyPopulatedOnEvaluationFailure(t *testing.T) {
	engine := NewEngine(nil, nil, nil)
	outputs := orderedmap.New[string, *high.OutputValue]()
	outputs.Set("good", high.NewExpressionOutputValue("$statusCode"))
	outputs.Set("bad", high.NewExpressionOutputValue("$steps.nonexistent.outputs.missing"))

	stepResult := &StepResult{Outputs: make(map[string]any)}
	err := engine.populateStepOutputs(
		&high.Step{StepId: "step", Outputs: outputs},
		&high.Workflow{WorkflowId: "workflow"},
		stepResult,
		&expression.Context{StatusCode: 200, Steps: map[string]*expression.StepContext{}},
	)
	require.Error(t, err)
	assert.Empty(t, stepResult.Outputs,
		"the earlier successful output must not be published when a later one fails")
}

// The same atomicity applies to workflow outputs, which are written into both the result
// and the expression context.
func TestEngine_WorkflowOutputsAreNotPartiallyPopulatedOnEvaluationFailure(t *testing.T) {
	engine := NewEngine(nil, nil, nil)
	outputs := orderedmap.New[string, *high.OutputValue]()
	outputs.Set("good", high.NewExpressionOutputValue("$statusCode"))
	outputs.Set("bad", high.NewExpressionOutputValue("$steps.nonexistent.outputs.missing"))

	workflowResult := &WorkflowResult{Outputs: make(map[string]any)}
	exprCtx := &expression.Context{
		StatusCode: 200,
		Steps:      map[string]*expression.StepContext{},
		Outputs:    make(map[string]any),
	}

	err := engine.populateWorkflowOutputs(
		&high.Workflow{WorkflowId: "workflow", Outputs: outputs},
		workflowResult,
		exprCtx,
	)
	require.Error(t, err)
	assert.Empty(t, workflowResult.Outputs)
	assert.Empty(t, exprCtx.Outputs,
		"a failed workflow output set must not leak into the expression context")
}
