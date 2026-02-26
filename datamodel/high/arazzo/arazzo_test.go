// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"context"
	"strings"
	"testing"

	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	lowmodel "github.com/pb33f/libopenapi/datamodel/low"
	low "github.com/pb33f/libopenapi/datamodel/low/arazzo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

// buildHighArazzo is a test helper that parses YAML, builds the low-level model, then creates
// the high-level model.
func buildHighArazzo(t *testing.T, yml string) *Arazzo {
	t.Helper()
	var rootNode yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yml), &rootNode))
	require.Equal(t, yaml.DocumentNode, rootNode.Kind)
	require.NotEmpty(t, rootNode.Content)

	mappingNode := rootNode.Content[0]

	lowDoc := &low.Arazzo{}
	require.NoError(t, lowmodel.BuildModel(mappingNode, lowDoc))
	require.NoError(t, lowDoc.Build(context.Background(), nil, mappingNode, nil))

	return NewArazzo(lowDoc)
}

const fullArazzoYAML = `arazzo: 1.0.1
info:
  title: Pet Store Workflows
  summary: Orchestration for the pet store
  description: Demonstrates pet store API orchestration
  version: 1.0.0
sourceDescriptions:
  - name: petStoreApi
    url: https://petstore.swagger.io/v2/swagger.json
    type: openapi
  - name: arazzoWorkflows
    url: https://example.com/workflows.arazzo.yaml
    type: arazzo
workflows:
  - workflowId: createPet
    summary: Create a new pet
    description: Full workflow to create a pet and verify it
    dependsOn:
      - verifyPet
    inputs:
      type: object
      properties:
        petName:
          type: string
    steps:
      - stepId: addPet
        operationId: addPet
        description: Add a new pet to the store
        parameters:
          - name: api_key
            in: header
            value: abc123
        requestBody:
          contentType: application/json
          payload:
            name: fluffy
            status: available
          replacements:
            - target: /name
              value: replaced-name
        successCriteria:
          - condition: $statusCode == 200
            type: simple
          - condition: $response.body#/id != null
            context: $response.body
            type:
              type: jsonpath
              version: draft-goessner-dispatch-jsonpath-00
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
      - stepId: getPet
        operationPath: '{$sourceDescriptions.petStoreApi}/pet/{$steps.addPet.outputs.petId}'
    successActions:
      - name: notifySuccess
        type: goto
        stepId: addPet
    failureActions:
      - name: notifyFailure
        type: end
    outputs:
      createdPetId: $steps.addPet.outputs.petId
    parameters:
      - name: store_id
        in: query
        value: store-1
  - workflowId: verifyPet
    summary: Verify a pet exists
    steps:
      - stepId: checkPet
        operationId: getPetById
components:
  inputs:
    petInput:
      type: object
      properties:
        name:
          type: string
  parameters:
    apiKeyParam:
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
`

// ---------------------------------------------------------------------------
// Arazzo (root document)
// ---------------------------------------------------------------------------

func TestNewArazzo_FullDocument(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)

	assert.Equal(t, "1.0.1", h.Arazzo)
	assert.NotNil(t, h.Info)
	assert.Len(t, h.SourceDescriptions, 2)
	assert.Len(t, h.Workflows, 2)
	assert.NotNil(t, h.Components)
}

func TestArazzo_GoLow(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	assert.NotNil(t, h.GoLow())
	assert.IsType(t, &low.Arazzo{}, h.GoLow())
}

func TestArazzo_GoLowUntyped(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	untyped := h.GoLowUntyped()
	assert.NotNil(t, untyped)
	_, ok := untyped.(*low.Arazzo)
	assert.True(t, ok)
}

func TestArazzo_Render(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	rendered, err := h.Render()
	require.NoError(t, err)
	assert.Contains(t, string(rendered), "arazzo: 1.0.1")
	assert.Contains(t, string(rendered), "info:")
	assert.Contains(t, string(rendered), "sourceDescriptions:")
	assert.Contains(t, string(rendered), "workflows:")
	assert.Contains(t, string(rendered), "components:")
}

