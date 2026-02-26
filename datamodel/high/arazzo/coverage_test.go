// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"context"
	"strings"
	"testing"

	lowmodel "github.com/pb33f/libopenapi/datamodel/low"
	low "github.com/pb33f/libopenapi/datamodel/low/arazzo"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

// buildHighFromYAML is a test helper that builds a full high-level Arazzo model from YAML.
func buildHighFromYAML(t *testing.T, yml string) *Arazzo {
	t.Helper()
	var rootNode yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yml), &rootNode))
	require.NotEmpty(t, rootNode.Content)

	mappingNode := rootNode.Content[0]

	lowDoc := &low.Arazzo{}
	require.NoError(t, lowmodel.BuildModel(mappingNode, lowDoc))
	require.NoError(t, lowDoc.Build(context.Background(), nil, mappingNode, nil))

	return NewArazzo(lowDoc)
}

// ---------------------------------------------------------------------------
// MarshalYAML extension loop coverage for each model
// ---------------------------------------------------------------------------

func TestInfo_MarshalYAML_WithExtensions(t *testing.T) {
	yml := `arazzo: 1.0.1
info:
  title: Test
  summary: Sum
  description: Desc
  version: 0.1.0
  x-info-ext: hello
sourceDescriptions:
  - name: api
    url: https://example.com
workflows:
  - workflowId: wf1
    steps:
      - stepId: s1
        operationId: op1`

	h := buildHighFromYAML(t, yml)
	require.NotNil(t, h.Info.Extensions)

	rendered, err := h.Info.Render()
	require.NoError(t, err)
	s := string(rendered)
	assert.Contains(t, s, "x-info-ext")
}

func TestSourceDescription_MarshalYAML_WithExtensions(t *testing.T) {
	yml := `arazzo: 1.0.1
info:
  title: Test
  version: 0.1.0
sourceDescriptions:
  - name: api
    url: https://example.com
    type: openapi
    x-sd-ext: vendor
workflows:
  - workflowId: wf1
    steps:
      - stepId: s1
        operationId: op1`

	h := buildHighFromYAML(t, yml)
	require.NotNil(t, h.SourceDescriptions[0].Extensions)

	rendered, err := h.SourceDescriptions[0].Render()
	require.NoError(t, err)
	s := string(rendered)
	assert.Contains(t, s, "x-sd-ext")
}

func TestCriterionExpressionType_MarshalYAML_WithExtensions(t *testing.T) {
	yml := `arazzo: 1.0.1
info:
  title: Test
  version: 0.1.0
sourceDescriptions:
  - name: api
    url: https://example.com
workflows:
  - workflowId: wf1
    steps:
      - stepId: s1
        operationId: op1
        successCriteria:
          - condition: $.data != null
            context: $response.body
            type:
              type: jsonpath
              version: draft-01
              x-cet-ext: custom`

	h := buildHighFromYAML(t, yml)
	cet := h.Workflows[0].Steps[0].SuccessCriteria[0].ExpressionType
	require.NotNil(t, cet)
	require.NotNil(t, cet.Extensions)

	rendered, err := cet.Render()
	require.NoError(t, err)
	s := string(rendered)
	assert.Contains(t, s, "x-cet-ext")
}

func TestPayloadReplacement_MarshalYAML_WithExtensions(t *testing.T) {
	yml := `arazzo: 1.0.1
info:
  title: Test
  version: 0.1.0
sourceDescriptions:
  - name: api
    url: https://example.com
workflows:
  - workflowId: wf1
    steps:
      - stepId: s1
        operationId: op1
        requestBody:
          contentType: application/json
          payload:
            name: test
          replacements:
            - target: /name
              value: replaced
              x-pr-ext: meta`

	h := buildHighFromYAML(t, yml)
	rep := h.Workflows[0].Steps[0].RequestBody.Replacements[0]
	require.NotNil(t, rep.Extensions)

	rendered, err := rep.Render()
	require.NoError(t, err)
	s := string(rendered)
	assert.Contains(t, s, "x-pr-ext")
}

func TestCriterion_MarshalYAML_WithExtensions(t *testing.T) {
	yml := `arazzo: 1.0.1
info:
  title: Test
  version: 0.1.0
sourceDescriptions:
  - name: api
    url: https://example.com
workflows:
  - workflowId: wf1
    steps:
      - stepId: s1
        operationId: op1
        successCriteria:
          - condition: $statusCode == 200
            x-crit-ext: info`

	h := buildHighFromYAML(t, yml)
	crit := h.Workflows[0].Steps[0].SuccessCriteria[0]
	require.NotNil(t, crit.Extensions)

	rendered, err := crit.Render()
	require.NoError(t, err)
	s := string(rendered)
	assert.Contains(t, s, "x-crit-ext")
}

