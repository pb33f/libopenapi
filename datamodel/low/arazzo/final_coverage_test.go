// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

// buildNode is a helper that unmarshals YAML into a yaml.Node and returns the mapping node.
func buildNode(t *testing.T, yml string) *yaml.Node {
	t.Helper()
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yml), &node))
	return node.Content[0]
}

// ---------------------------------------------------------------------------
// Step.Build() exercising all extractArray branches
// ---------------------------------------------------------------------------

func TestFinalCov_Step_Build_RequestBodyEmpty(t *testing.T) {
	// requestBody as a mapping with no fields
	yml := `stepId: s1
operationId: op1
requestBody:
  contentType: text/plain`

	root := buildNode(t, yml)
	var step Step
	require.NoError(t, low.BuildModel(root, &step))
	err := step.Build(context.Background(), nil, root, nil)
	assert.NoError(t, err)
	assert.False(t, step.RequestBody.IsEmpty())
	assert.Equal(t, "text/plain", step.RequestBody.Value.ContentType.Value)
}

func TestFinalCov_Step_Build_AllArrays(t *testing.T) {
	yml := `stepId: s1
operationId: op1
parameters:
  - name: p1
    in: query
    value: v1
  - name: p2
    in: header
    value: v2
requestBody:
  contentType: application/json
  payload:
    key: value
  replacements:
    - target: /key
      value: newval
successCriteria:
  - condition: $statusCode == 200
  - condition: $statusCode == 201
    context: $response.body
onSuccess:
  - name: end-action
    type: end
  - name: goto-action
    type: goto
    stepId: s2
onFailure:
  - name: retry-action
    type: retry
    retryAfter: 1.5
    retryLimit: 3
  - name: end-fail
    type: end
outputs:
  result: $response.body#/id`

	root := buildNode(t, yml)
	var step Step
	require.NoError(t, low.BuildModel(root, &step))
	require.NoError(t, step.Build(context.Background(), nil, root, nil))

	assert.Equal(t, "s1", step.StepId.Value)
	assert.Len(t, step.Parameters.Value, 2)
	assert.NotNil(t, step.RequestBody.Value)
	assert.Len(t, step.SuccessCriteria.Value, 2)
	assert.Len(t, step.OnSuccess.Value, 2)
	assert.Len(t, step.OnFailure.Value, 2)
	assert.NotNil(t, step.Outputs.Value)
}

func TestFinalCov_Step_Build_ParametersNotSeq(t *testing.T) {
	yml := `stepId: s1
operationId: op1
parameters: not-a-sequence`

	root := buildNode(t, yml)
	var step Step
	require.NoError(t, low.BuildModel(root, &step))
	err := step.Build(context.Background(), nil, root, nil)
	assert.NoError(t, err)
	assert.Nil(t, step.Parameters.Value)
}

func TestFinalCov_Step_Build_SuccessCriteriaNotSeq(t *testing.T) {
	yml := `stepId: s1
operationId: op1
successCriteria: not-a-sequence`

	root := buildNode(t, yml)
	var step Step
	require.NoError(t, low.BuildModel(root, &step))
	assert.NoError(t, step.Build(context.Background(), nil, root, nil))
	assert.Nil(t, step.SuccessCriteria.Value)
}

func TestFinalCov_Step_Build_OnSuccessNotSeq(t *testing.T) {
	yml := `stepId: s1
operationId: op1
onSuccess: not-a-sequence`

	root := buildNode(t, yml)
	var step Step
	require.NoError(t, low.BuildModel(root, &step))
	assert.NoError(t, step.Build(context.Background(), nil, root, nil))
	assert.Nil(t, step.OnSuccess.Value)
}

func TestFinalCov_Step_Build_OnFailureNotSeq(t *testing.T) {
	yml := `stepId: s1
operationId: op1
onFailure: not-a-sequence`

	root := buildNode(t, yml)
	var step Step
	require.NoError(t, low.BuildModel(root, &step))
	assert.NoError(t, step.Build(context.Background(), nil, root, nil))
	assert.Nil(t, step.OnFailure.Value)
}

func TestFinalCov_Step_Build_WithWorkflowId(t *testing.T) {
	yml := `stepId: s1
workflowId: other-workflow`

	root := buildNode(t, yml)
	var step Step
	require.NoError(t, low.BuildModel(root, &step))
	require.NoError(t, step.Build(context.Background(), nil, root, nil))
	assert.Equal(t, "other-workflow", step.WorkflowId.Value)
}