func TestArazzo_MarshalYAML_FieldOrdering(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	rendered, err := h.Render()
	require.NoError(t, err)

	s := string(rendered)
	arazzoIdx := strings.Index(s, "arazzo:")
	infoIdx := strings.Index(s, "info:")
	sdIdx := strings.Index(s, "sourceDescriptions:")
	wfIdx := strings.Index(s, "workflows:")
	compIdx := strings.Index(s, "components:")

	// Verify field ordering: arazzo, info, sourceDescriptions, workflows, components
	assert.True(t, arazzoIdx < infoIdx, "arazzo should come before info")
	assert.True(t, infoIdx < sdIdx, "info should come before sourceDescriptions")
	assert.True(t, sdIdx < wfIdx, "sourceDescriptions should come before workflows")
	assert.True(t, wfIdx < compIdx, "workflows should come before components")
}

func TestArazzo_RoundTrip(t *testing.T) {
	h1 := buildHighArazzo(t, fullArazzoYAML)
	rendered1, err := h1.Render()
	require.NoError(t, err)

	// Parse the rendered output again
	var rootNode yaml.Node
	require.NoError(t, yaml.Unmarshal(rendered1, &rootNode))
	lowDoc := &low.Arazzo{}
	require.NoError(t, lowmodel.BuildModel(rootNode.Content[0], lowDoc))
	require.NoError(t, lowDoc.Build(context.Background(), nil, rootNode.Content[0], nil))
	h2 := NewArazzo(lowDoc)

	assert.Equal(t, h1.Arazzo, h2.Arazzo)
	assert.Equal(t, h1.Info.Title, h2.Info.Title)
	assert.Equal(t, h1.Info.Version, h2.Info.Version)
	assert.Len(t, h2.SourceDescriptions, len(h1.SourceDescriptions))
	assert.Len(t, h2.Workflows, len(h1.Workflows))
}

func TestArazzo_AddOpenAPISourceDocument(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	doc1 := &v3.Document{Version: "3.1.0"}
	doc2 := &v3.Document{Version: "3.0.3"}

	h.AddOpenAPISourceDocument(nil, doc1)
	h.AddOpenAPISourceDocument(doc2)

	docs := h.GetOpenAPISourceDocuments()
	require.Len(t, docs, 2)
	assert.Same(t, doc1, docs[0])
	assert.Same(t, doc2, docs[1])
}

func TestArazzo_GetOpenAPISourceDocuments_ReturnsCopy(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	doc1 := &v3.Document{Version: "3.1.0"}
	doc2 := &v3.Document{Version: "3.0.3"}

	h.AddOpenAPISourceDocument(doc1, doc2)
	docs := h.GetOpenAPISourceDocuments()
	require.Len(t, docs, 2)

	docs[0] = nil
	after := h.GetOpenAPISourceDocuments()
	require.Len(t, after, 2)
	assert.Same(t, doc1, after[0])
	assert.Same(t, doc2, after[1])
}

func TestArazzo_AddOpenAPISourceDocument_NilReceiver(t *testing.T) {
	var h *Arazzo
	h.AddOpenAPISourceDocument(&v3.Document{Version: "3.1.0"})
	assert.Nil(t, h.GetOpenAPISourceDocuments())
}

func TestArazzo_MinimalDocument(t *testing.T) {
	yml := `arazzo: 1.0.1
info:
  title: Minimal
  version: 0.1.0
sourceDescriptions:
  - name: api
    url: https://example.com/api.yaml
    type: openapi
workflows:
  - workflowId: simple
    steps:
      - stepId: one
        operationId: doSomething
`
	h := buildHighArazzo(t, yml)
	assert.Equal(t, "1.0.1", h.Arazzo)
	assert.Equal(t, "Minimal", h.Info.Title)
	assert.Len(t, h.SourceDescriptions, 1)
	assert.Len(t, h.Workflows, 1)
	assert.Nil(t, h.Components)
}

func TestArazzo_EmptyComponents(t *testing.T) {
	yml := `arazzo: 1.0.1
info:
  title: Test
  version: 0.1.0
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.yaml
workflows:
  - workflowId: wf1
    steps:
      - stepId: s1
        operationId: op1
components: {}
`
	// Components object exists but is empty
	h := buildHighArazzo(t, yml)
	// Even an empty mapping is extracted; verify no crash
	assert.NotNil(t, h)
}

// ---------------------------------------------------------------------------
// Info
// ---------------------------------------------------------------------------

