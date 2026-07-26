// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package expression

import (
	"testing"

	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestEvaluateArazzo11RuntimeSources(t *testing.T) {
	payload := buildYAMLNode(t, `items:
  - id: order-1
`)
	ctx := &Context{
		Self:            "https://example.com/workflow.yaml",
		RequestPayload:  payload,
		ResponseQuery:   map[string]string{"cursor": "next"},
		ResponsePath:    map[string]string{"id": "42"},
		ResponsePayload: payload,
		MessageHeaders:  map[string]string{"Correlation-ID": "abc"},
		MessageQuery:    map[string]string{"partition": "1"},
		MessagePath:     map[string]string{"topic": "orders"},
		MessageBody:     payload,
		MessagePayload:  payload,
		Inputs: map[string]any{
			"object":           map[string]any{"items": []any{map[string]any{"id": "input-1"}}},
			"node":             payload,
			"nested":           map[string]any{"member": map[string]any{"id": "nested-input"}},
			"nested.member.id": "exact-input",
		},
		Outputs: map[string]any{
			"object": map[string]any{"id": "output-1"},
			"nested": map[string]any{"id": "nested-output"},
		},
		Steps: map[string]*StepContext{
			"step": {
				Inputs:  map[string]any{"request": map[string]any{"id": "step-input"}},
				Outputs: map[string]any{"result": map[string]any{"id": "step-1"}},
			},
		},
		Workflows: map[string]*WorkflowContext{
			"flow": {
				Inputs:  map[string]any{"request": map[string]any{"id": "workflow-input"}},
				Outputs: map[string]any{"result": []any{map[string]any{"id": "workflow-output"}}},
			},
		},
		SourceDescs: map[string]*SourceDescContext{
			"source": {
				URL:        "https://example.com/openapi.yaml",
				Type:       "openapi",
				Operations: map[string]any{"get": "operation"},
				Workflows:  map[string]any{"flow": "workflow"},
				Fields:     map[string]any{"x-field": "field"},
			},
		},
		Components: &ComponentsContext{
			Parameters:     map[string]any{"parameter": "p"},
			SuccessActions: map[string]any{"success": "s"},
			FailureActions: map[string]any{"failure": "f"},
		},
	}

	tests := []struct {
		input string
		want  any
	}{
		{"$self", "https://example.com/workflow.yaml"},
		{"$request.payload#/items/0/id", "order-1"},
		{"$response.query.cursor", "next"},
		{"$response.path.id", "42"},
		{"$response.payload#/items/0/id", "order-1"},
		{"$message.header.Correlation-ID", "abc"},
		{"$message.query.partition", "1"},
		{"$message.path.topic", "orders"},
		{"$message.body#/items/0/id", "order-1"},
		{"$message.payload#/items/0/id", "order-1"},
		{"$inputs.object#/items/0/id", "input-1"},
		{"$inputs.node#/items/0/id", "order-1"},
		{"$inputs.nested.member.id", "exact-input"},
		{"$outputs.object#/id", "output-1"},
		{"$outputs.nested.id", "nested-output"},
		{"$steps.step.outputs.result#/id", "step-1"},
		{"$steps.step.outputs.result.id", "step-1"},
		{"$workflows.flow.inputs.request#/id", "workflow-input"},
		{"$workflows.flow.inputs.request.id", "workflow-input"},
		{"$workflows.flow.outputs.result#/0/id", "workflow-output"},
		{"$sourceDescriptions.source.get", "operation"},
		{"$sourceDescriptions.source.flow", "workflow"},
		{"$sourceDescriptions.source.x-field", "field"},
		{"$sourceDescriptions.source.url", "https://example.com/openapi.yaml"},
		{"$sourceDescriptions.source.type", "openapi"},
		{"$components.parameters.parameter", "p"},
		{"$components.successActions.success", "s"},
		{"$components.failureActions.failure", "f"},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			value, err := EvaluateString(test.input, ctx)
			require.NoError(t, err)
			assert.Equal(t, test.want, value)
		})
	}
}

