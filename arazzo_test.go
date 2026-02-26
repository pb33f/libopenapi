// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package libopenapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewArazzoDocument_ValidFull(t *testing.T) {
	yml := []byte(`arazzo: 1.0.1
info:
  title: Pet Store Workflows
  summary: Orchestrate pet store actions
  description: Full end-to-end pet store orchestration
  version: 1.0.0
sourceDescriptions:
  - name: petStoreApi
    url: https://petstore.swagger.io/v2/swagger.json
    type: openapi
workflows:
  - workflowId: createPet
    summary: Create a new pet
    description: Creates a pet end-to-end
    steps:
      - stepId: addPet
        operationId: addPet
        parameters:
          - name: api_key
            in: header
            value: abc123
        requestBody:
          contentType: application/json
          payload:
            name: fluffy
        successCriteria:
          - condition: $statusCode == 200
        onSuccess:
          - name: done
            type: end
        onFailure:
          - name: retryOnce
            type: retry
            retryAfter: 1.0
            retryLimit: 1
        outputs:
          petId: $response.body#/id
    outputs:
      createdPetId: $steps.addPet.outputs.petId
components:
  parameters:
    apiKey:
      name: api_key
      in: header
      value: default-key
  successActions:
    logAndEnd:
      name: logAndEnd
      type: end
  failureActions:
    retryDefault:
      name: retryDefault
      type: retry
      retryAfter: 2.0
      retryLimit: 5
`)
	doc, err := NewArazzoDocument(yml)
	require.NoError(t, err)
	require.NotNil(t, doc)

	assert.Equal(t, "1.0.1", doc.Arazzo)
	require.NotNil(t, doc.Info)
	assert.Equal(t, "Pet Store Workflows", doc.Info.Title)
	assert.Equal(t, "Orchestrate pet store actions", doc.Info.Summary)
	assert.Equal(t, "Full end-to-end pet store orchestration", doc.Info.Description)
	assert.Equal(t, "1.0.0", doc.Info.Version)

	require.Len(t, doc.SourceDescriptions, 1)
	assert.Equal(t, "petStoreApi", doc.SourceDescriptions[0].Name)
	assert.Equal(t, "openapi", doc.SourceDescriptions[0].Type)

	require.Len(t, doc.Workflows, 1)
	wf := doc.Workflows[0]
	assert.Equal(t, "createPet", wf.WorkflowId)
	assert.Equal(t, "Create a new pet", wf.Summary)

	require.Len(t, wf.Steps, 1)
	step := wf.Steps[0]
	assert.Equal(t, "addPet", step.StepId)
	assert.Equal(t, "addPet", step.OperationId)
	require.Len(t, step.Parameters, 1)
	assert.Equal(t, "api_key", step.Parameters[0].Name)
	assert.NotNil(t, step.RequestBody)
	assert.Equal(t, "application/json", step.RequestBody.ContentType)
	require.Len(t, step.SuccessCriteria, 1)
	require.Len(t, step.OnSuccess, 1)
	require.Len(t, step.OnFailure, 1)

	require.NotNil(t, doc.Components)
	require.NotNil(t, doc.Components.Parameters)
	p, ok := doc.Components.Parameters.Get("apiKey")
	assert.True(t, ok)
	assert.Equal(t, "api_key", p.Name)

	require.NotNil(t, doc.Components.SuccessActions)
	sa, ok := doc.Components.SuccessActions.Get("logAndEnd")
	assert.True(t, ok)
	assert.Equal(t, "end", sa.Type)

	require.NotNil(t, doc.Components.FailureActions)
	fa, ok := doc.Components.FailureActions.Get("retryDefault")
	assert.True(t, ok)
	assert.Equal(t, "retry", fa.Type)
}

func TestNewArazzoDocument_Minimal(t *testing.T) {
	yml := []byte(`arazzo: 1.0.1
info:
  title: Minimal Arazzo
  version: 0.1.0
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.yaml
    type: openapi
workflows:
  - workflowId: simpleWorkflow
    steps:
      - stepId: step1
        operationId: getUser
`)
	doc, err := NewArazzoDocument(yml)
	require.NoError(t, err)
	require.NotNil(t, doc)

	assert.Equal(t, "1.0.1", doc.Arazzo)
	assert.Equal(t, "Minimal Arazzo", doc.Info.Title)
	assert.Equal(t, "0.1.0", doc.Info.Version)
	assert.Len(t, doc.SourceDescriptions, 1)
	assert.Len(t, doc.Workflows, 1)
	assert.Nil(t, doc.Components)
}

func TestNewArazzoDocument_InvalidYAML(t *testing.T) {
	yml := []byte(`{{{ not valid yaml`)
	doc, err := NewArazzoDocument(yml)
	assert.Error(t, err)
	assert.Nil(t, doc)
	assert.Contains(t, err.Error(), "failed to parse YAML")
}

