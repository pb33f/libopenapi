// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package expression

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// buildYAMLNode unmarshals a YAML string into a *yaml.Node.
func buildYAMLNode(t *testing.T, src string) *yaml.Node {
	t.Helper()
	var node yaml.Node
	err := yaml.Unmarshal([]byte(src), &node)
	assert.NoError(t, err)
	return &node
}

// fullContext returns a Context populated with values for every field.
func fullContext(t *testing.T) *Context {
	t.Helper()
	return &Context{
		URL:        "https://api.example.com/pets",
		Method:     "GET",
		StatusCode: 200,
		RequestHeaders: map[string]string{
			"X-Api-Key":    "abc123",
			"Content-Type": "application/json",
		},
		RequestQuery: map[string]string{
			"page":  "1",
			"limit": "10",
		},
		RequestPath: map[string]string{
			"petId": "42",
		},
		RequestBody: buildYAMLNode(t, `name: Fido
age: 3
tags:
  - good
  - dog
data:
  - id: 100
    value: first
  - id: 200
    value: second
nested:
  a/b: slash
  a~c: tilde
`),
		ResponseHeaders: map[string]string{
			"Content-Type":  "application/json",
			"X-Request-Id":  "req-999",
		},
		ResponseBody: buildYAMLNode(t, `results:
  - id: 1
    name: Fido
  - id: 2
    name: Rex
total: 2
`),
		Inputs: map[string]any{
			"petId":   "42",
			"verbose": true,
		},
		Outputs: map[string]any{
			"result": "ok",
			"count":  5,
		},
		Steps: map[string]*StepContext{
			"getPet": {
				Inputs:  map[string]any{"id": "42"},
				Outputs: map[string]any{"petId": "pet-42", "name": "Fido"},
			},
			"emptyStep": {},
		},
		Workflows: map[string]*WorkflowContext{
			"getUser": {
				Inputs:  map[string]any{"userId": "u1"},
				Outputs: map[string]any{"name": "Alice", "role": "admin"},
			},
		},
		SourceDescs: map[string]*SourceDescContext{
			"petStore": {URL: "https://petstore.example.com/v1"},
		},
		Components: &ComponentsContext{
			Parameters:     map[string]any{"myParam": "paramValue"},
			SuccessActions: map[string]any{"retry": "3x"},
			FailureActions: map[string]any{"alert": "email"},
			Inputs:         map[string]any{"someInput": "inputValue"},
		},
	}
}

// ---------------------------------------------------------------------------
// Evaluate() -- each expression type with matching context
// ---------------------------------------------------------------------------

func TestEvaluate_URL(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: URL}, ctx)
	assert.NoError(t, err)
	assert.Equal(t, "https://api.example.com/pets", val)
}

func TestEvaluate_Method(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: Method}, ctx)
	assert.NoError(t, err)
	assert.Equal(t, "GET", val)
}

func TestEvaluate_StatusCode(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: StatusCode}, ctx)
	assert.NoError(t, err)
	assert.Equal(t, 200, val)
}

func TestEvaluate_RequestHeader(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: RequestHeader, Property: "X-Api-Key"}, ctx)
	assert.NoError(t, err)
	assert.Equal(t, "abc123", val)
}

func TestEvaluate_RequestQuery(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: RequestQuery, Property: "page"}, ctx)
	assert.NoError(t, err)
	assert.Equal(t, "1", val)
}

func TestEvaluate_RequestPath(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: RequestPath, Property: "petId"}, ctx)
	assert.NoError(t, err)
	assert.Equal(t, "42", val)
}

func TestEvaluate_RequestBody_NoPointer(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: RequestBody}, ctx)
	assert.NoError(t, err)
	assert.NotNil(t, val)
	// With no JSON pointer, we get the raw node
	_, ok := val.(*yaml.Node)
	assert.True(t, ok)
}

func TestEvaluate_RequestBody_Pointer(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: RequestBody, JSONPointer: "/name"}, ctx)
	assert.NoError(t, err)
	assert.Equal(t, "Fido", val)
}

func TestEvaluate_RequestBody_DeepPointer(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: RequestBody, JSONPointer: "/data/0/id"}, ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(100), val)
}

func TestEvaluate_ResponseHeader(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: ResponseHeader, Property: "Content-Type"}, ctx)
	assert.NoError(t, err)
	assert.Equal(t, "application/json", val)
}