func TestFinalCov_Step_Build_WithOperationPath(t *testing.T) {
	yml := `stepId: s1
operationPath: '{$sourceDescriptions.api.url}#/pets/get'`

	root := buildNode(t, yml)
	var step Step
	require.NoError(t, low.BuildModel(root, &step))
	require.NoError(t, step.Build(context.Background(), nil, root, nil))
	assert.False(t, step.OperationPath.IsEmpty())
}

// ---------------------------------------------------------------------------
// Workflow.Build() exercising all extractArray branches
// ---------------------------------------------------------------------------

func TestFinalCov_Workflow_Build_AllArrays(t *testing.T) {
	yml := `workflowId: wf1
summary: Test workflow
description: A test
inputs:
  type: object
dependsOn:
  - wf0
steps:
  - stepId: s1
    operationId: op1
successActions:
  - name: end
    type: end
failureActions:
  - name: retry
    type: retry
    retryAfter: 2.0
    retryLimit: 5
outputs:
  result: $steps.s1.outputs.id
parameters:
  - name: p1
    in: query
    value: v1`

	root := buildNode(t, yml)
	var wf Workflow
	require.NoError(t, low.BuildModel(root, &wf))
	require.NoError(t, wf.Build(context.Background(), nil, root, nil))

	assert.Equal(t, "wf1", wf.WorkflowId.Value)
	assert.Len(t, wf.DependsOn.Value, 1)
	assert.Len(t, wf.Steps.Value, 1)
	assert.Len(t, wf.SuccessActions.Value, 1)
	assert.Len(t, wf.FailureActions.Value, 1)
	assert.NotNil(t, wf.Outputs.Value)
	assert.Len(t, wf.Parameters.Value, 1)
}

func TestFinalCov_Workflow_Build_StepsNotSeq(t *testing.T) {
	yml := `workflowId: wf1
steps: not-a-sequence`

	root := buildNode(t, yml)
	var wf Workflow
	require.NoError(t, low.BuildModel(root, &wf))
	assert.NoError(t, wf.Build(context.Background(), nil, root, nil))
	assert.Nil(t, wf.Steps.Value)
}

func TestFinalCov_Workflow_Build_SuccessActionsNotSeq(t *testing.T) {
	yml := `workflowId: wf1
steps:
  - stepId: s1
    operationId: op1
successActions: not-a-sequence`

	root := buildNode(t, yml)
	var wf Workflow
	require.NoError(t, low.BuildModel(root, &wf))
	assert.NoError(t, wf.Build(context.Background(), nil, root, nil))
	assert.Nil(t, wf.SuccessActions.Value)
}

func TestFinalCov_Workflow_Build_FailureActionsNotSeq(t *testing.T) {
	yml := `workflowId: wf1
steps:
  - stepId: s1
    operationId: op1
failureActions: not-a-sequence`

	root := buildNode(t, yml)
	var wf Workflow
	require.NoError(t, low.BuildModel(root, &wf))
	assert.NoError(t, wf.Build(context.Background(), nil, root, nil))
	assert.Nil(t, wf.FailureActions.Value)
}

func TestFinalCov_Workflow_Build_ParametersNotSeq(t *testing.T) {
	yml := `workflowId: wf1
steps:
  - stepId: s1
    operationId: op1
parameters: not-a-sequence`

	root := buildNode(t, yml)
	var wf Workflow
	require.NoError(t, low.BuildModel(root, &wf))
	assert.NoError(t, wf.Build(context.Background(), nil, root, nil))
	assert.Nil(t, wf.Parameters.Value)
}

// ---------------------------------------------------------------------------
// Arazzo.Build() exercising all branches
// ---------------------------------------------------------------------------