func TestNewInfo_AllFields(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	info := h.Info
	require.NotNil(t, info)
	assert.Equal(t, "Pet Store Workflows", info.Title)
	assert.Equal(t, "Orchestration for the pet store", info.Summary)
	assert.Equal(t, "Demonstrates pet store API orchestration", info.Description)
	assert.Equal(t, "1.0.0", info.Version)
}

func TestInfo_GoLow(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	assert.NotNil(t, h.Info.GoLow())
}

func TestInfo_GoLowUntyped(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	assert.NotNil(t, h.Info.GoLowUntyped())
}

func TestInfo_Render(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	rendered, err := h.Info.Render()
	require.NoError(t, err)
	assert.Contains(t, string(rendered), "title: Pet Store Workflows")
	assert.Contains(t, string(rendered), "version: 1.0.0")
}

func TestInfo_MarshalYAML_FieldOrdering(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	rendered, err := h.Info.Render()
	require.NoError(t, err)

	s := string(rendered)
	titleIdx := strings.Index(s, "title:")
	summaryIdx := strings.Index(s, "summary:")
	descIdx := strings.Index(s, "description:")
	versionIdx := strings.Index(s, "version:")

	assert.True(t, titleIdx < summaryIdx)
	assert.True(t, summaryIdx < descIdx)
	assert.True(t, descIdx < versionIdx)
}

func TestInfo_MinimalFields(t *testing.T) {
	yml := `arazzo: 1.0.1
info:
  title: Minimal
  version: 0.0.1
sourceDescriptions:
  - name: api
    url: https://example.com
workflows:
  - workflowId: wf
    steps:
      - stepId: s1
        operationId: op
`
	h := buildHighArazzo(t, yml)
	assert.Equal(t, "Minimal", h.Info.Title)
	assert.Equal(t, "0.0.1", h.Info.Version)
	assert.Empty(t, h.Info.Summary)
	assert.Empty(t, h.Info.Description)
}

// ---------------------------------------------------------------------------
// SourceDescription
// ---------------------------------------------------------------------------

func TestNewSourceDescription_AllFields(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	require.Len(t, h.SourceDescriptions, 2)

	sd1 := h.SourceDescriptions[0]
	assert.Equal(t, "petStoreApi", sd1.Name)
	assert.Equal(t, "https://petstore.swagger.io/v2/swagger.json", sd1.URL)
	assert.Equal(t, "openapi", sd1.Type)

	sd2 := h.SourceDescriptions[1]
	assert.Equal(t, "arazzoWorkflows", sd2.Name)
	assert.Equal(t, "arazzo", sd2.Type)
}

func TestSourceDescription_GoLow(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	assert.NotNil(t, h.SourceDescriptions[0].GoLow())
}

func TestSourceDescription_GoLowUntyped(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	assert.NotNil(t, h.SourceDescriptions[0].GoLowUntyped())
}

func TestSourceDescription_Render(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	rendered, err := h.SourceDescriptions[0].Render()
	require.NoError(t, err)
	assert.Contains(t, string(rendered), "name: petStoreApi")
	assert.Contains(t, string(rendered), "url:")
	assert.Contains(t, string(rendered), "type: openapi")
}

// ---------------------------------------------------------------------------
// Workflow
// ---------------------------------------------------------------------------

func TestNewWorkflow_AllFields(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	require.Len(t, h.Workflows, 2)

	wf := h.Workflows[0]
	assert.Equal(t, "createPet", wf.WorkflowId)
	assert.Equal(t, "Create a new pet", wf.Summary)
	assert.Equal(t, "Full workflow to create a pet and verify it", wf.Description)
	assert.NotNil(t, wf.Inputs)
	assert.Equal(t, []string{"verifyPet"}, wf.DependsOn)
	assert.Len(t, wf.Steps, 2)
	assert.Len(t, wf.SuccessActions, 1)
	assert.Len(t, wf.FailureActions, 1)
	assert.NotNil(t, wf.Outputs)
	assert.Len(t, wf.Parameters, 1)
}

func TestWorkflow_GoLow(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	assert.NotNil(t, h.Workflows[0].GoLow())
}

func TestWorkflow_GoLowUntyped(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	assert.NotNil(t, h.Workflows[0].GoLowUntyped())
}