func TestRequestBody_MarshalYAML_WithExtensions(t *testing.T) {
	yml := `arazzo: 1.0.1
info:
  title: Test
  version: 0.1.0
sourceDescriptions:
  - name: api
    url: https://example.com
workflows:
  - workflowId: wf1
    steps:
      - stepId: s1
        operationId: op1
        requestBody:
          contentType: application/json
          payload:
            name: test
          x-rb-ext: data`

	h := buildHighFromYAML(t, yml)
	rb := h.Workflows[0].Steps[0].RequestBody
	require.NotNil(t, rb.Extensions)

	rendered, err := rb.Render()
	require.NoError(t, err)
	s := string(rendered)
	assert.Contains(t, s, "x-rb-ext")
}

func TestStep_MarshalYAML_WithExtensions(t *testing.T) {
	yml := `arazzo: 1.0.1
info:
  title: Test
  version: 0.1.0
sourceDescriptions:
  - name: api
    url: https://example.com
workflows:
  - workflowId: wf1
    steps:
      - stepId: s1
        operationId: op1
        x-step-ext: val`

	h := buildHighFromYAML(t, yml)
	step := h.Workflows[0].Steps[0]
	require.NotNil(t, step.Extensions)

	rendered, err := step.Render()
	require.NoError(t, err)
	s := string(rendered)
	assert.Contains(t, s, "x-step-ext")
}

func TestWorkflow_MarshalYAML_WithExtensions(t *testing.T) {
	yml := `arazzo: 1.0.1
info:
  title: Test
  version: 0.1.0
sourceDescriptions:
  - name: api
    url: https://example.com
workflows:
  - workflowId: wf1
    steps:
      - stepId: s1
        operationId: op1
    x-wf-ext: meta`

	h := buildHighFromYAML(t, yml)
	wf := h.Workflows[0]
	require.NotNil(t, wf.Extensions)

	rendered, err := wf.Render()
	require.NoError(t, err)
	s := string(rendered)
	assert.Contains(t, s, "x-wf-ext")
}

func TestComponents_MarshalYAML_WithExtensions(t *testing.T) {
	yml := `arazzo: 1.0.1
info:
  title: Test
  version: 0.1.0
sourceDescriptions:
  - name: api
    url: https://example.com
workflows:
  - workflowId: wf1
    steps:
      - stepId: s1
        operationId: op1
components:
  parameters:
    p1:
      name: key
      in: header
      value: val
  x-comp-ext: data`

	h := buildHighFromYAML(t, yml)
	comp := h.Components
	require.NotNil(t, comp)
	require.NotNil(t, comp.Extensions)

	rendered, err := comp.Render()
	require.NoError(t, err)
	s := string(rendered)
	assert.Contains(t, s, "x-comp-ext")
}

func TestArazzo_MarshalYAML_WithExtensions(t *testing.T) {
	yml := `arazzo: 1.0.1
info:
  title: Test
  version: 0.1.0
sourceDescriptions:
  - name: api
    url: https://example.com
workflows:
  - workflowId: wf1
    steps:
      - stepId: s1
        operationId: op1
x-arazzo-ext: top`

	h := buildHighFromYAML(t, yml)
	require.NotNil(t, h.Extensions)

	rendered, err := h.Render()
	require.NoError(t, err)
	s := string(rendered)
	assert.Contains(t, s, "x-arazzo-ext")
}

// ---------------------------------------------------------------------------
// FailureAction MarshalYAML: non-reusable with retryAfter=0 and retryLimit=0
// (uses != 0 check, so zero values should NOT be included in output)
// ---------------------------------------------------------------------------

func ptrFloat64(v float64) *float64 { return &v }
func ptrInt64(v int64) *int64       { return &v }

func TestFailureAction_MarshalYAML_NilRetryFields(t *testing.T) {
	// Create a FailureAction with nil retryAfter and retryLimit (not set)
	fa := &FailureAction{
		Name: "testAction",
		Type: "retry",
	}

	rendered, err := fa.Render()
	require.NoError(t, err)
	s := string(rendered)

	assert.Contains(t, s, "name: testAction")
	assert.Contains(t, s, "type: retry")
	// retryAfter and retryLimit with nil values should NOT appear in output
	assert.NotContains(t, s, "retryAfter")
	assert.NotContains(t, s, "retryLimit")
}