func TestFinalCov_Arazzo_Build_Full(t *testing.T) {
	yml := `arazzo: 1.0.1
info:
  title: Test
  summary: Summary
  description: Description
  version: 0.1.0
sourceDescriptions:
  - name: api
    url: https://example.com
    type: openapi
  - name: other
    url: https://other.com
    type: arazzo
workflows:
  - workflowId: wf1
    steps:
      - stepId: s1
        operationId: op1
  - workflowId: wf2
    steps:
      - stepId: s2
        operationPath: '{$sourceDescriptions.api.url}#/path/op'
components:
  parameters:
    sharedParam:
      name: shared
      in: query
      value: sharedVal
  successActions:
    sharedSuccess:
      name: end
      type: end
  failureActions:
    sharedFailure:
      name: retry
      type: retry
  inputs:
    sharedInput:
      type: string`

	root := buildNode(t, yml)
	var a Arazzo
	require.NoError(t, low.BuildModel(root, &a))
	require.NoError(t, a.Build(context.Background(), nil, root, nil))

	assert.Equal(t, "1.0.1", a.Arazzo.Value)
	assert.False(t, a.Info.IsEmpty())
	assert.Len(t, a.SourceDescriptions.Value, 2)
	assert.Len(t, a.Workflows.Value, 2)
	assert.False(t, a.Components.IsEmpty())
}

// ---------------------------------------------------------------------------
// Components.Build() exercising extractObjectMap branches
// ---------------------------------------------------------------------------

func TestFinalCov_Components_Build_MultipleParams(t *testing.T) {
	yml := `parameters:
  p1:
    name: param1
    in: query
    value: val1
  p2:
    name: param2
    in: header
    value: val2`

	root := buildNode(t, yml)
	var comp Components
	require.NoError(t, low.BuildModel(root, &comp))
	require.NoError(t, comp.Build(context.Background(), nil, root, nil))
	assert.Equal(t, 2, comp.Parameters.Value.Len())
}

func TestFinalCov_Components_Build_MultipleSuccessActions(t *testing.T) {
	yml := `successActions:
  sa1:
    name: end-action
    type: end
  sa2:
    name: goto-action
    type: goto
    stepId: step1`

	root := buildNode(t, yml)
	var comp Components
	require.NoError(t, low.BuildModel(root, &comp))
	require.NoError(t, comp.Build(context.Background(), nil, root, nil))
	assert.Equal(t, 2, comp.SuccessActions.Value.Len())
}

func TestFinalCov_Components_Build_MultipleFailureActions(t *testing.T) {
	yml := `failureActions:
  fa1:
    name: retry-action
    type: retry
    retryAfter: 1.0
    retryLimit: 3
  fa2:
    name: end-action
    type: end`

	root := buildNode(t, yml)
	var comp Components
	require.NoError(t, low.BuildModel(root, &comp))
	require.NoError(t, comp.Build(context.Background(), nil, root, nil))
	assert.Equal(t, 2, comp.FailureActions.Value.Len())
}

// ---------------------------------------------------------------------------
// RequestBody.Build() exercising replacements
// ---------------------------------------------------------------------------

func TestFinalCov_RequestBody_Build_MultipleReplacements(t *testing.T) {
	yml := `contentType: application/json
payload:
  name: test
replacements:
  - target: /name
    value: newName
  - target: /id
    value: 123`

	root := buildNode(t, yml)
	var rb RequestBody
	require.NoError(t, low.BuildModel(root, &rb))
	require.NoError(t, rb.Build(context.Background(), nil, root, nil))
	assert.Len(t, rb.Replacements.Value, 2)
}

func TestFinalCov_RequestBody_Build_ReplacementsNotSeq(t *testing.T) {
	yml := `contentType: application/json
replacements: not-a-sequence`

	root := buildNode(t, yml)
	var rb RequestBody
	require.NoError(t, low.BuildModel(root, &rb))
	assert.NoError(t, rb.Build(context.Background(), nil, root, nil))
	assert.Nil(t, rb.Replacements.Value)
}

// ---------------------------------------------------------------------------
// SuccessAction.Build() exercising criteria and componentRef
// ---------------------------------------------------------------------------

func TestFinalCov_SuccessAction_Build_MultipleCriteria(t *testing.T) {
	yml := `name: goto-action
type: goto
stepId: s2
criteria:
  - condition: $statusCode == 200
  - condition: $response.body#/ok == true
    context: $response.body`

	root := buildNode(t, yml)
	var sa SuccessAction
	require.NoError(t, low.BuildModel(root, &sa))
	require.NoError(t, sa.Build(context.Background(), nil, root, nil))
	assert.Len(t, sa.Criteria.Value, 2)
}