func TestWorkflow_Render(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	rendered, err := h.Workflows[0].Render()
	require.NoError(t, err)
	s := string(rendered)
	assert.Contains(t, s, "workflowId: createPet")
	assert.Contains(t, s, "steps:")
}

func TestWorkflow_MarshalYAML_FieldOrdering(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	rendered, err := h.Workflows[0].Render()
	require.NoError(t, err)

	s := string(rendered)
	wfIdIdx := strings.Index(s, "workflowId:")
	stepsIdx := strings.Index(s, "steps:")

	assert.True(t, wfIdIdx < stepsIdx)
}

func TestWorkflow_Outputs(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	wf := h.Workflows[0]
	require.NotNil(t, wf.Outputs)
	val, ok := wf.Outputs.Get("createdPetId")
	assert.True(t, ok)
	assert.Equal(t, "$steps.addPet.outputs.petId", val)
}

// ---------------------------------------------------------------------------
// Step
// ---------------------------------------------------------------------------

func TestNewStep_AllFields(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	step := h.Workflows[0].Steps[0]

	assert.Equal(t, "addPet", step.StepId)
	assert.Equal(t, "addPet", step.OperationId)
	assert.Equal(t, "Add a new pet to the store", step.Description)
	assert.Empty(t, step.OperationPath)
	assert.Empty(t, step.WorkflowId)
	assert.Len(t, step.Parameters, 1)
	assert.NotNil(t, step.RequestBody)
	assert.Len(t, step.SuccessCriteria, 2)
	assert.Len(t, step.OnSuccess, 1)
	assert.Len(t, step.OnFailure, 1)
	assert.NotNil(t, step.Outputs)
}

func TestStep_WithOperationPath(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	step := h.Workflows[0].Steps[1]

	assert.Equal(t, "getPet", step.StepId)
	assert.NotEmpty(t, step.OperationPath)
	assert.Empty(t, step.OperationId)
}

func TestStep_GoLow(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	assert.NotNil(t, h.Workflows[0].Steps[0].GoLow())
}

func TestStep_GoLowUntyped(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	assert.NotNil(t, h.Workflows[0].Steps[0].GoLowUntyped())
}

func TestStep_Render(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	rendered, err := h.Workflows[0].Steps[0].Render()
	require.NoError(t, err)
	s := string(rendered)
	assert.Contains(t, s, "stepId: addPet")
	assert.Contains(t, s, "operationId: addPet")
}

func TestStep_Outputs(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	step := h.Workflows[0].Steps[0]
	require.NotNil(t, step.Outputs)
	val, ok := step.Outputs.Get("petId")
	assert.True(t, ok)
	assert.Equal(t, "$response.body#/id", val)
}

// ---------------------------------------------------------------------------
// Parameter
// ---------------------------------------------------------------------------

func TestNewParameter_AllFields(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	param := h.Workflows[0].Steps[0].Parameters[0]

	assert.Equal(t, "api_key", param.Name)
	assert.Equal(t, "header", param.In)
	assert.NotNil(t, param.Value)
	assert.Equal(t, "abc123", param.Value.Value)
	assert.Empty(t, param.Reference)
}

func TestParameter_IsReusable_False(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	param := h.Workflows[0].Steps[0].Parameters[0]
	assert.False(t, param.IsReusable())
}

func TestParameter_IsReusable_True(t *testing.T) {
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
          - reference: $components.parameters.apiKeyParam
            value: override-value
components:
  parameters:
    apiKeyParam:
      name: api_key
      in: header
      value: default-key
`
	h := buildHighArazzo(t, yml)
	param := h.Workflows[0].Steps[0].Parameters[0]
	assert.True(t, param.IsReusable())
	assert.Equal(t, "$components.parameters.apiKeyParam", param.Reference)
}

func TestParameter_GoLow(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	assert.NotNil(t, h.Workflows[0].Steps[0].Parameters[0].GoLow())
}

func TestParameter_GoLowUntyped(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	assert.NotNil(t, h.Workflows[0].Steps[0].Parameters[0].GoLowUntyped())
}

func TestParameter_Render(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	rendered, err := h.Workflows[0].Steps[0].Parameters[0].Render()
	require.NoError(t, err)
	s := string(rendered)
	assert.Contains(t, s, "name: api_key")
	assert.Contains(t, s, "in: header")
}

func TestParameter_Render_Reusable(t *testing.T) {
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
          - reference: $components.parameters.apiKeyParam
            value: override-value
components:
  parameters:
    apiKeyParam:
      name: api_key
      in: header
      value: default-key
`
	h := buildHighArazzo(t, yml)
	rendered, err := h.Workflows[0].Steps[0].Parameters[0].Render()
	require.NoError(t, err)
	s := string(rendered)
	// Reusable params render reference first, no name/in
	assert.Contains(t, s, "reference:")
}