func TestEvaluateArazzo11RuntimeSourceErrors(t *testing.T) {
	payload := buildYAMLNode(t, "value: ok")
	tests := []struct {
		expr Expression
		ctx  *Context
	}{
		{expr: Expression{Type: Self}, ctx: &Context{}},
		{expr: Expression{Type: RequestPayload}, ctx: &Context{}},
		{expr: Expression{Type: ResponsePayload}, ctx: &Context{}},
		{expr: Expression{Type: MessageBody}, ctx: &Context{}},
		{expr: Expression{Type: MessagePayload}, ctx: &Context{}},
		{expr: Expression{Type: ResponseQuery, Property: "missing"}, ctx: &Context{ResponseQuery: map[string]string{}}},
		{expr: Expression{Type: ResponsePath, Property: "missing"}, ctx: &Context{ResponsePath: map[string]string{}}},
		{expr: Expression{Type: MessageHeader}, ctx: &Context{}},
		{expr: Expression{Type: MessageHeader, Property: "missing"}, ctx: &Context{MessageHeaders: map[string]string{}}},
		{expr: Expression{Type: MessageQuery}, ctx: &Context{}},
		{expr: Expression{Type: MessageQuery, Property: "missing"}, ctx: &Context{MessageQuery: map[string]string{}}},
		{expr: Expression{Type: MessagePath}, ctx: &Context{}},
		{expr: Expression{Type: MessagePath, Property: "missing"}, ctx: &Context{MessagePath: map[string]string{}}},
		{expr: Expression{Type: Inputs, Name: "value", JSONPointer: "relative"}, ctx: &Context{Inputs: map[string]any{"value": payload}}},
		{expr: Expression{Type: Inputs, Name: "value", JSONPointer: "/missing"}, ctx: &Context{Inputs: map[string]any{"value": map[string]any{}}}},
		{expr: Expression{Type: Inputs, Name: "value", JSONPointer: "/abc"}, ctx: &Context{Inputs: map[string]any{"value": []any{"x"}}}},
		{expr: Expression{Type: Inputs, Name: "value", JSONPointer: "/2"}, ctx: &Context{Inputs: map[string]any{"value": []any{"x"}}}},
		{expr: Expression{Type: Inputs, Name: "value", JSONPointer: "/child"}, ctx: &Context{Inputs: map[string]any{"value": "scalar"}}},
		{expr: Expression{Type: Inputs, Name: "value.child"}, ctx: &Context{Inputs: map[string]any{"value": "scalar"}}},
		{expr: Expression{Type: Outputs, Name: "value.child"}, ctx: &Context{Outputs: map[string]any{"value": "scalar"}}},
		{
			expr: Expression{Type: Steps, Name: "step", Tail: "outputs.value.child"},
			ctx:  &Context{Steps: map[string]*StepContext{"step": {Outputs: map[string]any{"value": "scalar"}}}},
		},
		{
			expr: Expression{Type: Steps, Name: "step", Tail: "inputs.value.child"},
			ctx:  &Context{Steps: map[string]*StepContext{"step": {Inputs: map[string]any{"value": "scalar"}}}},
		},
		{
			expr: Expression{Type: Workflows, Name: "flow", Tail: "outputs.value.child"},
			ctx:  &Context{Workflows: map[string]*WorkflowContext{"flow": {Outputs: map[string]any{"value": "scalar"}}}},
		},
		{
			expr: Expression{Type: Workflows, Name: "flow", Tail: "inputs.value.child"},
			ctx:  &Context{Workflows: map[string]*WorkflowContext{"flow": {Inputs: map[string]any{"value": "scalar"}}}},
		},
		{expr: Expression{Type: ComponentSuccessActions, Name: "value"}, ctx: &Context{}},
		{expr: Expression{Type: ComponentSuccessActions, Name: "value"}, ctx: &Context{Components: &ComponentsContext{}}},
		{expr: Expression{Type: ComponentSuccessActions, Name: "missing"}, ctx: &Context{Components: &ComponentsContext{SuccessActions: map[string]any{}}}},
		{expr: Expression{Type: ComponentFailureActions, Name: "value"}, ctx: &Context{Components: &ComponentsContext{}}},
		{expr: Expression{Type: ComponentFailureActions, Name: "missing"}, ctx: &Context{Components: &ComponentsContext{FailureActions: map[string]any{}}}},
	}
	for _, test := range tests {
		_, err := Evaluate(test.expr, test.ctx)
		assert.Error(t, err)
	}

	_, err := resolveComponentValue(&ComponentsContext{}, "value", "unknown")
	assert.Error(t, err)
	_, err = resolveJSONPointer(payload, "relative")
	assert.Error(t, err)
	_, err = resolveStructuredSource(payload, "relative", "payload")
	assert.Error(t, err)

	_, found, err := resolveNamedValue(map[string]any{"value": "scalar"}, "value.child")
	assert.True(t, found)
	assert.Error(t, err)
	_, found, err = resolveNamedValue(map[string]any{"value": map[string]any{}}, "value.child")
	assert.True(t, found)
	assert.Error(t, err)
	_, found, err = resolveNamedValue(map[string]any{"other": true}, "value.child")
	assert.False(t, found)
	assert.NoError(t, err)

	escapedNode := buildYAMLNode(t, `"a/b~c": escaped`)
	value, found, err := resolveNamedValue(map[string]any{"node": escapedNode}, "node.a/b~c")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "escaped", value)
	_, found, err = resolveNamedValue(map[string]any{"node": escapedNode}, "node.missing")
	assert.True(t, found)
	assert.Error(t, err)
}

func TestEvaluateArazzo11JSONPointerRootAndEmptyMember(t *testing.T) {
	node := buildYAMLNode(t, `"": empty
value: present
`)
	value, err := resolveJSONPointer(node, "/")
	require.NoError(t, err)
	assert.Equal(t, "empty", value)

	root, err := resolveJSONPointer(node, "")
	require.NoError(t, err)
	assert.Same(t, node, root)

	value, err = resolveValuePointer(map[string]any{"": "empty"}, "/")
	require.NoError(t, err)
	assert.Equal(t, "empty", value)

	rootValue := map[string]any{"value": "present"}
	value, err = resolveValuePointer(rootValue, "")
	require.NoError(t, err)
	assert.Equal(t, rootValue, value)
}