func TestFinalCov_SuccessAction_Build_CriteriaNotSeq(t *testing.T) {
	yml := `name: end
type: end
criteria: not-a-sequence`

	root := buildNode(t, yml)
	var sa SuccessAction
	require.NoError(t, low.BuildModel(root, &sa))
	assert.NoError(t, sa.Build(context.Background(), nil, root, nil))
	assert.Nil(t, sa.Criteria.Value)
}

func TestFinalCov_SuccessAction_Build_ComponentRef(t *testing.T) {
	yml := `reference: $components.successActions.myAction`

	root := buildNode(t, yml)
	var sa SuccessAction
	require.NoError(t, low.BuildModel(root, &sa))
	require.NoError(t, sa.Build(context.Background(), nil, root, nil))
	assert.True(t, sa.IsReusable())
	assert.Equal(t, "$components.successActions.myAction", sa.ComponentRef.Value)
}

func TestFinalCov_SuccessAction_Build_WithWorkflowId(t *testing.T) {
	yml := `name: goto-workflow
type: goto
workflowId: other-workflow`

	root := buildNode(t, yml)
	var sa SuccessAction
	require.NoError(t, low.BuildModel(root, &sa))
	require.NoError(t, sa.Build(context.Background(), nil, root, nil))
	assert.Equal(t, "other-workflow", sa.WorkflowId.Value)
}

// ---------------------------------------------------------------------------
// FailureAction.Build() exercising criteria and componentRef
// ---------------------------------------------------------------------------

func TestFinalCov_FailureAction_Build_MultipleCriteria(t *testing.T) {
	yml := `name: retry-action
type: retry
retryAfter: 2.5
retryLimit: 10
criteria:
  - condition: $statusCode >= 500`

	root := buildNode(t, yml)
	var fa FailureAction
	require.NoError(t, low.BuildModel(root, &fa))
	require.NoError(t, fa.Build(context.Background(), nil, root, nil))
	assert.Equal(t, 2.5, fa.RetryAfter.Value)
	assert.Equal(t, int64(10), fa.RetryLimit.Value)
	assert.Len(t, fa.Criteria.Value, 1)
}

func TestFinalCov_FailureAction_Build_CriteriaNotSeq(t *testing.T) {
	yml := `name: end
type: end
criteria: not-a-sequence`

	root := buildNode(t, yml)
	var fa FailureAction
	require.NoError(t, low.BuildModel(root, &fa))
	assert.NoError(t, fa.Build(context.Background(), nil, root, nil))
	assert.Nil(t, fa.Criteria.Value)
}

func TestFinalCov_FailureAction_Build_ComponentRef(t *testing.T) {
	yml := `reference: $components.failureActions.myAction`

	root := buildNode(t, yml)
	var fa FailureAction
	require.NoError(t, low.BuildModel(root, &fa))
	require.NoError(t, fa.Build(context.Background(), nil, root, nil))
	assert.True(t, fa.IsReusable())
}

func TestFinalCov_FailureAction_Build_WithWorkflowId(t *testing.T) {
	yml := `name: goto-workflow
type: goto
workflowId: other-workflow`

	root := buildNode(t, yml)
	var fa FailureAction
	require.NoError(t, low.BuildModel(root, &fa))
	require.NoError(t, fa.Build(context.Background(), nil, root, nil))
	assert.Equal(t, "other-workflow", fa.WorkflowId.Value)
}