// ---------------------------------------------------------------------------
// Criterion
// ---------------------------------------------------------------------------

func TestNewCriterion_ScalarSimple(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	criteria := h.Workflows[0].Steps[0].SuccessCriteria
	require.Len(t, criteria, 2)

	c := criteria[0]
	assert.Equal(t, "$statusCode == 200", c.Condition)
	assert.Equal(t, "simple", c.Type)
	assert.Nil(t, c.ExpressionType)
	assert.Equal(t, "simple", c.GetEffectiveType())
}

func TestNewCriterion_ScalarRegex(t *testing.T) {
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
          - condition: "^2[0-9]{2}$"
            context: $statusCode
            type: regex
`
	h := buildHighArazzo(t, yml)
	c := h.Workflows[0].Steps[0].SuccessCriteria[0]
	assert.Equal(t, "^2[0-9]{2}$", c.Condition)
	assert.Equal(t, "$statusCode", c.Context)
	assert.Equal(t, "regex", c.Type)
	assert.Nil(t, c.ExpressionType)
	assert.Equal(t, "regex", c.GetEffectiveType())
}

func TestNewCriterion_MappingCriterionExpressionType(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	criteria := h.Workflows[0].Steps[0].SuccessCriteria
	c := criteria[1]

	assert.Equal(t, "$response.body#/id != null", c.Condition)
	assert.Equal(t, "$response.body", c.Context)
	assert.Empty(t, c.Type)
	assert.NotNil(t, c.ExpressionType)
	assert.Equal(t, "jsonpath", c.ExpressionType.Type)
	assert.Equal(t, "draft-goessner-dispatch-jsonpath-00", c.ExpressionType.Version)
	assert.Equal(t, "jsonpath", c.GetEffectiveType())
}

func TestCriterion_GetEffectiveType_Default(t *testing.T) {
	// When neither Type nor ExpressionType is set, default to "simple"
	c := &Criterion{}
	assert.Equal(t, "simple", c.GetEffectiveType())
}

func TestCriterion_GoLow(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	assert.NotNil(t, h.Workflows[0].Steps[0].SuccessCriteria[0].GoLow())
}

func TestCriterion_GoLowUntyped(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	assert.NotNil(t, h.Workflows[0].Steps[0].SuccessCriteria[0].GoLowUntyped())
}

func TestCriterion_Render_ScalarType(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	rendered, err := h.Workflows[0].Steps[0].SuccessCriteria[0].Render()
	require.NoError(t, err)
	s := string(rendered)
	assert.Contains(t, s, "condition:")
	assert.Contains(t, s, "type: simple")
}

func TestCriterion_Render_MappingType(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	rendered, err := h.Workflows[0].Steps[0].SuccessCriteria[1].Render()
	require.NoError(t, err)
	s := string(rendered)
	assert.Contains(t, s, "condition:")
	assert.Contains(t, s, "type:")
}

// ---------------------------------------------------------------------------
// CriterionExpressionType
// ---------------------------------------------------------------------------

func TestNewCriterionExpressionType(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	cet := h.Workflows[0].Steps[0].SuccessCriteria[1].ExpressionType
	require.NotNil(t, cet)

	assert.Equal(t, "jsonpath", cet.Type)
	assert.Equal(t, "draft-goessner-dispatch-jsonpath-00", cet.Version)
}

func TestCriterionExpressionType_GoLow(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	cet := h.Workflows[0].Steps[0].SuccessCriteria[1].ExpressionType
	assert.NotNil(t, cet.GoLow())
}

func TestCriterionExpressionType_GoLowUntyped(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	cet := h.Workflows[0].Steps[0].SuccessCriteria[1].ExpressionType
	assert.NotNil(t, cet.GoLowUntyped())
}

func TestCriterionExpressionType_Render(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	cet := h.Workflows[0].Steps[0].SuccessCriteria[1].ExpressionType
	rendered, err := cet.Render()
	require.NoError(t, err)
	s := string(rendered)
	assert.Contains(t, s, "type: jsonpath")
	assert.Contains(t, s, "version:")
}

// ---------------------------------------------------------------------------
// SuccessAction
// ---------------------------------------------------------------------------

func TestNewSuccessAction_AllFields(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)

	// Step-level onSuccess
	sa := h.Workflows[0].Steps[0].OnSuccess[0]
	assert.Equal(t, "logSuccess", sa.Name)
	assert.Equal(t, "end", sa.Type)
	assert.Empty(t, sa.WorkflowId)
	assert.Empty(t, sa.StepId)

	// Workflow-level successActions
	wsa := h.Workflows[0].SuccessActions[0]
	assert.Equal(t, "notifySuccess", wsa.Name)
	assert.Equal(t, "goto", wsa.Type)
	assert.Equal(t, "addPet", wsa.StepId)
}

func TestSuccessAction_IsReusable_False(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	assert.False(t, h.Workflows[0].Steps[0].OnSuccess[0].IsReusable())
}

func TestSuccessAction_IsReusable_True(t *testing.T) {
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
          - reference: $components.successActions.logAndEnd
components:
  successActions:
    logAndEnd:
      name: logAndEnd
      type: end
`
	h := buildHighArazzo(t, yml)
	sa := h.Workflows[0].Steps[0].OnSuccess[0]
	assert.True(t, sa.IsReusable())
	assert.Equal(t, "$components.successActions.logAndEnd", sa.Reference)
}