func TestFailureAction_MarshalYAML_ZeroRetryFields(t *testing.T) {
	// Explicitly set to zero - should appear in output (distinguishable from nil)
	fa := &FailureAction{
		Name:       "testAction",
		Type:       "retry",
		RetryAfter: ptrFloat64(0),
		RetryLimit: ptrInt64(0),
	}

	rendered, err := fa.Render()
	require.NoError(t, err)
	s := string(rendered)

	assert.Contains(t, s, "name: testAction")
	assert.Contains(t, s, "type: retry")
	assert.Contains(t, s, "retryAfter")
	assert.Contains(t, s, "retryLimit")
}

func TestFailureAction_MarshalYAML_NonZeroRetryFields(t *testing.T) {
	fa := &FailureAction{
		Name:       "retryAction",
		Type:       "retry",
		RetryAfter: ptrFloat64(3.5),
		RetryLimit: ptrInt64(10),
	}

	rendered, err := fa.Render()
	require.NoError(t, err)
	s := string(rendered)

	assert.Contains(t, s, "name: retryAction")
	assert.Contains(t, s, "type: retry")
	assert.Contains(t, s, "retryAfter")
	assert.Contains(t, s, "retryLimit")
}

func TestFailureAction_MarshalYAML_WithExtensions(t *testing.T) {
	yml := `arazzo: 1.0.1
info:
  title: Test
  version: 0.1.0
sourceDescriptions:
  - name: api
    url: https://example.com
workflows:
  - workflowId: wf1
    steps:
      - stepId: s1
        operationId: op1
        onFailure:
          - name: retryAction
            type: retry
            retryAfter: 1.0
            retryLimit: 3
            x-fa-ext: info`

	h := buildHighFromYAML(t, yml)
	fa := h.Workflows[0].Steps[0].OnFailure[0]
	require.NotNil(t, fa.Extensions)

	rendered, err := fa.Render()
	require.NoError(t, err)
	s := string(rendered)
	assert.Contains(t, s, "x-fa-ext")
}

func TestFailureAction_MarshalYAML_Reusable(t *testing.T) {
	// Reusable failure action should render only the reference
	fa := &FailureAction{
		Reference: "$components.failureActions.myAction",
	}

	rendered, err := fa.Render()
	require.NoError(t, err)
	s := string(rendered)

	assert.Contains(t, s, "reference:")
	assert.NotContains(t, s, "name:")
	assert.NotContains(t, s, "type:")
}

// ---------------------------------------------------------------------------
// SuccessAction MarshalYAML: non-reusable with extensions
// ---------------------------------------------------------------------------

func TestSuccessAction_MarshalYAML_NonReusableWithExtensions(t *testing.T) {
	yml := `arazzo: 1.0.1
info:
  title: Test
  version: 0.1.0
sourceDescriptions:
  - name: api
    url: https://example.com
workflows:
  - workflowId: wf1
    steps:
      - stepId: s1
        operationId: op1
        onSuccess:
          - name: endAction
            type: end
            x-sa-ext: info`

	h := buildHighFromYAML(t, yml)
	sa := h.Workflows[0].Steps[0].OnSuccess[0]
	require.NotNil(t, sa.Extensions)

	rendered, err := sa.Render()
	require.NoError(t, err)
	s := string(rendered)
	assert.Contains(t, s, "x-sa-ext")
	assert.Contains(t, s, "name: endAction")
	assert.Contains(t, s, "type: end")
}

// ---------------------------------------------------------------------------
// NewFailureAction with workflowId set (covers the !fa.WorkflowId.IsEmpty() branch)
// ---------------------------------------------------------------------------

func TestNewFailureAction_WithWorkflowId(t *testing.T) {
	yml := `arazzo: 1.0.1
info:
  title: Test
  version: 0.1.0
sourceDescriptions:
  - name: api
    url: https://example.com
workflows:
  - workflowId: wf1
    steps:
      - stepId: s1
        operationId: op1
        onFailure:
          - name: goToOther
            type: goto
            workflowId: otherWorkflow
            stepId: otherStep`

	h := buildHighFromYAML(t, yml)
	fa := h.Workflows[0].Steps[0].OnFailure[0]

	assert.Equal(t, "goToOther", fa.Name)
	assert.Equal(t, "goto", fa.Type)
	assert.Equal(t, "otherWorkflow", fa.WorkflowId)
	assert.Equal(t, "otherStep", fa.StepId)
	assert.Nil(t, fa.RetryAfter)
	assert.Nil(t, fa.RetryLimit)
	assert.False(t, fa.IsReusable())
}

// ---------------------------------------------------------------------------
// NewSuccessAction with workflowId set (covers the !sa.WorkflowId.IsEmpty() branch)
// ---------------------------------------------------------------------------