func TestNewArazzoDocument_EmptyInput(t *testing.T) {
	doc, err := NewArazzoDocument([]byte{})
	assert.Error(t, err)
	assert.Nil(t, doc)
}

func TestNewArazzoDocument_ScalarYAML(t *testing.T) {
	// A scalar is not a mapping node
	yml := []byte(`just a string`)
	doc, err := NewArazzoDocument(yml)
	assert.Error(t, err)
	assert.Nil(t, doc)
	assert.Contains(t, err.Error(), "expected YAML mapping")
}

func TestNewArazzoDocument_ArrayYAML(t *testing.T) {
	// A sequence is not a mapping node
	yml := []byte(`- item1
- item2
`)
	doc, err := NewArazzoDocument(yml)
	assert.Error(t, err)
	assert.Nil(t, doc)
	assert.Contains(t, err.Error(), "expected YAML mapping")
}

func TestNewArazzoDocument_MultipleWorkflows(t *testing.T) {
	yml := []byte(`arazzo: 1.0.1
info:
  title: Multi-Workflow
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: https://example.com/api.yaml
workflows:
  - workflowId: workflow1
    steps:
      - stepId: s1
        operationId: op1
  - workflowId: workflow2
    dependsOn:
      - workflow1
    steps:
      - stepId: s2
        operationId: op2
  - workflowId: workflow3
    dependsOn:
      - workflow1
      - workflow2
    steps:
      - stepId: s3
        operationId: op3
`)
	doc, err := NewArazzoDocument(yml)
	require.NoError(t, err)
	require.NotNil(t, doc)

	assert.Len(t, doc.Workflows, 3)
	assert.Equal(t, "workflow1", doc.Workflows[0].WorkflowId)
	assert.Equal(t, "workflow2", doc.Workflows[1].WorkflowId)
	assert.Equal(t, "workflow3", doc.Workflows[2].WorkflowId)

	assert.Empty(t, doc.Workflows[0].DependsOn)
	assert.Equal(t, []string{"workflow1"}, doc.Workflows[1].DependsOn)
	assert.Equal(t, []string{"workflow1", "workflow2"}, doc.Workflows[2].DependsOn)
}

func TestNewArazzoDocument_MultipleSourceDescriptions(t *testing.T) {
	yml := []byte(`arazzo: 1.0.1
info:
  title: Multi-Source
  version: 1.0.0
sourceDescriptions:
  - name: primaryApi
    url: https://api.example.com/openapi.yaml
    type: openapi
  - name: secondaryApi
    url: https://other.example.com/openapi.json
    type: openapi
  - name: subWorkflows
    url: https://example.com/workflows.arazzo.yaml
    type: arazzo
workflows:
  - workflowId: combined
    steps:
      - stepId: fromPrimary
        operationId: getPrimary
`)
	doc, err := NewArazzoDocument(yml)
	require.NoError(t, err)
	require.Len(t, doc.SourceDescriptions, 3)
	assert.Equal(t, "primaryApi", doc.SourceDescriptions[0].Name)
	assert.Equal(t, "secondaryApi", doc.SourceDescriptions[1].Name)
	assert.Equal(t, "subWorkflows", doc.SourceDescriptions[2].Name)
	assert.Equal(t, "arazzo", doc.SourceDescriptions[2].Type)
}

func TestNewArazzoDocument_CriterionExpressionType(t *testing.T) {
	yml := []byte(`arazzo: 1.0.1
info:
  title: Criterion Test
  version: 1.0.0
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
            type: simple
          - condition: $.data.id != null
            context: $response.body
            type:
              type: jsonpath
              version: draft-goessner-dispatch-jsonpath-00
          - condition: "^2[0-9]{2}$"
            context: $statusCode
            type: regex
`)
	doc, err := NewArazzoDocument(yml)
	require.NoError(t, err)

	criteria := doc.Workflows[0].Steps[0].SuccessCriteria
	require.Len(t, criteria, 3)

	// Simple scalar type
	assert.Equal(t, "simple", criteria[0].Type)
	assert.Nil(t, criteria[0].ExpressionType)
	assert.Equal(t, "simple", criteria[0].GetEffectiveType())

	// Mapping CriterionExpressionType
	assert.Empty(t, criteria[1].Type)
	require.NotNil(t, criteria[1].ExpressionType)
	assert.Equal(t, "jsonpath", criteria[1].ExpressionType.Type)
	assert.Equal(t, "jsonpath", criteria[1].GetEffectiveType())

	// Regex scalar type
	assert.Equal(t, "regex", criteria[2].Type)
	assert.Nil(t, criteria[2].ExpressionType)
	assert.Equal(t, "regex", criteria[2].GetEffectiveType())
}

