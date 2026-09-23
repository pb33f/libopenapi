// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"context"
	"errors"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

const arazzo11ModelYAML = `arazzo: 1.1.0
$self: https://example.com/workflows/root.arazzo.yaml
info:
  title: Arazzo 1.1
  version: 1.0.0
sourceDescriptions:
  - name: messages
    url: asyncapi.yaml
    type: asyncapi
workflows:
  - workflowId: publish
    steps:
      - stepId: send
        channelPath: $sourceDescriptions.messages#/channels/orders
        action: send
        correlationId: order-id
        timeout: 2500
        dependsOn: [prepare]
        requestBody:
          contentType: application/json
          payload:
            id: old
          replacements:
            - target: $.id
              targetSelectorType:
                type: jsonpath
                version: rfc9535
              value:
                context: $inputs
                selector: $.id
                type: jsonpath
        onSuccess:
          - name: done
            type: end
            parameters:
              - name: result
                value: $outputs.result
        onFailure:
          - name: retry
            type: retry
            retryLimit: 1
            parameters:
              - name: reason
                value: failed
        outputs:
          selected:
            context: $response.body
            selector: $.id
            type:
              type: jsonpath
              version: rfc9535
    outputs:
      status: $steps.send.outputs.selected
`

func buildLowArazzo11(t *testing.T) *Arazzo {
	t.Helper()
	var document yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(arazzo11ModelYAML), &document))
	root := document.Content[0]
	model := new(Arazzo)
	require.NoError(t, low.BuildModel(root, model))
	require.NoError(t, model.Build(context.Background(), nil, root, nil))
	return model
}

func TestArazzo11LowModel_AllFieldsAndUnions(t *testing.T) {
	document := buildLowArazzo11(t)
	assert.Equal(t, "https://example.com/workflows/root.arazzo.yaml", document.Self.Value)
	assert.NotZero(t, document.Hash())

	workflow := document.Workflows.Value[0].Value
	step := workflow.Steps.Value[0].Value
	assert.Equal(t, "$sourceDescriptions.messages#/channels/orders", step.ChannelPath.Value)
	assert.Equal(t, "send", step.Action.Value)
	assert.Equal(t, "order-id", step.CorrelationId.Value)
	assert.EqualValues(t, 2500, step.Timeout.Value)
	require.Len(t, step.DependsOn.Value, 1)
	assert.Equal(t, "prepare", step.DependsOn.Value[0].Value)
	require.Len(t, step.OnSuccess.Value[0].Value.Parameters.Value, 1)
	require.Len(t, step.OnFailure.Value[0].Value.Parameters.Value, 1)

	replacement := step.RequestBody.Value.Replacements.Value[0].Value
	require.Equal(t, yaml.MappingNode, replacement.TargetSelectorType.Value.Kind)
	assert.NotZero(t, replacement.Hash())

	stepOutput := step.Outputs.Value.First().Value().Value
	assert.True(t, stepOutput.IsSelector())
	assert.False(t, stepOutput.IsExpression())
	assert.Same(t, step.Outputs.Value.First().Key().KeyNode, stepOutput.GetKeyNode())
	assert.Same(t, step.Outputs.Value.First().Value().ValueNode, stepOutput.GetRootNode())
	assert.Equal(t, context.Background(), stepOutput.GetContext())
	assert.Nil(t, stepOutput.GetIndex())
	assert.NotZero(t, stepOutput.Hash())

	selector := stepOutput.Selector.Value
	assert.Equal(t, "$response.body", selector.Context.Value)
	assert.Equal(t, "$.id", selector.Selector.Value)
	assert.NotNil(t, selector.GetContext())
	assert.Nil(t, selector.GetIndex())
	assert.Same(t, selector.RootNode, selector.GetRootNode())
	assert.Same(t, selector.KeyNode, selector.GetKeyNode())
	assert.Nil(t, selector.FindExtension("x-missing"))
	assert.Equal(t, selector.Extensions, selector.GetExtensions())
	assert.NotZero(t, selector.Hash())

	workflowOutput := workflow.Outputs.Value.First().Value().Value
	assert.True(t, workflowOutput.IsExpression())
	assert.False(t, workflowOutput.IsSelector())
	assert.Equal(t, "$steps.send.outputs.selected", workflowOutput.Expression.Value)
	assert.NotZero(t, workflowOutput.Hash())
}