func TestNewSuccessAction_WithWorkflowId(t *testing.T) {
	yml := `arazzo: 1.0.1
info:
  title: Test
  version: 0.1.0
sourceDescriptions:
  - name: api
    url: https://example.com
workflows:
  - workflowId: wf1
    steps:
      - stepId: s1
        operationId: op1
        onSuccess:
          - name: goToOther
            type: goto
            workflowId: otherWorkflow`

	h := buildHighFromYAML(t, yml)
	sa := h.Workflows[0].Steps[0].OnSuccess[0]

	assert.Equal(t, "goToOther", sa.Name)
	assert.Equal(t, "goto", sa.Type)
	assert.Equal(t, "otherWorkflow", sa.WorkflowId)
	assert.Empty(t, sa.StepId)
	assert.False(t, sa.IsReusable())
}

// ---------------------------------------------------------------------------
// NewFailureAction with RetryAfter/RetryLimit empty (not set in low model)
// ---------------------------------------------------------------------------

func TestNewFailureAction_EmptyRetryFields(t *testing.T) {
	yml := `arazzo: 1.0.1
info:
  title: Test
  version: 0.1.0
sourceDescriptions:
  - name: api
    url: https://example.com
workflows:
  - workflowId: wf1
    steps:
      - stepId: s1
        operationId: op1
        onFailure:
          - name: endAction
            type: end`

	h := buildHighFromYAML(t, yml)
	fa := h.Workflows[0].Steps[0].OnFailure[0]

	assert.Equal(t, "endAction", fa.Name)
	assert.Equal(t, "end", fa.Type)
	assert.Nil(t, fa.RetryAfter)
	assert.Nil(t, fa.RetryLimit)
	assert.False(t, fa.IsReusable())
}

// ---------------------------------------------------------------------------
// Round-trip MarshalYAML with extension entries for all models
// ---------------------------------------------------------------------------

func TestRoundTrip_AllModelsWithExtensions(t *testing.T) {
	yml := `arazzo: 1.0.1
info:
  title: Full Test
  summary: Summary text
  description: Description text
  version: 1.0.0
  x-info-extra: infoVal
sourceDescriptions:
  - name: petStoreApi
    url: https://petstore.example.com/openapi.json
    type: openapi
    x-sd-vendor: acme
workflows:
  - workflowId: createPet
    summary: Create a pet
    description: Create a pet workflow
    dependsOn:
      - verifyPet
    inputs:
      type: object
    steps:
      - stepId: addPet
        operationId: addPet
        description: Add a pet
        parameters:
          - name: api_key
            in: header
            value: abc123
        requestBody:
          contentType: application/json
          payload:
            name: fluffy
          replacements:
            - target: /name
              value: replaced
        successCriteria:
          - condition: $statusCode == 200
            type: simple
          - condition: $response.body#/id != null
            context: $response.body
            type:
              type: jsonpath
              version: draft-01
        onSuccess:
          - name: logSuccess
            type: end
        onFailure:
          - name: retryAdd
            type: retry
            retryAfter: 1.5
            retryLimit: 3
        outputs:
          petId: $response.body#/id
        x-step-custom: stepVal
    successActions:
      - name: notify
        type: goto
        stepId: addPet
    failureActions:
      - name: abort
        type: end
    outputs:
      result: $steps.addPet.outputs.petId
    parameters:
      - name: storeId
        in: query
        value: store-1
    x-wf-custom: wfVal
  - workflowId: verifyPet
    steps:
      - stepId: check
        operationId: getPetById
components:
  inputs:
    petInput:
      type: object
  parameters:
    apiKey:
      name: api_key
      in: header
      value: default
  successActions:
    logEnd:
      name: logEnd
      type: end
  failureActions:
    retryDefault:
      name: retryDefault
      type: retry
      retryAfter: 2.0
      retryLimit: 5
x-top-level: topVal`

	h1 := buildHighFromYAML(t, yml)

	// Render to YAML
	rendered1, err := h1.Render()
	require.NoError(t, err)
	s := string(rendered1)

	// Verify extensions are in the rendered output
	assert.Contains(t, s, "x-info-extra")
	assert.Contains(t, s, "x-sd-vendor")
	assert.Contains(t, s, "x-step-custom")
	assert.Contains(t, s, "x-wf-custom")
	assert.Contains(t, s, "x-top-level")

	// Re-parse and verify round-trip
	var rootNode yaml.Node
	require.NoError(t, yaml.Unmarshal(rendered1, &rootNode))
	lowDoc := &low.Arazzo{}
	require.NoError(t, lowmodel.BuildModel(rootNode.Content[0], lowDoc))
	require.NoError(t, lowDoc.Build(context.Background(), nil, rootNode.Content[0], nil))
	h2 := NewArazzo(lowDoc)

	assert.Equal(t, h1.Arazzo, h2.Arazzo)
	assert.Equal(t, h1.Info.Title, h2.Info.Title)
	assert.Equal(t, h1.Info.Summary, h2.Info.Summary)
	assert.Equal(t, h1.Info.Version, h2.Info.Version)
	assert.Len(t, h2.SourceDescriptions, len(h1.SourceDescriptions))
	assert.Len(t, h2.Workflows, len(h1.Workflows))
}