func TestNewArazzoDocument_WithExtensions(t *testing.T) {
	yml := []byte(`arazzo: 1.0.1
info:
  title: Extension Test
  version: 1.0.0
  x-info-ext: value1
sourceDescriptions:
  - name: api
    url: https://example.com
    x-source-ext: value2
workflows:
  - workflowId: wf1
    steps:
      - stepId: s1
        operationId: op1
x-root-ext: value3
`)
	doc, err := NewArazzoDocument(yml)
	require.NoError(t, err)

	// Root extensions
	require.NotNil(t, doc.Extensions)
	rootExt, ok := doc.Extensions.Get("x-root-ext")
	assert.True(t, ok)
	assert.Equal(t, "value3", rootExt.Value)

	// Info extensions
	require.NotNil(t, doc.Info.Extensions)
	infoExt, ok := doc.Info.Extensions.Get("x-info-ext")
	assert.True(t, ok)
	assert.Equal(t, "value1", infoExt.Value)
}

func TestNewArazzoDocument_ReusableObjects(t *testing.T) {
	yml := []byte(`arazzo: 1.0.1
info:
  title: Reusable Test
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: https://example.com
workflows:
  - workflowId: wf1
    steps:
      - stepId: s1
        operationId: op1
        parameters:
          - reference: $components.parameters.sharedParam
            value: overridden
        onSuccess:
          - reference: $components.successActions.logAndEnd
        onFailure:
          - reference: $components.failureActions.retryDefault
components:
  parameters:
    sharedParam:
      name: shared
      in: header
      value: default
  successActions:
    logAndEnd:
      name: logAndEnd
      type: end
  failureActions:
    retryDefault:
      name: retryDefault
      type: retry
`)
	doc, err := NewArazzoDocument(yml)
	require.NoError(t, err)

	step := doc.Workflows[0].Steps[0]

	// Reusable parameter
	require.Len(t, step.Parameters, 1)
	assert.True(t, step.Parameters[0].IsReusable())
	assert.Equal(t, "$components.parameters.sharedParam", step.Parameters[0].Reference)

	// Reusable success action
	require.Len(t, step.OnSuccess, 1)
	assert.True(t, step.OnSuccess[0].IsReusable())
	assert.Equal(t, "$components.successActions.logAndEnd", step.OnSuccess[0].Reference)

	// Reusable failure action
	require.Len(t, step.OnFailure, 1)
	assert.True(t, step.OnFailure[0].IsReusable())
	assert.Equal(t, "$components.failureActions.retryDefault", step.OnFailure[0].Reference)
}

func TestNewArazzoDocument_GoLowAccess(t *testing.T) {
	yml := []byte(`arazzo: 1.0.1
info:
  title: GoLow Test
  version: 0.1.0
sourceDescriptions:
  - name: api
    url: https://example.com
workflows:
  - workflowId: wf1
    steps:
      - stepId: s1
        operationId: op1
`)
	doc, err := NewArazzoDocument(yml)
	require.NoError(t, err)

	lowDoc := doc.GoLow()
	assert.NotNil(t, lowDoc)
	assert.Equal(t, "1.0.1", lowDoc.Arazzo.Value)
	assert.Equal(t, "GoLow Test", lowDoc.Info.Value.Title.Value)
}

func TestNewArazzoDocument_Render(t *testing.T) {
	yml := []byte(`arazzo: 1.0.1
info:
  title: Render Test
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: https://example.com
workflows:
  - workflowId: wf1
    steps:
      - stepId: s1
        operationId: op1
`)
	doc, err := NewArazzoDocument(yml)
	require.NoError(t, err)

	rendered, err := doc.Render()
	require.NoError(t, err)
	assert.Contains(t, string(rendered), "arazzo: 1.0.1")
	assert.Contains(t, string(rendered), "title: Render Test")
}

func TestNewArazzoDocument_RoundTrip(t *testing.T) {
	yml := []byte(`arazzo: 1.0.1
info:
  title: RoundTrip Test
  version: 2.0.0
sourceDescriptions:
  - name: myApi
    url: https://example.com/api.yaml
    type: openapi
workflows:
  - workflowId: roundTripWf
    summary: A round-trip workflow
    steps:
      - stepId: firstStep
        operationId: doSomething
        parameters:
          - name: token
            in: header
            value: secret
`)
	doc1, err := NewArazzoDocument(yml)
	require.NoError(t, err)

	rendered, err := doc1.Render()
	require.NoError(t, err)

	doc2, err := NewArazzoDocument(rendered)
	require.NoError(t, err)

	assert.Equal(t, doc1.Arazzo, doc2.Arazzo)
	assert.Equal(t, doc1.Info.Title, doc2.Info.Title)
	assert.Equal(t, doc1.Info.Version, doc2.Info.Version)
	assert.Len(t, doc2.SourceDescriptions, len(doc1.SourceDescriptions))
	assert.Len(t, doc2.Workflows, len(doc1.Workflows))
	assert.Equal(t, doc1.Workflows[0].WorkflowId, doc2.Workflows[0].WorkflowId)
}