func TestOutputValue_BuildRejectsUnexpectedNodeKind(t *testing.T) {
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("- one\n- two"), &node))
	output := new(OutputValue)
	err := output.Build(context.Background(), nil, node.Content[0], nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be a scalar or Selector Object")
}

func TestOutputValue_BuildPropagatesMalformedSelectorType(t *testing.T) {
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("context: $response.body\nselector: $.id\ntype: [jsonpath]"), &node))
	output := new(OutputValue)
	err := output.Build(context.Background(), nil, node.Content[0], nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "selector type")
}

func TestSelector_BuildScalarTypeAndExtensionHash(t *testing.T) {
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("context: $inputs\nselector: /id\ntype: jsonpointer\nx-note: yes"), &node))
	selector := new(Selector)
	require.NoError(t, low.BuildModel(node.Content[0], selector))
	require.NoError(t, selector.Build(context.Background(), nil, node.Content[0], nil))
	require.NotNil(t, selector.FindExtension("x-note"))
	firstHash := selector.Hash()
	selector.Context.Value = "$response.body"
	assert.NotEqual(t, firstHash, selector.Hash())
}

func TestSelector_BuildRejectsSequenceType(t *testing.T) {
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("context: $inputs\nselector: $.id\ntype: [jsonpath]"), &node))
	selector := new(Selector)
	require.NoError(t, low.BuildModel(node.Content[0], selector))
	err := selector.Build(context.Background(), nil, node.Content[0], nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "line 3")
}

func TestExtractOutputValuesMap_AbsentAndWrongKind(t *testing.T) {
	var absent yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("name: value"), &absent))
	outputs, err := extractOutputValuesMap(context.Background(), OutputsLabel, absent.Content[0], nil)
	require.NoError(t, err)
	assert.True(t, outputs.IsEmpty())

	var scalar yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("outputs: invalid"), &scalar))
	outputs, err = extractOutputValuesMap(context.Background(), OutputsLabel, scalar.Content[0], nil)
	require.Error(t, err)
	assert.Nil(t, outputs.Value)
	assert.NotNil(t, outputs.ValueNode)

	oddOutputs := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "orphan"},
		},
	}
	oddRoot := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: OutputsLabel},
			oddOutputs,
		},
	}
	outputs, err = extractOutputValuesMap(context.Background(), OutputsLabel, oddRoot, nil)
	require.NoError(t, err)
	require.NotNil(t, outputs.Value)
	assert.Equal(t, 0, outputs.Value.Len())

	var malformed yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("outputs:\n  bad: [one]"), &malformed))
	_, err = extractOutputValuesMap(context.Background(), OutputsLabel, malformed.Content[0], nil)
	require.Error(t, err)
}

func TestArazzo11HashChangesForNewFields(t *testing.T) {
	document := buildLowArazzo11(t)
	base := document.Hash()
	document.Self.Value = "https://example.com/changed"
	assert.NotEqual(t, base, document.Hash())

	step := document.Workflows.Value[0].Value.Steps.Value[0].Value
	mutations := []func(){
		func() { step.ChannelPath.Value = "changed-channel" },
		func() { step.Action.Value = "receive" },
		func() { step.CorrelationId.Value = "changed-correlation" },
		func() { step.Timeout.Value++ },
		func() { step.DependsOn.Value[0].Value = "changed-dependency" },
	}
	for _, mutate := range mutations {
		before := step.Hash()
		mutate()
		assert.NotEqual(t, before, step.Hash())
	}
}