func TestFinalCov_FailureAction_Build_InvalidRetry(t *testing.T) {
	yml := `name: end
type: end
retryAfter: not-a-number
retryLimit: also-not-a-number`

	root := buildNode(t, yml)
	var fa FailureAction
	require.NoError(t, low.BuildModel(root, &fa))
	err := fa.Build(context.Background(), nil, root, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid retryAfter value")
}

// ---------------------------------------------------------------------------
// Hash consistency with all fields populated
// ---------------------------------------------------------------------------

func TestFinalCov_Step_Hash_AllFields(t *testing.T) {
	yml := `stepId: s1
description: A step
operationId: op1
parameters:
  - name: p1
    in: query
    value: v1
requestBody:
  contentType: application/json
  payload: "{}"
  replacements:
    - target: /key
      value: val
successCriteria:
  - condition: $statusCode == 200
onSuccess:
  - name: end
    type: end
onFailure:
  - name: retry
    type: retry
    retryAfter: 1.0
    retryLimit: 3
outputs:
  result: $response.body#/id`

	r1 := buildNode(t, yml)
	r2 := buildNode(t, yml)

	var s1, s2 Step
	_ = low.BuildModel(r1, &s1)
	_ = s1.Build(context.Background(), nil, r1, nil)
	_ = low.BuildModel(r2, &s2)
	_ = s2.Build(context.Background(), nil, r2, nil)

	assert.Equal(t, s1.Hash(), s2.Hash())
}

func TestFinalCov_Workflow_Hash_AllFields(t *testing.T) {
	yml := `workflowId: wf1
summary: My Workflow
description: A workflow
inputs:
  type: object
dependsOn:
  - wf0
steps:
  - stepId: s1
    operationId: op1
successActions:
  - name: end
    type: end
failureActions:
  - name: retry
    type: retry
outputs:
  result: $steps.s1.outputs.id
parameters:
  - name: p1
    in: query
    value: v1`

	r1 := buildNode(t, yml)
	r2 := buildNode(t, yml)

	var w1, w2 Workflow
	_ = low.BuildModel(r1, &w1)
	_ = w1.Build(context.Background(), nil, r1, nil)
	_ = low.BuildModel(r2, &w2)
	_ = w2.Build(context.Background(), nil, r2, nil)

	assert.Equal(t, w1.Hash(), w2.Hash())
}

func TestFinalCov_SuccessAction_Hash_AllFields(t *testing.T) {
	yml := `name: goto-action
type: goto
workflowId: wf2
stepId: s3
criteria:
  - condition: $statusCode == 200`

	r1 := buildNode(t, yml)
	r2 := buildNode(t, yml)

	var s1, s2 SuccessAction
	_ = low.BuildModel(r1, &s1)
	_ = s1.Build(context.Background(), nil, r1, nil)
	_ = low.BuildModel(r2, &s2)
	_ = s2.Build(context.Background(), nil, r2, nil)

	assert.Equal(t, s1.Hash(), s2.Hash())
}

func TestFinalCov_SuccessAction_Hash_ComponentRef(t *testing.T) {
	yml := `reference: $components.successActions.myAction`

	r1 := buildNode(t, yml)
	r2 := buildNode(t, yml)

	var s1, s2 SuccessAction
	_ = low.BuildModel(r1, &s1)
	_ = s1.Build(context.Background(), nil, r1, nil)
	_ = low.BuildModel(r2, &s2)
	_ = s2.Build(context.Background(), nil, r2, nil)

	assert.Equal(t, s1.Hash(), s2.Hash())
}

func TestFinalCov_FailureAction_Hash_AllFields(t *testing.T) {
	yml := `name: retry-action
type: retry
workflowId: wf2
stepId: s3
retryAfter: 1.5
retryLimit: 5
criteria:
  - condition: $statusCode >= 500`

	r1 := buildNode(t, yml)
	r2 := buildNode(t, yml)

	var f1, f2 FailureAction
	_ = low.BuildModel(r1, &f1)
	_ = f1.Build(context.Background(), nil, r1, nil)
	_ = low.BuildModel(r2, &f2)
	_ = f2.Build(context.Background(), nil, r2, nil)

	assert.Equal(t, f1.Hash(), f2.Hash())
}

func TestFinalCov_FailureAction_Hash_ComponentRef(t *testing.T) {
	yml := `reference: $components.failureActions.myAction`

	r1 := buildNode(t, yml)
	r2 := buildNode(t, yml)

	var f1, f2 FailureAction
	_ = low.BuildModel(r1, &f1)
	_ = f1.Build(context.Background(), nil, r1, nil)
	_ = low.BuildModel(r2, &f2)
	_ = f2.Build(context.Background(), nil, r2, nil)

	assert.Equal(t, f1.Hash(), f2.Hash())
}

// ---------------------------------------------------------------------------
// Getters coverage
// ---------------------------------------------------------------------------

func TestFinalCov_Step_Getters(t *testing.T) {
	yml := `stepId: s1
operationId: op1
x-step-ext: val`

	root := buildNode(t, yml)
	keyNode := &yaml.Node{Value: "step"}
	var step Step
	_ = low.BuildModel(root, &step)
	_ = step.Build(context.Background(), keyNode, root, nil)

	assert.Equal(t, keyNode, step.GetKeyNode())
	assert.Equal(t, root, step.GetRootNode())
	assert.Nil(t, step.GetIndex())
	assert.NotNil(t, step.GetContext())
	assert.NotNil(t, step.GetExtensions())
	ext := step.FindExtension("x-step-ext")
	require.NotNil(t, ext)
}

func TestFinalCov_Workflow_Getters(t *testing.T) {
	yml := `workflowId: wf1
steps:
  - stepId: s1
    operationId: op1
x-wf-ext: val`

	root := buildNode(t, yml)
	keyNode := &yaml.Node{Value: "workflow"}
	var wf Workflow
	_ = low.BuildModel(root, &wf)
	_ = wf.Build(context.Background(), keyNode, root, nil)

	assert.Equal(t, keyNode, wf.GetKeyNode())
	assert.Equal(t, root, wf.GetRootNode())
	assert.Nil(t, wf.GetIndex())
	assert.NotNil(t, wf.GetContext())
	assert.NotNil(t, wf.GetExtensions())
	ext := wf.FindExtension("x-wf-ext")
	require.NotNil(t, ext)
}

func TestFinalCov_FailureAction_Getters(t *testing.T) {
	yml := `name: end
type: end
x-fa-ext: val`

	root := buildNode(t, yml)
	keyNode := &yaml.Node{Value: "fa"}
	var fa FailureAction
	_ = low.BuildModel(root, &fa)
	_ = fa.Build(context.Background(), keyNode, root, nil)

	assert.Equal(t, keyNode, fa.GetKeyNode())
	assert.Equal(t, root, fa.GetRootNode())
	assert.Nil(t, fa.GetIndex())
	assert.NotNil(t, fa.GetContext())
	ext := fa.FindExtension("x-fa-ext")
	require.NotNil(t, ext)
}

func TestFinalCov_SuccessAction_Getters(t *testing.T) {
	yml := `name: end
type: end
x-sa-ext: val`

	root := buildNode(t, yml)
	keyNode := &yaml.Node{Value: "sa"}
	var sa SuccessAction
	_ = low.BuildModel(root, &sa)
	_ = sa.Build(context.Background(), keyNode, root, nil)

	assert.Equal(t, keyNode, sa.GetKeyNode())
	assert.Equal(t, root, sa.GetRootNode())
	assert.Nil(t, sa.GetIndex())
	assert.NotNil(t, sa.GetContext())
	ext := sa.FindExtension("x-sa-ext")
	require.NotNil(t, ext)
}

func TestFinalCov_Criterion_Getters(t *testing.T) {
	yml := `condition: $statusCode == 200`

	root := buildNode(t, yml)
	keyNode := &yaml.Node{Value: "crit"}
	var crit Criterion
	_ = low.BuildModel(root, &crit)
	_ = crit.Build(context.Background(), keyNode, root, nil)

	assert.Equal(t, keyNode, crit.GetKeyNode())
	assert.Equal(t, root, crit.GetRootNode())
	assert.Nil(t, crit.GetIndex())
	assert.NotNil(t, crit.GetContext())
	assert.Nil(t, crit.FindExtension("x-nope"))
}

func TestFinalCov_Parameter_Getters(t *testing.T) {
	yml := `name: p1
in: query
value: v1
x-param-ext: val`

	root := buildNode(t, yml)
	keyNode := &yaml.Node{Value: "param"}
	var p Parameter
	_ = low.BuildModel(root, &p)
	_ = p.Build(context.Background(), keyNode, root, nil)

	assert.Equal(t, keyNode, p.GetKeyNode())
	assert.Equal(t, root, p.GetRootNode())
	assert.Nil(t, p.GetIndex())
	assert.NotNil(t, p.GetContext())
	ext := p.FindExtension("x-param-ext")
	require.NotNil(t, ext)
}

func TestFinalCov_RequestBody_Getters(t *testing.T) {
	yml := `contentType: application/json
x-rb-ext: val`

	root := buildNode(t, yml)
	keyNode := &yaml.Node{Value: "rb"}
	var rb RequestBody
	_ = low.BuildModel(root, &rb)
	_ = rb.Build(context.Background(), keyNode, root, nil)

	assert.Equal(t, keyNode, rb.GetKeyNode())
	assert.Equal(t, root, rb.GetRootNode())
	assert.Nil(t, rb.GetIndex())
	assert.NotNil(t, rb.GetContext())
	ext := rb.FindExtension("x-rb-ext")
	require.NotNil(t, ext)
}

func TestFinalCov_Parameter_ComponentRef(t *testing.T) {
	yml := `reference: $components.parameters.sharedParam`

	root := buildNode(t, yml)
	var p Parameter
	require.NoError(t, low.BuildModel(root, &p))
	require.NoError(t, p.Build(context.Background(), nil, root, nil))
	assert.True(t, p.IsReusable())
}

func TestFinalCov_SourceDescription_Build(t *testing.T) {
	yml := `name: api
url: https://example.com/api.yaml
type: openapi
x-custom: myval`

	root := buildNode(t, yml)
	keyNode := &yaml.Node{Value: "sd"}
	var sd SourceDescription
	require.NoError(t, low.BuildModel(root, &sd))
	require.NoError(t, sd.Build(context.Background(), keyNode, root, nil))

	assert.Equal(t, "api", sd.Name.Value)
	assert.Equal(t, keyNode, sd.GetKeyNode())
	assert.Equal(t, root, sd.GetRootNode())
	assert.Nil(t, sd.GetIndex())
	assert.NotNil(t, sd.GetContext())
	ext := sd.FindExtension("x-custom")
	require.NotNil(t, ext)
}

func TestFinalCov_Info_Build_AllFields(t *testing.T) {
	yml := `title: Test API
summary: A test
description: Detailed description
version: 1.0.0
x-info-ext: val`

	root := buildNode(t, yml)
	keyNode := &yaml.Node{Value: "info"}
	var info Info
	require.NoError(t, low.BuildModel(root, &info))
	require.NoError(t, info.Build(context.Background(), keyNode, root, nil))

	assert.Equal(t, "Test API", info.Title.Value)
	assert.Equal(t, "A test", info.Summary.Value)
	assert.Equal(t, keyNode, info.GetKeyNode())
	assert.Equal(t, root, info.GetRootNode())
	assert.Nil(t, info.GetIndex())
	assert.NotNil(t, info.GetContext())
	ext := info.FindExtension("x-info-ext")
	require.NotNil(t, ext)
}

func TestFinalCov_Info_Hash_Consistency(t *testing.T) {
	yml := `title: Test
summary: S
description: D
version: 1.0.0`

	r1 := buildNode(t, yml)
	r2 := buildNode(t, yml)

	var i1, i2 Info
	_ = low.BuildModel(r1, &i1)
	_ = i1.Build(context.Background(), nil, r1, nil)
	_ = low.BuildModel(r2, &i2)
	_ = i2.Build(context.Background(), nil, r2, nil)

	assert.Equal(t, i1.Hash(), i2.Hash())
}

func TestFinalCov_PayloadReplacement_Build(t *testing.T) {
	yml := `target: /name
value: newName`

	root := buildNode(t, yml)
	keyNode := &yaml.Node{Value: "rep"}
	var pr PayloadReplacement
	require.NoError(t, low.BuildModel(root, &pr))
	require.NoError(t, pr.Build(context.Background(), keyNode, root, nil))

	assert.Equal(t, "/name", pr.Target.Value)
	assert.Equal(t, keyNode, pr.GetKeyNode())
	assert.Equal(t, root, pr.GetRootNode())
	assert.Nil(t, pr.GetIndex())
	assert.NotNil(t, pr.GetContext())
	assert.Nil(t, pr.FindExtension("x-nope"))
}

func TestFinalCov_PayloadReplacement_Hash(t *testing.T) {
	yml := `target: /name
value: newName`

	r1 := buildNode(t, yml)
	r2 := buildNode(t, yml)

	var p1, p2 PayloadReplacement
	_ = low.BuildModel(r1, &p1)
	_ = p1.Build(context.Background(), nil, r1, nil)
	_ = low.BuildModel(r2, &p2)
	_ = p2.Build(context.Background(), nil, r2, nil)

	assert.Equal(t, p1.Hash(), p2.Hash())
}

func TestFinalCov_Criterion_TypeAsMapping(t *testing.T) {
	yml := `condition: $response.body#/ok == true
context: $response.body
type:
  type: jsonpath
  version: draft-goessner-dispatch-jsonpath-00`

	root := buildNode(t, yml)
	ymlSafe := `condition: $response.body#/ok == true`
	safeRoot := buildNode(t, ymlSafe)

	var crit Criterion
	require.NoError(t, low.BuildModel(safeRoot, &crit))
	require.NoError(t, crit.Build(context.Background(), nil, root, nil))

	assert.False(t, crit.Type.IsEmpty())
	assert.Equal(t, yaml.MappingNode, crit.Type.Value.Kind)
}

// ---------------------------------------------------------------------------
// helpers.go: edge cases for extract functions
// ---------------------------------------------------------------------------

func TestFinalCov_ExtractStringArray_NotSeq(t *testing.T) {
	yml := `dependsOn: not-a-sequence`
	root := buildNode(t, yml)
	result := extractStringArray(DependsOnLabel, root)
	assert.Nil(t, result.Value)
}

func TestFinalCov_ExtractStringArray_Empty(t *testing.T) {
	yml := `dependsOn: []`
	root := buildNode(t, yml)
	result := extractStringArray(DependsOnLabel, root)
	assert.NotNil(t, result.Value)
	assert.Len(t, result.Value, 0)
}

func TestFinalCov_ExtractRawNode_NotFound(t *testing.T) {
	yml := `someKey: value`
	root := buildNode(t, yml)
	result := extractRawNode("missingKey", root)
	assert.Nil(t, result.Value)
}

func TestFinalCov_ExtractExpressionsMap_NotMapping(t *testing.T) {
	yml := `outputs: not-a-mapping`
	root := buildNode(t, yml)
	result := extractExpressionsMap(OutputsLabel, root)
	assert.Nil(t, result.Value)
}

func TestFinalCov_ExtractExpressionsMap_Empty(t *testing.T) {
	yml := `outputs: {}`
	root := buildNode(t, yml)
	result := extractExpressionsMap(OutputsLabel, root)
	assert.NotNil(t, result.Value)
	assert.Equal(t, 0, result.Value.Len())
}

func TestFinalCov_ExtractRawNodeMap_NotMapping(t *testing.T) {
	yml := `inputs: not-a-mapping`
	root := buildNode(t, yml)
	result := extractRawNodeMap(InputsLabel, root)
	assert.Nil(t, result.Value)
}

func TestFinalCov_ExtractRawNodeMap_Empty(t *testing.T) {
	yml := `inputs: {}`
	root := buildNode(t, yml)
	result := extractRawNodeMap(InputsLabel, root)
	assert.NotNil(t, result.Value)
	assert.Equal(t, 0, result.Value.Len())
}

func TestFinalCov_ExtractArray_OddContentRoot(t *testing.T) {
	root := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "parameters"},
			{Kind: yaml.SequenceNode, Content: []*yaml.Node{
				{Kind: yaml.MappingNode, Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "name"},
					{Kind: yaml.ScalarNode, Value: "p1"},
				}},
			}},
			{Kind: yaml.ScalarNode, Value: "orphan"},
		},
	}

	result, err := extractArray[Parameter](context.Background(), "parameters", root, nil)
	assert.NoError(t, err)
	assert.Len(t, result.Value, 1)
}

func TestFinalCov_ExtractObjectMap_OddValueContent(t *testing.T) {
	root := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "parameters"},
			{Kind: yaml.MappingNode, Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "p1"},
				{Kind: yaml.MappingNode, Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "name"},
					{Kind: yaml.ScalarNode, Value: "param1"},
				}},
				{Kind: yaml.ScalarNode, Value: "orphan"},
			}},
		},
	}

	result, err := extractObjectMap[Parameter](context.Background(), "parameters", root, nil)
	assert.NoError(t, err)
	assert.NotNil(t, result.Value)
	assert.Equal(t, 1, result.Value.Len())
}

// ---------------------------------------------------------------------------
// Hash: nil extensions
// ---------------------------------------------------------------------------

func TestFinalCov_HashExtensions_NilMap(t *testing.T) {
	var step Step
	step.StepId = low.NodeReference[string]{Value: "s1", ValueNode: &yaml.Node{Kind: yaml.ScalarNode, Value: "s1"}}
	step.Extensions = nil
	h := step.Hash()
	assert.NotZero(t, h)
}