// ---------------------------------------------------------------------------
// Criterion MarshalYAML: no type set (default simple)
// ---------------------------------------------------------------------------

func TestCriterion_MarshalYAML_NoType(t *testing.T) {
	c := &Criterion{
		Condition: "$statusCode == 200",
	}

	rendered, err := c.Render()
	require.NoError(t, err)
	s := string(rendered)
	assert.Contains(t, s, "condition:")
	// No type set, so type should not appear
	assert.NotContains(t, s, "type:")
}

func TestCriterion_MarshalYAML_WithContext(t *testing.T) {
	c := &Criterion{
		Context:   "$response.body",
		Condition: "$statusCode == 200",
		Type:      "regex",
	}

	rendered, err := c.Render()
	require.NoError(t, err)
	s := string(rendered)
	assert.Contains(t, s, "context:")
	assert.Contains(t, s, "condition:")
	assert.Contains(t, s, "type: regex")
}

func TestCriterion_MarshalYAML_WithExpressionType(t *testing.T) {
	c := &Criterion{
		Condition: "$.data != null",
		ExpressionType: &CriterionExpressionType{
			Type:    "jsonpath",
			Version: "draft-01",
		},
	}

	rendered, err := c.Render()
	require.NoError(t, err)
	s := string(rendered)
	assert.Contains(t, s, "condition:")
	assert.Contains(t, s, "type:")
	assert.Contains(t, s, "jsonpath")
}

// ---------------------------------------------------------------------------
// Parameter MarshalYAML: reusable with extensions
// ---------------------------------------------------------------------------

func TestParameter_MarshalYAML_WithExtensions(t *testing.T) {
	yml := `arazzo: 1.0.1
info:
  title: Test
  version: 0.1.0
sourceDescriptions:
  - name: api
    url: https://example.com
workflows:
  - workflowId: wf1
    steps:
      - stepId: s1
        operationId: op1
        parameters:
          - name: key
            in: header
            value: val
            x-param-ext: pval`

	h := buildHighFromYAML(t, yml)
	param := h.Workflows[0].Steps[0].Parameters[0]
	require.NotNil(t, param.Extensions)

	rendered, err := param.Render()
	require.NoError(t, err)
	s := string(rendered)
	assert.Contains(t, s, "x-param-ext")
}

// ---------------------------------------------------------------------------
// Components MarshalYAML: empty maps should not appear
// ---------------------------------------------------------------------------

func TestComponents_MarshalYAML_EmptyMaps(t *testing.T) {
	comp := &Components{}

	rendered, err := comp.Render()
	require.NoError(t, err)
	s := string(rendered)
	assert.Equal(t, "{}\n", s)
}

func TestComponents_MarshalYAML_OnlyInputs(t *testing.T) {
	inputs := orderedmap.New[string, *yaml.Node]()
	inputs.Set("myInput", &yaml.Node{Kind: yaml.ScalarNode, Value: "test"})
	comp := &Components{
		Inputs: inputs,
	}

	rendered, err := comp.Render()
	require.NoError(t, err)
	s := string(rendered)
	assert.Contains(t, s, "inputs:")
	assert.NotContains(t, s, "parameters:")
	assert.NotContains(t, s, "successActions:")
	assert.NotContains(t, s, "failureActions:")
}

// ---------------------------------------------------------------------------
// Step MarshalYAML: all fields (outputs, parameters, criteria, etc.)
// ---------------------------------------------------------------------------

func TestStep_MarshalYAML_AllFields(t *testing.T) {
	yml := `arazzo: 1.0.1
info:
  title: Test
  version: 0.1.0
sourceDescriptions:
  - name: api
    url: https://example.com
workflows:
  - workflowId: wf1
    steps:
      - stepId: fullStep
        operationId: op1
        description: Full step
        parameters:
          - name: p1
            in: query
            value: v1
        requestBody:
          contentType: application/json
          payload:
            key: val
        successCriteria:
          - condition: $statusCode == 200
        onSuccess:
          - name: done
            type: end
        onFailure:
          - name: retry
            type: retry
            retryAfter: 1.0
            retryLimit: 2
        outputs:
          result: $response.body`

	h := buildHighFromYAML(t, yml)
	step := h.Workflows[0].Steps[0]

	rendered, err := step.Render()
	require.NoError(t, err)
	s := string(rendered)

	assert.Contains(t, s, "stepId: fullStep")
	assert.Contains(t, s, "operationId: op1")
	assert.Contains(t, s, "description:")
	assert.Contains(t, s, "parameters:")
	assert.Contains(t, s, "requestBody:")
	assert.Contains(t, s, "successCriteria:")
	assert.Contains(t, s, "onSuccess:")
	assert.Contains(t, s, "onFailure:")
	assert.Contains(t, s, "outputs:")
}