func TestSuccessAction_GoLow(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	assert.NotNil(t, h.Workflows[0].Steps[0].OnSuccess[0].GoLow())
}

func TestSuccessAction_GoLowUntyped(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	assert.NotNil(t, h.Workflows[0].Steps[0].OnSuccess[0].GoLowUntyped())
}

func TestSuccessAction_Render(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	rendered, err := h.Workflows[0].Steps[0].OnSuccess[0].Render()
	require.NoError(t, err)
	s := string(rendered)
	assert.Contains(t, s, "name: logSuccess")
	assert.Contains(t, s, "type: end")
}

func TestSuccessAction_Render_Reusable(t *testing.T) {
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
          - reference: $components.successActions.logAndEnd
components:
  successActions:
    logAndEnd:
      name: logAndEnd
      type: end
`
	h := buildHighArazzo(t, yml)
	rendered, err := h.Workflows[0].Steps[0].OnSuccess[0].Render()
	require.NoError(t, err)
	s := string(rendered)
	assert.Contains(t, s, "reference:")
	// Reusable rendering only includes reference
	assert.NotContains(t, s, "name:")
}

// ---------------------------------------------------------------------------
// FailureAction
// ---------------------------------------------------------------------------

func TestNewFailureAction_AllFields(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)

	// Step-level onFailure
	fa := h.Workflows[0].Steps[0].OnFailure[0]
	assert.Equal(t, "retryAdd", fa.Name)
	assert.Equal(t, "retry", fa.Type)
	require.NotNil(t, fa.RetryAfter)
	assert.Equal(t, 1.5, *fa.RetryAfter)
	require.NotNil(t, fa.RetryLimit)
	assert.Equal(t, int64(3), *fa.RetryLimit)
	assert.Empty(t, fa.WorkflowId)
	assert.Empty(t, fa.StepId)
}

func TestFailureAction_IsReusable_False(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	assert.False(t, h.Workflows[0].Steps[0].OnFailure[0].IsReusable())
}

func TestFailureAction_IsReusable_True(t *testing.T) {
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
          - reference: $components.failureActions.retryDefault
components:
  failureActions:
    retryDefault:
      name: retryDefault
      type: retry
      retryAfter: 2.0
      retryLimit: 5
`
	h := buildHighArazzo(t, yml)
	fa := h.Workflows[0].Steps[0].OnFailure[0]
	assert.True(t, fa.IsReusable())
	assert.Equal(t, "$components.failureActions.retryDefault", fa.Reference)
}

func TestFailureAction_GoLow(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	assert.NotNil(t, h.Workflows[0].Steps[0].OnFailure[0].GoLow())
}

