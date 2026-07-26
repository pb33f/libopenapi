// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package libopenapi

import (
	"bytes"
	gocontext "context"
	"encoding/json"
	"log/slog"
	"os"
	"reflect"
	"sync"
	"testing"
	"unsafe"

	"github.com/pb33f/libopenapi/datamodel/low"
	lowArazzo "github.com/pb33f/libopenapi/datamodel/low/arazzo"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

type arazzoContextKey string

type officialArazzoSchemaMetadata struct {
	Specification string `json:"specification"`
	Schema        string `json:"schema"`
	Iteration     string `json:"iteration"`
	SHA256        string `json:"sha256"`
	Bytes         int    `json:"bytes"`
}

//go:linkname arazzoLowBuildModelFieldCache github.com/pb33f/libopenapi/datamodel/low.buildModelFieldCache
var arazzoLowBuildModelFieldCache sync.Map

// TestArazzoOfficialSchemaMetadataIsPinned records which official schema iterations
// this package was built against. It pins the identifiers only — no schema bytes are
// stored here and nothing is hashed, so this asserts a documented claim rather than a
// verified contract. The authoritative check belongs in libopenapi-validator, which
// embeds and compiles these schemas; the checksums below exist so that a mismatch
// there can be traced back to the iteration this model targeted.
func TestArazzoOfficialSchemaMetadataIsPinned(t *testing.T) {
	metadataBytes, err := os.ReadFile("arazzo/testdata/official-schema-metadata.json")
	require.NoError(t, err)
	var metadata map[string]officialArazzoSchemaMetadata
	require.NoError(t, json.Unmarshal(metadataBytes, &metadata))

	assert.Equal(t, officialArazzoSchemaMetadata{
		Specification: "https://spec.openapis.org/arazzo/v1.0.1.html",
		Schema:        "https://spec.openapis.org/arazzo/1.0/schema/2025-10-15",
		Iteration:     "2025-10-15",
		SHA256:        "b8715bd824fffcb2accf5077977d37c9e7a15be60d785e7a3a51cf600fd46ad4",
		Bytes:         23972,
	}, metadata["arazzo-1.0"])
	assert.Equal(t, officialArazzoSchemaMetadata{
		Specification: "https://spec.openapis.org/arazzo/v1.1.0.html",
		Schema:        "https://spec.openapis.org/arazzo/1.1/schema/2026-04-15",
		Iteration:     "2026-04-15",
		SHA256:        "37be908409bdb2f7bffe61fa23685c7e84cbeebfafac475a1d01dbc50ff7ab9e",
		Bytes:         32347,
	}, metadata["arazzo-1.1"])
	assert.Len(t, metadata, 2)
}

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

func TestNewArazzoDocumentWithConfiguration_OriginAndContext(t *testing.T) {
	yml := []byte(`arazzo: 1.1.0
$self: ../portable/root.yaml
info:
  title: Configured
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: api.yaml
    type: openapi
workflows:
  - workflowId: configured
    steps:
      - stepId: step
        operationId: get
`)
	key := arazzoContextKey("configured")
	ctx := gocontext.WithValue(gocontext.Background(), key, "value")
	config := &ArazzoDocumentConfiguration{
		Context:            ctx,
		RetrievalURI:       "https://retrieval.example/workflows/source.yaml",
		ApplicationBaseURI: "https://application.example/default/",
	}
	document, err := NewArazzoDocumentWithConfiguration(yml, config)
	require.NoError(t, err)
	require.NotNil(t, document)
	assert.Equal(t, "value", document.GoLow().GetContext().Value(key))
	origin := document.GetDocumentOrigin()
	require.NotNil(t, origin)
	assert.Equal(t, "https://retrieval.example/portable/root.yaml", origin.ResolvedIdentity)
	assert.Equal(t, "https://retrieval.example/workflows/source.yaml", origin.RetrievalURI)

	config.RetrievalURI = "https://mutated.example/root.yaml"
	assert.Equal(t, "https://retrieval.example/workflows/source.yaml", document.GetDocumentOrigin().RetrievalURI)
}

func TestNewArazzoDocumentWithConfiguration_InvalidOrigin(t *testing.T) {
	yml := []byte(`arazzo: 1.1.0
info:
  title: Configured
  version: 1.0.0
sourceDescriptions: []
workflows: []
`)
	document, err := NewArazzoDocumentWithConfiguration(yml, &ArazzoDocumentConfiguration{
		RetrievalURI: "https://example.com/%zz",
	})
	assert.Nil(t, document)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve arazzo document origin")
}

func TestNewArazzoDocumentWithConfiguration_MalformedAuthoredSelfStillBuilds(t *testing.T) {
	yml := []byte(`arazzo: 1.1.0
info:
  title: Malformed authored identity
  version: 1.0.0
$self: https://identity.example/%zz
sourceDescriptions: []
workflows: []
`)
	document, err := NewArazzoDocumentWithConfiguration(yml, &ArazzoDocumentConfiguration{
		RetrievalURI: "https://retrieval.example/workflows/root.yaml",
	})
	require.NoError(t, err)
	require.NotNil(t, document)
	assert.Equal(t, "https://identity.example/%zz", document.Self)
	origin := document.GetDocumentOrigin()
	require.NotNil(t, origin)
	assert.Equal(t, "https://identity.example/%zz", origin.AuthoredSelf)
	assert.Equal(t, "https://retrieval.example/workflows/root.yaml", origin.ResolvedIdentity)
}

func TestNewArazzoDocument_Arazzo11OfficialFixtureYAMLJSONEquivalent(t *testing.T) {
	yamlBytes, err := os.ReadFile("arazzo/testdata/arazzo-1.1-official.yaml")
	require.NoError(t, err)
	yamlDocument, err := NewArazzoDocument(yamlBytes)
	require.NoError(t, err)

	var value any
	require.NoError(t, yaml.Unmarshal(yamlBytes, &value))
	jsonBytes, err := json.Marshal(value)
	require.NoError(t, err)
	jsonDocument, err := NewArazzoDocument(jsonBytes)
	require.NoError(t, err)

	assert.Equal(t, yamlDocument.Arazzo, jsonDocument.Arazzo)
	assert.Equal(t, yamlDocument.Self, jsonDocument.Self)
	require.Len(t, yamlDocument.Workflows, 1)
	require.Len(t, jsonDocument.Workflows, 1)
	assert.Equal(t, yamlDocument.Workflows[0].WorkflowId, jsonDocument.Workflows[0].WorkflowId)
	require.Len(t, yamlDocument.Workflows[0].Steps, 3)
	require.Len(t, jsonDocument.Workflows[0].Steps, 3)
	for index := range yamlDocument.Workflows[0].Steps {
		yamlStep := yamlDocument.Workflows[0].Steps[index]
		jsonStep := jsonDocument.Workflows[0].Steps[index]
		assert.Equal(t, yamlStep.StepId, jsonStep.StepId)
		assert.Equal(t, yamlStep.Action, jsonStep.Action)
		assert.Equal(t, yamlStep.DependsOn, jsonStep.DependsOn)
		assert.Equal(t, yamlStep.Timeout, jsonStep.Timeout)
	}

	rendered, err := yamlDocument.Render()
	require.NoError(t, err)
	reloaded, err := NewArazzoDocument(rendered)
	require.NoError(t, err)
	assert.Equal(t, yamlDocument.Self, reloaded.Self)
	assert.True(t, reloaded.Workflows[0].Steps[0].Outputs.First().Value().IsSelector())
}

func TestNewArazzoDocument_Arazzo10GoldenRenderCompatibility(t *testing.T) {
	fixture, err := os.ReadFile("arazzo/testdata/arazzo-1.0-render.yaml")
	require.NoError(t, err)
	document, err := NewArazzoDocument(fixture)
	require.NoError(t, err)
	rendered, err := document.Render()
	require.NoError(t, err)
	assert.Equal(t, string(fixture), string(rendered))
	assert.NotContains(t, string(rendered), "$self")
	assert.NotContains(t, string(rendered), "channelPath")
	assert.NotContains(t, string(rendered), "targetSelectorType")
}

func TestNewArazzoDocument_ConcurrentParseAndRender(t *testing.T) {
	fixture, err := os.ReadFile("arazzo/testdata/arazzo-1.1-official.yaml")
	require.NoError(t, err)
	const workers = 32
	var waitGroup sync.WaitGroup
	errorsChannel := make(chan error, workers)
	for range workers {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			document, buildErr := NewArazzoDocument(fixture)
			if buildErr != nil {
				errorsChannel <- buildErr
				return
			}
			if _, renderErr := document.Render(); renderErr != nil {
				errorsChannel <- renderErr
			}
		}()
	}
	waitGroup.Wait()
	close(errorsChannel)
	for concurrentErr := range errorsChannel {
		assert.NoError(t, concurrentErr)
	}
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

func TestNewArazzoDocument_BuildModelError(t *testing.T) {
	yml := []byte(`arazzo: 1.0.1
`)
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal(yml, &root))

	var seed lowArazzo.Arazzo
	require.NoError(t, low.BuildModel(root.Content[0], &seed))

	arazzoType := reflect.TypeOf(lowArazzo.Arazzo{})
	original, ok := arazzoLowBuildModelFieldCache.Load(arazzoType)
	require.True(t, ok)

	origType := reflect.TypeOf(original)
	elemType := origType.Elem()
	replacement := reflect.MakeSlice(origType, 1, 1)
	elem := reflect.New(elemType).Elem()
	setArazzoUnexportedField(elem.FieldByName("lookupKey"), "arazzo")
	setArazzoUnexportedField(elem.FieldByName("index"), 0)
	setArazzoUnexportedField(elem.FieldByName("kind"), reflect.Bool)
	replacement.Index(0).Set(elem)

	arazzoLowBuildModelFieldCache.Store(arazzoType, replacement.Interface())
	t.Cleanup(func() {
		arazzoLowBuildModelFieldCache.Store(arazzoType, original)
	})

	doc, err := NewArazzoDocument(yml)
	assert.Error(t, err)
	assert.Nil(t, doc)
	assert.Contains(t, err.Error(), "failed to build low-level model")
	assert.Contains(t, err.Error(), "unsupported type")
}