func TestStep_MarshalYAML_WithWorkflowId(t *testing.T) {
	yml := `arazzo: 1.0.1
info:
  title: Test
  version: 0.1.0
sourceDescriptions:
  - name: api
    url: https://example.com
workflows:
  - workflowId: wf1
    steps:
      - stepId: call
        workflowId: other-wf`

	h := buildHighFromYAML(t, yml)
	step := h.Workflows[0].Steps[0]

	rendered, err := step.Render()
	require.NoError(t, err)
	s := string(rendered)

	assert.Contains(t, s, "workflowId:")
	assert.NotContains(t, s, "operationId:")
	assert.NotContains(t, s, "operationPath:")
}

func TestStep_MarshalYAML_WithOperationPath(t *testing.T) {
	yml := `arazzo: 1.0.1
info:
  title: Test
  version: 0.1.0
sourceDescriptions:
  - name: api
    url: https://example.com
workflows:
  - workflowId: wf1
    steps:
      - stepId: s1
        operationPath: "{$sourceDescriptions.api}/pets"`

	h := buildHighFromYAML(t, yml)
	step := h.Workflows[0].Steps[0]

	rendered, err := step.Render()
	require.NoError(t, err)
	s := string(rendered)

	assert.Contains(t, s, "operationPath:")
	assert.NotContains(t, s, "operationId:")
}

// ---------------------------------------------------------------------------
// Workflow MarshalYAML: all fields
// ---------------------------------------------------------------------------

func TestWorkflow_MarshalYAML_AllFields(t *testing.T) {
	yml := `arazzo: 1.0.1
info:
  title: Test
  version: 0.1.0
sourceDescriptions:
  - name: api
    url: https://example.com
workflows:
  - workflowId: fullWf
    summary: Full workflow
    description: Described
    dependsOn:
      - otherWf
    inputs:
      type: object
    steps:
      - stepId: s1
        operationId: op1
    successActions:
      - name: done
        type: end
    failureActions:
      - name: retry
        type: retry
        retryAfter: 1.0
        retryLimit: 2
    outputs:
      result: $steps.s1.outputs.r
    parameters:
      - name: pk
        in: query
        value: val`

	h := buildHighFromYAML(t, yml)
	wf := h.Workflows[0]

	rendered, err := wf.Render()
	require.NoError(t, err)
	s := string(rendered)

	assert.Contains(t, s, "workflowId: fullWf")
	assert.Contains(t, s, "summary:")
	assert.Contains(t, s, "description:")
	assert.Contains(t, s, "dependsOn:")
	assert.Contains(t, s, "inputs:")
	assert.Contains(t, s, "steps:")
	assert.Contains(t, s, "successActions:")
	assert.Contains(t, s, "failureActions:")
	assert.Contains(t, s, "outputs:")
	assert.Contains(t, s, "parameters:")
}

// ---------------------------------------------------------------------------
// Workflow MarshalYAML field ordering
// ---------------------------------------------------------------------------

func TestWorkflow_MarshalYAML_FieldOrdering_Full(t *testing.T) {
	yml := `arazzo: 1.0.1
info:
  title: Test
  version: 0.1.0
sourceDescriptions:
  - name: api
    url: https://example.com
workflows:
  - workflowId: ordered
    summary: sum
    description: desc
    steps:
      - stepId: s1
        operationId: op1
    outputs:
      r: v`

	h := buildHighFromYAML(t, yml)
	rendered, err := h.Workflows[0].Render()
	require.NoError(t, err)
	s := string(rendered)

	wfIdIdx := strings.Index(s, "workflowId:")
	sumIdx := strings.Index(s, "summary:")
	descIdx := strings.Index(s, "description:")

	assert.True(t, wfIdIdx < sumIdx)
	assert.True(t, sumIdx < descIdx)
}

// ---------------------------------------------------------------------------
// SuccessAction MarshalYAML: with criteria, workflowId, stepId
// ---------------------------------------------------------------------------