func TestFailureAction_GoLowUntyped(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	assert.NotNil(t, h.Workflows[0].Steps[0].OnFailure[0].GoLowUntyped())
}

func TestFailureAction_Render(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	rendered, err := h.Workflows[0].Steps[0].OnFailure[0].Render()
	require.NoError(t, err)
	s := string(rendered)
	assert.Contains(t, s, "name: retryAdd")
	assert.Contains(t, s, "type: retry")
	assert.Contains(t, s, "retryAfter:")
	assert.Contains(t, s, "retryLimit:")
}

// ---------------------------------------------------------------------------
// RequestBody
// ---------------------------------------------------------------------------

func TestNewRequestBody_AllFields(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	rb := h.Workflows[0].Steps[0].RequestBody
	require.NotNil(t, rb)

	assert.Equal(t, "application/json", rb.ContentType)
	assert.NotNil(t, rb.Payload)
	assert.Len(t, rb.Replacements, 1)
}

func TestRequestBody_GoLow(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	assert.NotNil(t, h.Workflows[0].Steps[0].RequestBody.GoLow())
}

func TestRequestBody_GoLowUntyped(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	assert.NotNil(t, h.Workflows[0].Steps[0].RequestBody.GoLowUntyped())
}

func TestRequestBody_Render(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	rendered, err := h.Workflows[0].Steps[0].RequestBody.Render()
	require.NoError(t, err)
	s := string(rendered)
	assert.Contains(t, s, "contentType: application/json")
	assert.Contains(t, s, "payload:")
	assert.Contains(t, s, "replacements:")
}

// ---------------------------------------------------------------------------
// PayloadReplacement
// ---------------------------------------------------------------------------

func TestNewPayloadReplacement_AllFields(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	rep := h.Workflows[0].Steps[0].RequestBody.Replacements[0]

	assert.Equal(t, "/name", rep.Target)
	assert.NotNil(t, rep.Value)
	assert.Equal(t, "replaced-name", rep.Value.Value)
}

func TestPayloadReplacement_GoLow(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	assert.NotNil(t, h.Workflows[0].Steps[0].RequestBody.Replacements[0].GoLow())
}

func TestPayloadReplacement_GoLowUntyped(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	assert.NotNil(t, h.Workflows[0].Steps[0].RequestBody.Replacements[0].GoLowUntyped())
}

func TestPayloadReplacement_Render(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	rendered, err := h.Workflows[0].Steps[0].RequestBody.Replacements[0].Render()
	require.NoError(t, err)
	s := string(rendered)
	assert.Contains(t, s, "target: /name")
	assert.Contains(t, s, "value:")
}

// ---------------------------------------------------------------------------
// Components
// ---------------------------------------------------------------------------

func TestNewComponents_AllMaps(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	comp := h.Components
	require.NotNil(t, comp)

	// Inputs
	require.NotNil(t, comp.Inputs)
	assert.Equal(t, 1, comp.Inputs.Len())
	_, ok := comp.Inputs.Get("petInput")
	assert.True(t, ok)

	// Parameters
	require.NotNil(t, comp.Parameters)
	assert.Equal(t, 1, comp.Parameters.Len())
	p, ok := comp.Parameters.Get("apiKeyParam")
	assert.True(t, ok)
	assert.Equal(t, "api_key", p.Name)
	assert.Equal(t, "header", p.In)

	// SuccessActions
	require.NotNil(t, comp.SuccessActions)
	assert.Equal(t, 1, comp.SuccessActions.Len())
	sa, ok := comp.SuccessActions.Get("logAndEnd")
	assert.True(t, ok)
	assert.Equal(t, "logAndEnd", sa.Name)
	assert.Equal(t, "end", sa.Type)

	// FailureActions
	require.NotNil(t, comp.FailureActions)
	assert.Equal(t, 1, comp.FailureActions.Len())
	fa, ok := comp.FailureActions.Get("retryDefault")
	assert.True(t, ok)
	assert.Equal(t, "retryDefault", fa.Name)
	assert.Equal(t, "retry", fa.Type)
	require.NotNil(t, fa.RetryAfter)
	assert.Equal(t, 2.0, *fa.RetryAfter)
	require.NotNil(t, fa.RetryLimit)
	assert.Equal(t, int64(5), *fa.RetryLimit)
}