func TestNewArazzoDocument_BuildError(t *testing.T) {
	yml := []byte(`arazzo: 1.0.1
info:
  title: Build Error
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: https://example.com/openapi.yaml
workflows:
  - workflowId: wf1
    steps:
      - stepId: s1
        operationId: op1
components:
  failureActions:
    badRetry:
      name: retry
      type: retry
      retryAfter: nope
`)
	doc, err := NewArazzoDocument(yml)
	assert.Error(t, err)
	assert.Nil(t, doc)
	assert.Contains(t, err.Error(), "failed to build arazzo document")
	assert.Contains(t, err.Error(), "invalid retryAfter")
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

func setArazzoUnexportedField(field reflect.Value, value any) {
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(reflect.ValueOf(value))
}

// The configured logger receives the resolved origin at debug level. Origin resolution weighs
// $self, the retrieval URI and the application base against each other, so the outcome is the
// detail worth tracing when a relative source URL resolves somewhere unexpected.
func TestNewArazzoDocumentWithConfiguration_LogsResolvedOrigin(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	spec := []byte(`arazzo: 1.1.0
$self: ./workflows/main.arazzo.yaml
info:
  title: origin logging
  version: 1.0.0
workflows:
  - workflowId: wf
    steps:
      - stepId: s1
        operationId: op
`)

	doc, err := NewArazzoDocumentWithConfiguration(spec, &ArazzoDocumentConfiguration{
		RetrievalURI:       "https://specs.example.com/root.arazzo.yaml",
		ApplicationBaseURI: "https://fallback.example.com/",
		Logger:             logger,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	logged := buf.String()
	assert.Contains(t, logged, "resolved arazzo document origin")
	assert.Contains(t, logged, "authoredSelf=./workflows/main.arazzo.yaml")
	assert.Contains(t, logged, "retrievalURI=https://specs.example.com/root.arazzo.yaml")
	assert.Contains(t, logged, "applicationBaseURI=https://fallback.example.com/")
	// $self is relative, so it resolves against the retrieval URI.
	assert.Contains(t, logged, "resolvedIdentity=https://specs.example.com/workflows/main.arazzo.yaml")
	assert.Contains(t, logged, "effectiveBaseURI=https://specs.example.com/workflows/main.arazzo.yaml")
}

// A nil logger must not be called, and must not stop the document building.
func TestNewArazzoDocumentWithConfiguration_NilLoggerIsSkipped(t *testing.T) {
	spec := []byte(`arazzo: 1.1.0
info:
  title: no logger
  version: 1.0.0
workflows:
  - workflowId: wf
    steps:
      - stepId: s1
        operationId: op
`)

	doc, err := NewArazzoDocumentWithConfiguration(spec, &ArazzoDocumentConfiguration{
		RetrievalURI: "https://specs.example.com/root.arazzo.yaml",
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	assert.Equal(t, "https://specs.example.com/root.arazzo.yaml", doc.GetDocumentOrigin().ResolvedIdentity)
}