func TestSuccessAction_MarshalYAML_AllFields(t *testing.T) {
	sa := &SuccessAction{
		Name:       "goTo",
		Type:       "goto",
		WorkflowId: "otherWf",
		StepId:     "step2",
		Criteria: []*Criterion{
			{Condition: "$statusCode == 200"},
		},
	}

	rendered, err := sa.Render()
	require.NoError(t, err)
	s := string(rendered)

	assert.Contains(t, s, "name: goTo")
	assert.Contains(t, s, "type: goto")
	assert.Contains(t, s, "workflowId:")
	assert.Contains(t, s, "stepId:")
	assert.Contains(t, s, "criteria:")
}

// ---------------------------------------------------------------------------
// FailureAction MarshalYAML: with criteria, workflowId, stepId
// ---------------------------------------------------------------------------

func TestFailureAction_MarshalYAML_AllFields(t *testing.T) {
	fa := &FailureAction{
		Name:       "retryAction",
		Type:       "retry",
		WorkflowId: "otherWf",
		StepId:     "step2",
		RetryAfter: ptrFloat64(2.5),
		RetryLimit: ptrInt64(10),
		Criteria: []*Criterion{
			{Condition: "$statusCode == 503"},
		},
	}

	rendered, err := fa.Render()
	require.NoError(t, err)
	s := string(rendered)

	assert.Contains(t, s, "name: retryAction")
	assert.Contains(t, s, "type: retry")
	assert.Contains(t, s, "workflowId:")
	assert.Contains(t, s, "stepId:")
	assert.Contains(t, s, "retryAfter:")
	assert.Contains(t, s, "retryLimit:")
	assert.Contains(t, s, "criteria:")
}

// ---------------------------------------------------------------------------
// RequestBody MarshalYAML: empty replacements should not appear
// ---------------------------------------------------------------------------

func TestRequestBody_MarshalYAML_NoReplacements(t *testing.T) {
	rb := &RequestBody{
		ContentType: "application/json",
		Payload:     &yaml.Node{Kind: yaml.ScalarNode, Value: "data"},
	}

	rendered, err := rb.Render()
	require.NoError(t, err)
	s := string(rendered)

	assert.Contains(t, s, "contentType:")
	assert.Contains(t, s, "payload:")
	assert.NotContains(t, s, "replacements:")
}

// ---------------------------------------------------------------------------
// PayloadReplacement MarshalYAML: nil value should not appear
// ---------------------------------------------------------------------------

func TestPayloadReplacement_MarshalYAML_NilValue(t *testing.T) {
	pr := &PayloadReplacement{
		Target: "/path",
	}

	rendered, err := pr.Render()
	require.NoError(t, err)
	s := string(rendered)

	assert.Contains(t, s, "target: /path")
	assert.NotContains(t, s, "value:")
}

// ---------------------------------------------------------------------------
// Arazzo MarshalYAML: minimal (no components)
// ---------------------------------------------------------------------------

func TestArazzo_MarshalYAML_Minimal(t *testing.T) {
	a := &Arazzo{
		Arazzo: "1.0.1",
		Info: &Info{
			Title:   "Test",
			Version: "0.1.0",
		},
	}

	rendered, err := a.Render()
	require.NoError(t, err)
	s := string(rendered)

	assert.Contains(t, s, "arazzo: 1.0.1")
	assert.Contains(t, s, "info:")
	assert.NotContains(t, s, "sourceDescriptions:")
	assert.NotContains(t, s, "workflows:")
	assert.NotContains(t, s, "components:")
}

// ---------------------------------------------------------------------------
// Workflow MarshalYAML: empty outputs (nil) should not appear
// ---------------------------------------------------------------------------

func TestWorkflow_MarshalYAML_NilOutputs(t *testing.T) {
	wf := &Workflow{
		WorkflowId: "wf1",
	}

	rendered, err := wf.Render()
	require.NoError(t, err)
	s := string(rendered)

	assert.Contains(t, s, "workflowId: wf1")
	assert.NotContains(t, s, "outputs:")
}

func TestWorkflow_MarshalYAML_EmptyOutputs(t *testing.T) {
	wf := &Workflow{
		WorkflowId: "wf1",
		Outputs:    orderedmap.New[string, string](),
	}

	rendered, err := wf.Render()
	require.NoError(t, err)
	s := string(rendered)

	// Empty outputs map (Len() == 0) should not appear
	assert.NotContains(t, s, "outputs:")
}

// ---------------------------------------------------------------------------
// Step MarshalYAML: nil outputs should not appear
// ---------------------------------------------------------------------------

func TestStep_MarshalYAML_NilOutputs(t *testing.T) {
	step := &Step{
		StepId:      "s1",
		OperationId: "op1",
	}

	rendered, err := step.Render()
	require.NoError(t, err)
	s := string(rendered)

	assert.NotContains(t, s, "outputs:")
}