func TestComponents_GoLow(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	assert.NotNil(t, h.Components.GoLow())
}

func TestComponents_GoLowUntyped(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	assert.NotNil(t, h.Components.GoLowUntyped())
}

func TestComponents_Render(t *testing.T) {
	h := buildHighArazzo(t, fullArazzoYAML)
	rendered, err := h.Components.Render()
	require.NoError(t, err)
	s := string(rendered)
	assert.Contains(t, s, "inputs:")
	assert.Contains(t, s, "parameters:")
	assert.Contains(t, s, "successActions:")
	assert.Contains(t, s, "failureActions:")
}

// ---------------------------------------------------------------------------
// Extensions
// ---------------------------------------------------------------------------

func TestArazzo_Extensions(t *testing.T) {
	yml := `arazzo: 1.0.1
info:
  title: Test
  version: 0.1.0
  x-custom-info: hello
sourceDescriptions:
  - name: api
    url: https://example.com
    x-vendor: acme
workflows:
  - workflowId: wf1
    steps:
      - stepId: s1
        operationId: op1
x-top-level: true
`
	h := buildHighArazzo(t, yml)

	// Top-level extension
	require.NotNil(t, h.Extensions)
	val, ok := h.Extensions.Get("x-top-level")
	assert.True(t, ok)
	assert.Equal(t, "true", val.Value)

	// Info extension
	require.NotNil(t, h.Info.Extensions)
	infoExt, ok := h.Info.Extensions.Get("x-custom-info")
	assert.True(t, ok)
	assert.Equal(t, "hello", infoExt.Value)

	// SourceDescription extension
	require.NotNil(t, h.SourceDescriptions[0].Extensions)
	sdExt, ok := h.SourceDescriptions[0].Extensions.Get("x-vendor")
	assert.True(t, ok)
	assert.Equal(t, "acme", sdExt.Value)
}

// ---------------------------------------------------------------------------
// Step with WorkflowId (instead of operationId/operationPath)
// ---------------------------------------------------------------------------

func TestStep_WithWorkflowId(t *testing.T) {
	yml := `arazzo: 1.0.1
info:
  title: Test
  version: 0.1.0
sourceDescriptions:
  - name: api
    url: https://example.com
workflows:
  - workflowId: parent
    steps:
      - stepId: callChild
        workflowId: $sourceDescriptions.api.childWorkflow
`
	h := buildHighArazzo(t, yml)
	step := h.Workflows[0].Steps[0]
	assert.Equal(t, "callChild", step.StepId)
	assert.Equal(t, "$sourceDescriptions.api.childWorkflow", step.WorkflowId)
	assert.Empty(t, step.OperationId)
	assert.Empty(t, step.OperationPath)
}

// ---------------------------------------------------------------------------
// SuccessAction with Criteria
// ---------------------------------------------------------------------------

func TestSuccessAction_WithCriteria(t *testing.T) {
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
          - name: conditionalEnd
            type: end
            criteria:
              - condition: $statusCode == 200
`
	h := buildHighArazzo(t, yml)
	sa := h.Workflows[0].Steps[0].OnSuccess[0]
	assert.Equal(t, "conditionalEnd", sa.Name)
	require.Len(t, sa.Criteria, 1)
	assert.Equal(t, "$statusCode == 200", sa.Criteria[0].Condition)
}

// ---------------------------------------------------------------------------
// FailureAction with Criteria
// ---------------------------------------------------------------------------

func TestFailureAction_WithCriteria(t *testing.T) {
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
          - name: conditionalRetry
            type: retry
            retryAfter: 0.5
            retryLimit: 2
            criteria:
              - condition: $statusCode == 429
`
	h := buildHighArazzo(t, yml)
	fa := h.Workflows[0].Steps[0].OnFailure[0]
	assert.Equal(t, "conditionalRetry", fa.Name)
	require.NotNil(t, fa.RetryAfter)
	assert.Equal(t, 0.5, *fa.RetryAfter)
	require.NotNil(t, fa.RetryLimit)
	assert.Equal(t, int64(2), *fa.RetryLimit)
	require.Len(t, fa.Criteria, 1)
	assert.Equal(t, "$statusCode == 429", fa.Criteria[0].Condition)
}