func TestEvaluate_ResponseBody_Pointer(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: ResponseBody, JSONPointer: "/total"}, ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), val)
}

func TestEvaluate_ResponseBody_NoPointer(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: ResponseBody}, ctx)
	assert.NoError(t, err)
	assert.NotNil(t, val)
}

func TestEvaluate_Inputs(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: Inputs, Name: "petId"}, ctx)
	assert.NoError(t, err)
	assert.Equal(t, "42", val)
}

func TestEvaluate_Outputs(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: Outputs, Name: "result"}, ctx)
	assert.NoError(t, err)
	assert.Equal(t, "ok", val)
}

func TestEvaluate_Steps_OutputField(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: Steps, Name: "getPet", Tail: "outputs.petId"}, ctx)
	assert.NoError(t, err)
	assert.Equal(t, "pet-42", val)
}

func TestEvaluate_Steps_InputField(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: Steps, Name: "getPet", Tail: "inputs.id"}, ctx)
	assert.NoError(t, err)
	assert.Equal(t, "42", val)
}

func TestEvaluate_Steps_NoTail(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: Steps, Name: "getPet"}, ctx)
	assert.NoError(t, err)
	// Returns the StepContext itself
	sc, ok := val.(*StepContext)
	assert.True(t, ok)
	assert.NotNil(t, sc)
}

func TestEvaluate_Steps_AllOutputs(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: Steps, Name: "getPet", Tail: "outputs"}, ctx)
	assert.NoError(t, err)
	m, ok := val.(map[string]any)
	assert.True(t, ok)
	assert.Contains(t, m, "petId")
	assert.Contains(t, m, "name")
}

func TestEvaluate_Steps_AllInputs(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: Steps, Name: "getPet", Tail: "inputs"}, ctx)
	assert.NoError(t, err)
	m, ok := val.(map[string]any)
	assert.True(t, ok)
	assert.Contains(t, m, "id")
}

func TestEvaluate_Workflows_OutputField(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: Workflows, Name: "getUser", Tail: "outputs.name"}, ctx)
	assert.NoError(t, err)
	assert.Equal(t, "Alice", val)
}

func TestEvaluate_Workflows_InputField(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: Workflows, Name: "getUser", Tail: "inputs.userId"}, ctx)
	assert.NoError(t, err)
	assert.Equal(t, "u1", val)
}

func TestEvaluate_Workflows_NoTail(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: Workflows, Name: "getUser"}, ctx)
	assert.NoError(t, err)
	_, ok := val.(*WorkflowContext)
	assert.True(t, ok)
}

func TestEvaluate_SourceDescriptions_URL(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: SourceDescriptions, Name: "petStore", Tail: "url"}, ctx)
	assert.NoError(t, err)
	assert.Equal(t, "https://petstore.example.com/v1", val)
}

func TestEvaluate_SourceDescriptions_NoTail(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: SourceDescriptions, Name: "petStore"}, ctx)
	assert.NoError(t, err)
	_, ok := val.(*SourceDescContext)
	assert.True(t, ok)
}

func TestEvaluate_ComponentParameters(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: ComponentParameters, Name: "myParam"}, ctx)
	assert.NoError(t, err)
	assert.Equal(t, "paramValue", val)
}

func TestEvaluate_Components_Inputs(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: Components, Name: "inputs", Tail: "someInput"}, ctx)
	assert.NoError(t, err)
	assert.Equal(t, "inputValue", val)
}

func TestEvaluate_Components_SuccessActions(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: Components, Name: "successActions", Tail: "retry"}, ctx)
	assert.NoError(t, err)
	assert.Equal(t, "3x", val)
}

func TestEvaluate_Components_FailureActions(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: Components, Name: "failureActions", Tail: "alert"}, ctx)
	assert.NoError(t, err)
	assert.Equal(t, "email", val)
}

// ---------------------------------------------------------------------------
// Evaluate() -- missing context / error paths
// ---------------------------------------------------------------------------