func TestStep_MarshalYAML_EmptyOutputs(t *testing.T) {
	step := &Step{
		StepId:      "s1",
		OperationId: "op1",
		Outputs:     orderedmap.New[string, string](),
	}

	rendered, err := step.Render()
	require.NoError(t, err)
	s := string(rendered)

	// Empty outputs should not appear
	assert.NotContains(t, s, "outputs:")
}

// ---------------------------------------------------------------------------
// Info MarshalYAML: minimal (only required fields)
// ---------------------------------------------------------------------------

func TestInfo_MarshalYAML_Minimal(t *testing.T) {
	info := &Info{
		Title:   "Minimal",
		Version: "0.0.1",
	}

	rendered, err := info.Render()
	require.NoError(t, err)
	s := string(rendered)

	assert.Contains(t, s, "title: Minimal")
	assert.Contains(t, s, "version: 0.0.1")
	assert.NotContains(t, s, "summary:")
	assert.NotContains(t, s, "description:")
}

// ---------------------------------------------------------------------------
// SourceDescription MarshalYAML: without type
// ---------------------------------------------------------------------------

func TestSourceDescription_MarshalYAML_NoType(t *testing.T) {
	sd := &SourceDescription{
		Name: "api",
		URL:  "https://example.com",
	}

	rendered, err := sd.Render()
	require.NoError(t, err)
	s := string(rendered)

	assert.Contains(t, s, "name: api")
	assert.Contains(t, s, "url:")
	assert.NotContains(t, s, "type:")
}

// ---------------------------------------------------------------------------
// CriterionExpressionType MarshalYAML: minimal (no version)
// ---------------------------------------------------------------------------

func TestCriterionExpressionType_MarshalYAML_Minimal(t *testing.T) {
	cet := &CriterionExpressionType{
		Type: "jsonpath",
	}

	rendered, err := cet.Render()
	require.NoError(t, err)
	s := string(rendered)

	assert.Contains(t, s, "type: jsonpath")
	assert.NotContains(t, s, "version:")
}

// ---------------------------------------------------------------------------
// Parameter MarshalYAML: reusable parameter
// ---------------------------------------------------------------------------

func TestParameter_MarshalYAML_Reusable(t *testing.T) {
	p := &Parameter{
		Reference: "$components.parameters.myParam",
		Value:     &yaml.Node{Kind: yaml.ScalarNode, Value: "override"},
	}

	rendered, err := p.Render()
	require.NoError(t, err)
	s := string(rendered)

	assert.Contains(t, s, "reference:")
	assert.Contains(t, s, "value:")
	// Reusable should not include name/in
	assert.NotContains(t, s, "name:")
	assert.NotContains(t, s, "in:")
}

func TestParameter_MarshalYAML_ReusableWithoutValue(t *testing.T) {
	p := &Parameter{
		Reference: "$components.parameters.myParam",
	}

	rendered, err := p.Render()
	require.NoError(t, err)
	s := string(rendered)

	assert.Contains(t, s, "reference:")
	assert.NotContains(t, s, "value:")
}

// ---------------------------------------------------------------------------
// SuccessAction MarshalYAML: reusable
// ---------------------------------------------------------------------------

func TestSuccessAction_MarshalYAML_Reusable(t *testing.T) {
	sa := &SuccessAction{
		Reference: "$components.successActions.myAction",
	}

	rendered, err := sa.Render()
	require.NoError(t, err)
	s := string(rendered)

	assert.Contains(t, s, "reference:")
	assert.NotContains(t, s, "name:")
	assert.NotContains(t, s, "type:")
}

// ---------------------------------------------------------------------------
// Components MarshalYAML: all maps populated
// ---------------------------------------------------------------------------

func TestComponents_MarshalYAML_AllMaps(t *testing.T) {
	inputs := orderedmap.New[string, *yaml.Node]()
	inputs.Set("in1", &yaml.Node{Kind: yaml.ScalarNode, Value: "test"})

	params := orderedmap.New[string, *Parameter]()
	params.Set("p1", &Parameter{Name: "key", In: "header"})

	successActions := orderedmap.New[string, *SuccessAction]()
	successActions.Set("sa1", &SuccessAction{Name: "done", Type: "end"})

	failureActions := orderedmap.New[string, *FailureAction]()
	failureActions.Set("fa1", &FailureAction{Name: "retry", Type: "retry"})

	comp := &Components{
		Inputs:         inputs,
		Parameters:     params,
		SuccessActions: successActions,
		FailureActions: failureActions,
	}

	rendered, err := comp.Render()
	require.NoError(t, err)
	s := string(rendered)

	assert.Contains(t, s, "inputs:")
	assert.Contains(t, s, "parameters:")
	assert.Contains(t, s, "successActions:")
	assert.Contains(t, s, "failureActions:")
}