func TestArazzo11NestedHashesCoverEveryNewField(t *testing.T) {
	t.Run("selector fields and step output", func(t *testing.T) {
		document := buildLowArazzo11(t)
		step := document.Workflows.Value[0].Value.Steps.Value[0].Value
		selector := step.Outputs.Value.First().Value().Value.Selector.Value

		beforeSelector := selector.Hash()
		beforeStep := step.Hash()
		selector.Selector.Value = "$.changed"
		assert.NotEqual(t, beforeSelector, selector.Hash())
		assert.NotEqual(t, beforeStep, step.Hash())

		beforeSelector = selector.Hash()
		selector.Type.Value.Content[1].Value = "xpath"
		assert.NotEqual(t, beforeSelector, selector.Hash())
	})

	t.Run("target selector type and request body", func(t *testing.T) {
		document := buildLowArazzo11(t)
		step := document.Workflows.Value[0].Value.Steps.Value[0].Value
		replacement := step.RequestBody.Value.Replacements.Value[0].Value
		beforeReplacement := replacement.Hash()
		beforeRequestBody := step.RequestBody.Value.Hash()
		beforeStep := step.Hash()
		replacement.TargetSelectorType.Value.Content[3].Value = "draft-goessner-dispatch-jsonpath-00"
		assert.NotEqual(t, beforeReplacement, replacement.Hash())
		assert.NotEqual(t, beforeRequestBody, step.RequestBody.Value.Hash())
		assert.NotEqual(t, beforeStep, step.Hash())
	})

	t.Run("action parameters", func(t *testing.T) {
		document := buildLowArazzo11(t)
		step := document.Workflows.Value[0].Value.Steps.Value[0].Value
		success := step.OnSuccess.Value[0].Value
		beforeSuccess := success.Hash()
		beforeStep := step.Hash()
		success.Parameters.Value[0].Value.Name.Value = "changed-success"
		assert.NotEqual(t, beforeSuccess, success.Hash())
		assert.NotEqual(t, beforeStep, step.Hash())

		failure := step.OnFailure.Value[0].Value
		beforeFailure := failure.Hash()
		beforeStep = step.Hash()
		failure.Parameters.Value[0].Value.Name.Value = "changed-failure"
		assert.NotEqual(t, beforeFailure, failure.Hash())
		assert.NotEqual(t, beforeStep, step.Hash())
	})

	t.Run("workflow output", func(t *testing.T) {
		document := buildLowArazzo11(t)
		workflow := document.Workflows.Value[0].Value
		output := workflow.Outputs.Value.First().Value().Value
		beforeOutput := output.Hash()
		beforeWorkflow := workflow.Hash()
		output.Expression.Value = "$statusCode"
		assert.NotEqual(t, beforeOutput, output.Hash())
		assert.NotEqual(t, beforeWorkflow, workflow.Hash())
	})
}

func TestArazzo11Build_PropagatesOutputUnionErrors(t *testing.T) {
	var stepNode yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("stepId: bad\noperationId: bad\noutputs:\n  invalid: [one]"), &stepNode))
	step := new(Step)
	require.NoError(t, low.BuildModel(stepNode.Content[0], step))
	require.Error(t, step.Build(context.Background(), nil, stepNode.Content[0], nil))

	var workflowNode yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("workflowId: bad\nsteps: []\noutputs:\n  invalid: [one]"), &workflowNode))
	workflow := new(Workflow)
	require.NoError(t, low.BuildModel(workflowNode.Content[0], workflow))
	require.Error(t, workflow.Build(context.Background(), nil, workflowNode.Content[0], nil))
}

func TestPayloadReplacement_BuildRejectsInvalidTargetSelectorType(t *testing.T) {
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("target: $.id\ntargetSelectorType: [jsonpath]\nvalue: 1"), &node))
	replacement := new(PayloadReplacement)
	require.NoError(t, low.BuildModel(node.Content[0], replacement))
	err := replacement.Build(context.Background(), nil, node.Content[0], nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "targetSelectorType")
}

func TestArazzo11ActionParameterExtractionErrors(t *testing.T) {
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("name: action\ntype: end"), &node))
	sentinel := errors.New("parameter extraction failed")

	originalSuccess := extractSuccessActionParameters
	extractSuccessActionParameters = func(
		context.Context,
		string,
		*yaml.Node,
		*index.SpecIndex,
	) (low.NodeReference[[]low.ValueReference[*Parameter]], error) {
		return low.NodeReference[[]low.ValueReference[*Parameter]]{}, sentinel
	}
	success := new(SuccessAction)
	require.NoError(t, low.BuildModel(node.Content[0], success))
	assert.ErrorIs(t, success.Build(context.Background(), nil, node.Content[0], nil), sentinel)
	extractSuccessActionParameters = originalSuccess

	originalFailure := extractFailureActionParameters
	extractFailureActionParameters = func(
		context.Context,
		string,
		*yaml.Node,
		*index.SpecIndex,
	) (low.NodeReference[[]low.ValueReference[*Parameter]], error) {
		return low.NodeReference[[]low.ValueReference[*Parameter]]{}, sentinel
	}
	failure := new(FailureAction)
	require.NoError(t, low.BuildModel(node.Content[0], failure))
	assert.ErrorIs(t, failure.Build(context.Background(), nil, node.Content[0], nil), sentinel)
	extractFailureActionParameters = originalFailure
}