func TestEvaluate_NilContext(t *testing.T) {
	_, err := Evaluate(Expression{Type: URL}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil context")
}

func TestEvaluate_Error_NilRequestHeaders(t *testing.T) {
	_, err := Evaluate(Expression{Type: RequestHeader, Property: "X-Api-Key"}, &Context{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no request headers")
}

func TestEvaluate_Error_MissingRequestHeader(t *testing.T) {
	ctx := &Context{RequestHeaders: map[string]string{"Accept": "text/html"}}
	_, err := Evaluate(Expression{Type: RequestHeader, Property: "X-Missing"}, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestEvaluate_Error_NilRequestQuery(t *testing.T) {
	_, err := Evaluate(Expression{Type: RequestQuery, Property: "page"}, &Context{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no request query")
}

func TestEvaluate_Error_MissingRequestQuery(t *testing.T) {
	ctx := &Context{RequestQuery: map[string]string{"limit": "5"}}
	_, err := Evaluate(Expression{Type: RequestQuery, Property: "offset"}, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestEvaluate_Error_NilRequestPath(t *testing.T) {
	_, err := Evaluate(Expression{Type: RequestPath, Property: "id"}, &Context{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no request path")
}

func TestEvaluate_Error_MissingRequestPath(t *testing.T) {
	ctx := &Context{RequestPath: map[string]string{"userId": "1"}}
	_, err := Evaluate(Expression{Type: RequestPath, Property: "orderId"}, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestEvaluate_Error_NilRequestBody(t *testing.T) {
	_, err := Evaluate(Expression{Type: RequestBody}, &Context{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no request body")
}

func TestEvaluate_Error_NilResponseHeaders(t *testing.T) {
	_, err := Evaluate(Expression{Type: ResponseHeader, Property: "X-Foo"}, &Context{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no response headers")
}

func TestEvaluate_Error_MissingResponseHeader(t *testing.T) {
	ctx := &Context{ResponseHeaders: map[string]string{"Accept": "json"}}
	_, err := Evaluate(Expression{Type: ResponseHeader, Property: "X-Missing"}, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestEvaluate_Error_NilResponseBody(t *testing.T) {
	_, err := Evaluate(Expression{Type: ResponseBody}, &Context{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no response body")
}

func TestEvaluate_Error_NilInputs(t *testing.T) {
	_, err := Evaluate(Expression{Type: Inputs, Name: "foo"}, &Context{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no inputs")
}

func TestEvaluate_Error_MissingInput(t *testing.T) {
	ctx := &Context{Inputs: map[string]any{"a": 1}}
	_, err := Evaluate(Expression{Type: Inputs, Name: "b"}, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestEvaluate_Error_NilOutputs(t *testing.T) {
	_, err := Evaluate(Expression{Type: Outputs, Name: "foo"}, &Context{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no outputs")
}

func TestEvaluate_Error_MissingOutput(t *testing.T) {
	ctx := &Context{Outputs: map[string]any{"x": 1}}
	_, err := Evaluate(Expression{Type: Outputs, Name: "y"}, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestEvaluate_Error_NilSteps(t *testing.T) {
	_, err := Evaluate(Expression{Type: Steps, Name: "s1"}, &Context{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no steps context")
}

func TestEvaluate_Error_MissingStep(t *testing.T) {
	ctx := &Context{Steps: map[string]*StepContext{"a": {}}}
	_, err := Evaluate(Expression{Type: Steps, Name: "b"}, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestEvaluate_Error_StepNoOutputs(t *testing.T) {
	ctx := &Context{Steps: map[string]*StepContext{"s": {}}}
	_, err := Evaluate(Expression{Type: Steps, Name: "s", Tail: "outputs.foo"}, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no outputs")
}

func TestEvaluate_Error_StepNoInputs(t *testing.T) {
	ctx := &Context{Steps: map[string]*StepContext{"s": {}}}
	_, err := Evaluate(Expression{Type: Steps, Name: "s", Tail: "inputs.bar"}, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no inputs")
}

func TestEvaluate_Error_StepMissingOutput(t *testing.T) {
	ctx := &Context{Steps: map[string]*StepContext{
		"s": {Outputs: map[string]any{"a": 1}},
	}}
	_, err := Evaluate(Expression{Type: Steps, Name: "s", Tail: "outputs.missing"}, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestEvaluate_Error_StepMissingInput(t *testing.T) {
	ctx := &Context{Steps: map[string]*StepContext{
		"s": {Inputs: map[string]any{"a": 1}},
	}}
	_, err := Evaluate(Expression{Type: Steps, Name: "s", Tail: "inputs.missing"}, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestEvaluate_Error_StepUnknownProperty(t *testing.T) {
	ctx := &Context{Steps: map[string]*StepContext{"s": {}}}
	_, err := Evaluate(Expression{Type: Steps, Name: "s", Tail: "unknown.prop"}, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown step property")
}

func TestEvaluate_Error_NilWorkflows(t *testing.T) {
	_, err := Evaluate(Expression{Type: Workflows, Name: "w"}, &Context{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no workflows context")
}

func TestEvaluate_Error_MissingWorkflow(t *testing.T) {
	ctx := &Context{Workflows: map[string]*WorkflowContext{"a": {}}}
	_, err := Evaluate(Expression{Type: Workflows, Name: "b"}, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestEvaluate_Error_WorkflowNoOutputs(t *testing.T) {
	ctx := &Context{Workflows: map[string]*WorkflowContext{"w": {}}}
	_, err := Evaluate(Expression{Type: Workflows, Name: "w", Tail: "outputs.foo"}, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no outputs")
}

func TestEvaluate_Error_WorkflowNoInputs(t *testing.T) {
	ctx := &Context{Workflows: map[string]*WorkflowContext{"w": {}}}
	_, err := Evaluate(Expression{Type: Workflows, Name: "w", Tail: "inputs.foo"}, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no inputs")
}

func TestEvaluate_Error_WorkflowMissingOutput(t *testing.T) {
	ctx := &Context{Workflows: map[string]*WorkflowContext{
		"w": {Outputs: map[string]any{"a": 1}},
	}}
	_, err := Evaluate(Expression{Type: Workflows, Name: "w", Tail: "outputs.missing"}, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestEvaluate_Error_WorkflowMissingInput(t *testing.T) {
	ctx := &Context{Workflows: map[string]*WorkflowContext{
		"w": {Inputs: map[string]any{"a": 1}},
	}}
	_, err := Evaluate(Expression{Type: Workflows, Name: "w", Tail: "inputs.missing"}, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestEvaluate_Error_WorkflowUnknownProperty(t *testing.T) {
	ctx := &Context{Workflows: map[string]*WorkflowContext{"w": {}}}
	_, err := Evaluate(Expression{Type: Workflows, Name: "w", Tail: "unknown"}, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown workflow property")
}

func TestEvaluate_Error_NilSourceDescs(t *testing.T) {
	_, err := Evaluate(Expression{Type: SourceDescriptions, Name: "sd"}, &Context{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no source descriptions")
}

func TestEvaluate_Error_MissingSourceDesc(t *testing.T) {
	ctx := &Context{SourceDescs: map[string]*SourceDescContext{"a": {}}}
	_, err := Evaluate(Expression{Type: SourceDescriptions, Name: "b"}, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestEvaluate_Error_SourceDescUnknownTail(t *testing.T) {
	ctx := &Context{SourceDescs: map[string]*SourceDescContext{"sd": {URL: "http://x"}}}
	_, err := Evaluate(Expression{Type: SourceDescriptions, Name: "sd", Tail: "unknown"}, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown source description property")
}

func TestEvaluate_Error_NilComponents(t *testing.T) {
	_, err := Evaluate(Expression{Type: Components, Name: "inputs", Tail: "foo"}, &Context{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no components")
}

func TestEvaluate_Error_ComponentsNoTail(t *testing.T) {
	ctx := &Context{Components: &ComponentsContext{}}
	_, err := Evaluate(Expression{Type: Components, Name: "inputs"}, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "incomplete components")
}

func TestEvaluate_Error_ComponentsUnknownType(t *testing.T) {
	ctx := &Context{Components: &ComponentsContext{}}
	_, err := Evaluate(Expression{Type: Components, Name: "unknown", Tail: "foo"}, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown component type")
}

func TestEvaluate_Error_NilComponentParameters(t *testing.T) {
	_, err := Evaluate(Expression{Type: ComponentParameters, Name: "p"}, &Context{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no component parameters")
}

func TestEvaluate_Error_MissingComponentParameter(t *testing.T) {
	ctx := &Context{Components: &ComponentsContext{Parameters: map[string]any{"a": 1}}}
	_, err := Evaluate(Expression{Type: ComponentParameters, Name: "b"}, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestEvaluate_Error_NilComponentsSuccessActions(t *testing.T) {
	ctx := &Context{Components: &ComponentsContext{}}
	_, err := Evaluate(Expression{Type: Components, Name: "successActions", Tail: "foo"}, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no component success actions")
}

func TestEvaluate_Error_NilComponentsFailureActions(t *testing.T) {
	ctx := &Context{Components: &ComponentsContext{}}
	_, err := Evaluate(Expression{Type: Components, Name: "failureActions", Tail: "foo"}, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no component failure actions")
}

func TestEvaluate_Error_NilComponentsInputs(t *testing.T) {
	ctx := &Context{Components: &ComponentsContext{}}
	_, err := Evaluate(Expression{Type: Components, Name: "inputs", Tail: "foo"}, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no component inputs")
}

func TestEvaluate_Error_MissingComponentSuccessAction(t *testing.T) {
	ctx := &Context{Components: &ComponentsContext{SuccessActions: map[string]any{"a": 1}}}
	_, err := Evaluate(Expression{Type: Components, Name: "successActions", Tail: "missing"}, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestEvaluate_Error_MissingComponentFailureAction(t *testing.T) {
	ctx := &Context{Components: &ComponentsContext{FailureActions: map[string]any{"a": 1}}}
	_, err := Evaluate(Expression{Type: Components, Name: "failureActions", Tail: "missing"}, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestEvaluate_Error_MissingComponentInput(t *testing.T) {
	ctx := &Context{Components: &ComponentsContext{Inputs: map[string]any{"a": 1}}}
	_, err := Evaluate(Expression{Type: Components, Name: "inputs", Tail: "missing"}, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestEvaluate_ResponseQuery_Unsupported(t *testing.T) {
	ctx := &Context{ResponseHeaders: map[string]string{"foo": "bar"}}
	_, err := Evaluate(Expression{Type: ResponseQuery, Property: "x"}, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

func TestEvaluate_ResponsePath_Unsupported(t *testing.T) {
	_, err := Evaluate(Expression{Type: ResponsePath, Property: "x"}, &Context{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

func TestEvaluate_UnsupportedExpressionType(t *testing.T) {
	_, err := Evaluate(Expression{Type: ExpressionType(999)}, &Context{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported expression type")
}

// ---------------------------------------------------------------------------
// JSON pointer resolution
// ---------------------------------------------------------------------------

func TestJSONPointer_ScalarString(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: RequestBody, JSONPointer: "/name"}, ctx)
	assert.NoError(t, err)
	assert.Equal(t, "Fido", val)
}

func TestJSONPointer_ScalarInt(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: RequestBody, JSONPointer: "/age"}, ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), val)
}

func TestJSONPointer_ArrayIndex(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: RequestBody, JSONPointer: "/tags/0"}, ctx)
	assert.NoError(t, err)
	assert.Equal(t, "good", val)
}

func TestJSONPointer_ArrayIndexSecond(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: RequestBody, JSONPointer: "/tags/1"}, ctx)
	assert.NoError(t, err)
	assert.Equal(t, "dog", val)
}

func TestJSONPointer_DeepNested(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: RequestBody, JSONPointer: "/data/1/value"}, ctx)
	assert.NoError(t, err)
	assert.Equal(t, "second", val)
}

func TestJSONPointer_MissingSegment(t *testing.T) {
	ctx := fullContext(t)
	_, err := Evaluate(Expression{Type: RequestBody, JSONPointer: "/nonexistent"}, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestJSONPointer_InvalidArrayIndex(t *testing.T) {
	ctx := fullContext(t)
	_, err := Evaluate(Expression{Type: RequestBody, JSONPointer: "/tags/abc"}, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid array index")
}

func TestJSONPointer_ArrayIndexOutOfBounds(t *testing.T) {
	ctx := fullContext(t)
	_, err := Evaluate(Expression{Type: RequestBody, JSONPointer: "/tags/99"}, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of bounds")
}

func TestJSONPointer_TraverseScalar(t *testing.T) {
	ctx := fullContext(t)
	_, err := Evaluate(Expression{Type: RequestBody, JSONPointer: "/name/deeper"}, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot traverse into scalar")
}

func TestJSONPointer_EscapedTilde0(t *testing.T) {
	// ~0 should unescape to ~
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: RequestBody, JSONPointer: "/nested/a~0c"}, ctx)
	assert.NoError(t, err)
	assert.Equal(t, "tilde", val)
}

func TestJSONPointer_EscapedTilde1(t *testing.T) {
	// ~1 should unescape to /
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: RequestBody, JSONPointer: "/nested/a~1b"}, ctx)
	assert.NoError(t, err)
	assert.Equal(t, "slash", val)
}

func TestJSONPointer_EmptyPointer(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: RequestBody, JSONPointer: ""}, ctx)
	assert.NoError(t, err)
	assert.NotNil(t, val)
}

func TestJSONPointer_RootSlash(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: RequestBody, JSONPointer: "/"}, ctx)
	assert.NoError(t, err)
	assert.NotNil(t, val)
}

func TestJSONPointer_ResponseBody(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: ResponseBody, JSONPointer: "/results/0/name"}, ctx)
	assert.NoError(t, err)
	assert.Equal(t, "Fido", val)
}

func TestJSONPointer_ResponseBody_ArrayIndex(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: ResponseBody, JSONPointer: "/results/1/id"}, ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), val)
}

// ---------------------------------------------------------------------------
// EvaluateString() -- parse + evaluate in one call
// ---------------------------------------------------------------------------

func TestEvaluateString_URL(t *testing.T) {
	ctx := fullContext(t)
	val, err := EvaluateString("$url", ctx)
	assert.NoError(t, err)
	assert.Equal(t, "https://api.example.com/pets", val)
}

func TestEvaluateString_Method(t *testing.T) {
	ctx := fullContext(t)
	val, err := EvaluateString("$method", ctx)
	assert.NoError(t, err)
	assert.Equal(t, "GET", val)
}

func TestEvaluateString_StatusCode(t *testing.T) {
	ctx := fullContext(t)
	val, err := EvaluateString("$statusCode", ctx)
	assert.NoError(t, err)
	assert.Equal(t, 200, val)
}

func TestEvaluateString_RequestHeader(t *testing.T) {
	ctx := fullContext(t)
	val, err := EvaluateString("$request.header.X-Api-Key", ctx)
	assert.NoError(t, err)
	assert.Equal(t, "abc123", val)
}

func TestEvaluateString_RequestBody_Pointer(t *testing.T) {
	ctx := fullContext(t)
	val, err := EvaluateString("$request.body#/name", ctx)
	assert.NoError(t, err)
	assert.Equal(t, "Fido", val)
}

func TestEvaluateString_Inputs(t *testing.T) {
	ctx := fullContext(t)
	val, err := EvaluateString("$inputs.petId", ctx)
	assert.NoError(t, err)
	assert.Equal(t, "42", val)
}

func TestEvaluateString_Steps(t *testing.T) {
	ctx := fullContext(t)
	val, err := EvaluateString("$steps.getPet.outputs.name", ctx)
	assert.NoError(t, err)
	assert.Equal(t, "Fido", val)
}

func TestEvaluateString_Workflows(t *testing.T) {
	ctx := fullContext(t)
	val, err := EvaluateString("$workflows.getUser.outputs.role", ctx)
	assert.NoError(t, err)
	assert.Equal(t, "admin", val)
}

func TestEvaluateString_SourceDescriptions(t *testing.T) {
	ctx := fullContext(t)
	val, err := EvaluateString("$sourceDescriptions.petStore.url", ctx)
	assert.NoError(t, err)
	assert.Equal(t, "https://petstore.example.com/v1", val)
}

func TestEvaluateString_ComponentParameters(t *testing.T) {
	ctx := fullContext(t)
	val, err := EvaluateString("$components.parameters.myParam", ctx)
	assert.NoError(t, err)
	assert.Equal(t, "paramValue", val)
}

func TestEvaluateString_ParseError(t *testing.T) {
	_, err := EvaluateString("notAnExpression", &Context{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must start with '$'")
}

func TestEvaluateString_NilContext(t *testing.T) {
	_, err := EvaluateString("$url", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil context")
}

// ---------------------------------------------------------------------------
// unescapeJSONPointer edge cases
// ---------------------------------------------------------------------------

func TestUnescapeJSONPointer_NoTilde(t *testing.T) {
	assert.Equal(t, "abc", UnescapeJSONPointer("abc"))
}

func TestUnescapeJSONPointer_Tilde0(t *testing.T) {
	assert.Equal(t, "a~c", UnescapeJSONPointer("a~0c"))
}

func TestUnescapeJSONPointer_Tilde1(t *testing.T) {
	assert.Equal(t, "a/c", UnescapeJSONPointer("a~1c"))
}

func TestUnescapeJSONPointer_Both(t *testing.T) {
	// ~0 -> ~, ~1 -> /
	assert.Equal(t, "~/", UnescapeJSONPointer("~0~1"))
}

func TestUnescapeJSONPointer_MultipleTilde1(t *testing.T) {
	assert.Equal(t, "a/b/c", UnescapeJSONPointer("a~1b~1c"))
}

// ---------------------------------------------------------------------------
// yamlNodeToValue edge cases
// ---------------------------------------------------------------------------

func TestYamlNodeToValue_Nil(t *testing.T) {
	assert.Nil(t, yamlNodeToValue(nil))
}

func TestYamlNodeToValue_BoolTrue(t *testing.T) {
	node := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"}
	assert.Equal(t, true, yamlNodeToValue(node))
}

func TestYamlNodeToValue_BoolFalse(t *testing.T) {
	node := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "false"}
	assert.Equal(t, false, yamlNodeToValue(node))
}

func TestYamlNodeToValue_Float(t *testing.T) {
	node := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!float", Value: "3.14"}
	val := yamlNodeToValue(node)
	assert.InDelta(t, 3.14, val, 0.001)
}

func TestYamlNodeToValue_Int(t *testing.T) {
	node := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: "42"}
	assert.Equal(t, int64(42), yamlNodeToValue(node))
}

func TestYamlNodeToValue_Null(t *testing.T) {
	node := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: ""}
	assert.Nil(t, yamlNodeToValue(node))
}

func TestYamlNodeToValue_String(t *testing.T) {
	node := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "hello"}
	assert.Equal(t, "hello", yamlNodeToValue(node))
}

func TestYamlNodeToValue_Mapping(t *testing.T) {
	node := &yaml.Node{Kind: yaml.MappingNode}
	val := yamlNodeToValue(node)
	assert.Equal(t, node, val)
}

func TestYamlNodeToValue_Sequence(t *testing.T) {
	node := &yaml.Node{Kind: yaml.SequenceNode}
	val := yamlNodeToValue(node)
	assert.Equal(t, node, val)
}

// ---------------------------------------------------------------------------
// Workflows -- all outputs / all inputs (no rest after segment)
// ---------------------------------------------------------------------------

func TestEvaluate_Workflows_AllOutputs(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: Workflows, Name: "getUser", Tail: "outputs"}, ctx)
	assert.NoError(t, err)
	m, ok := val.(map[string]any)
	assert.True(t, ok)
	assert.Contains(t, m, "name")
	assert.Contains(t, m, "role")
}

func TestEvaluate_Workflows_AllInputs(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: Workflows, Name: "getUser", Tail: "inputs"}, ctx)
	assert.NoError(t, err)
	m, ok := val.(map[string]any)
	assert.True(t, ok)
	assert.Contains(t, m, "userId")
}

// ---------------------------------------------------------------------------
// Components parameters via Components type (general resolver path)
// ---------------------------------------------------------------------------

func TestEvaluate_Components_Parameters(t *testing.T) {
	ctx := fullContext(t)
	val, err := Evaluate(Expression{Type: Components, Name: "parameters", Tail: "myParam"}, ctx)
	assert.NoError(t, err)
	assert.Equal(t, "paramValue", val)
}

func TestEvaluate_Error_ComponentsParametersNilMap(t *testing.T) {
	ctx := &Context{Components: &ComponentsContext{}}
	_, err := Evaluate(Expression{Type: Components, Name: "parameters", Tail: "x"}, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no component parameters")
}

func TestEvaluate_Error_ComponentsParametersMissing(t *testing.T) {
	ctx := &Context{Components: &ComponentsContext{Parameters: map[string]any{"a": 1}}}
	_, err := Evaluate(Expression{Type: Components, Name: "parameters", Tail: "missing"}, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ---------------------------------------------------------------------------
// ResponseQuery nil headers edge case
// ---------------------------------------------------------------------------

func TestEvaluate_ResponseQuery_NotSupported(t *testing.T) {
	_, err := Evaluate(Expression{Type: ResponseQuery, Property: "x"}, &Context{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}
