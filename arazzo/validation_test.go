// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"errors"
	"strings"
	"testing"

	high "github.com/pb33f/libopenapi/datamodel/high/arazzo"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

// makeValueNode creates a simple scalar *yaml.Node for use in parameter values.
func makeValueNode(val string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Value: val}
}

// validMinimalDoc returns a valid minimal Arazzo document for tests to modify.
func validMinimalDoc() *high.Arazzo {
	return &high.Arazzo{
		Arazzo: "1.0.1",
		Info: &high.Info{
			Title:   "Test API Workflows",
			Version: "1.0.0",
		},
		SourceDescriptions: []*high.SourceDescription{
			{
				Name: "petStore",
				URL:  "https://petstore.swagger.io/v2/swagger.json",
				Type: "openapi",
			},
		},
		Workflows: []*high.Workflow{
			{
				WorkflowId: "createPet",
				Steps: []*high.Step{
					{
						StepId:      "addPet",
						OperationId: "addPet",
					},
				},
			},
		},
	}
}

func buildOpenAPISourceDoc(specPath string, path, method, operationID string) *v3high.Document {
	pathItem := &v3high.PathItem{}
	switch strings.ToLower(method) {
	case "get":
		pathItem.Get = &v3high.Operation{OperationId: operationID}
	case "put":
		pathItem.Put = &v3high.Operation{OperationId: operationID}
	case "post":
		pathItem.Post = &v3high.Operation{OperationId: operationID}
	case "delete":
		pathItem.Delete = &v3high.Operation{OperationId: operationID}
	case "options":
		pathItem.Options = &v3high.Operation{OperationId: operationID}
	case "head":
		pathItem.Head = &v3high.Operation{OperationId: operationID}
	case "patch":
		pathItem.Patch = &v3high.Operation{OperationId: operationID}
	case "trace":
		pathItem.Trace = &v3high.Operation{OperationId: operationID}
	case "query":
		pathItem.Query = &v3high.Operation{OperationId: operationID}
	}

	paths := &v3high.Paths{
		PathItems: orderedmap.New[string, *v3high.PathItem](),
	}
	paths.PathItems.Set(path, pathItem)

	doc := &v3high.Document{
		Paths: paths,
	}
	if specPath != "" {
		doc.Index = index.NewSpecIndexWithConfig(nil, &index.SpecIndexConfig{
			SpecAbsolutePath: specPath,
		})
	}
	return doc
}

// ---------------------------------------------------------------------------
// Rule 1: Version check
// ---------------------------------------------------------------------------

func TestValidate_Rule1_ValidVersion(t *testing.T) {
	doc := validMinimalDoc()
	doc.Arazzo = "1.0.1"
	result := Validate(doc)
	assert.Nil(t, result)
}

func TestValidate_NilDocument(t *testing.T) {
	result := Validate(nil)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "document", result.Errors[0].Path)
	assert.ErrorIs(t, result.Errors[0].Cause, ErrInvalidArazzo)
}

func TestValidate_Rule1_ValidVersion_1_0_0(t *testing.T) {
	doc := validMinimalDoc()
	doc.Arazzo = "1.0.0"
	result := Validate(doc)
	assert.Nil(t, result)
}

func TestValidate_Rule1_ValidVersion_1_0_99(t *testing.T) {
	doc := validMinimalDoc()
	doc.Arazzo = "1.0.99"
	result := Validate(doc)
	assert.Nil(t, result)
}

func TestValidate_Rule1_InvalidVersion_2_0_0(t *testing.T) {
	doc := validMinimalDoc()
	doc.Arazzo = "2.0.0"
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Error(), "unsupported arazzo version")
}

func TestValidate_Rule1_InvalidVersion_0_9(t *testing.T) {
	doc := validMinimalDoc()
	doc.Arazzo = "0.9.0"
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
}

func TestValidate_Rule1_MissingVersion(t *testing.T) {
	doc := validMinimalDoc()
	doc.Arazzo = ""
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrMissingArazzoField) {
			found = true
		}
	}
	assert.True(t, found, "expected ErrMissingArazzoField")
}

// ---------------------------------------------------------------------------
// Rule 2: Required fields
// ---------------------------------------------------------------------------

func TestValidate_Rule2_MissingInfo(t *testing.T) {
	doc := validMinimalDoc()
	doc.Info = nil
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrMissingInfo) {
			found = true
		}
	}
	assert.True(t, found, "expected ErrMissingInfo")
}

func TestValidate_Rule2_MissingInfoTitle(t *testing.T) {
	doc := validMinimalDoc()
	doc.Info.Title = ""
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Error(), "missing required 'title'")
}

func TestValidate_Rule2_MissingInfoVersion(t *testing.T) {
	doc := validMinimalDoc()
	doc.Info.Version = ""
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Error(), "missing required 'version'")
}

func TestValidate_Rule2_MissingSourceDescriptions(t *testing.T) {
	doc := validMinimalDoc()
	doc.SourceDescriptions = nil
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrMissingSourceDescriptions) {
			found = true
		}
	}
	assert.True(t, found, "expected ErrMissingSourceDescriptions")
}

func TestValidate_Rule2_EmptySourceDescriptions(t *testing.T) {
	doc := validMinimalDoc()
	doc.SourceDescriptions = []*high.SourceDescription{}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
}

func TestValidate_Rule2_MissingWorkflows(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows = nil
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrMissingWorkflows) {
			found = true
		}
	}
	assert.True(t, found, "expected ErrMissingWorkflows")
}

func TestValidate_Rule2_EmptyWorkflows(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows = []*high.Workflow{}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
}

func TestValidate_Rule2_SourceDescMissingName(t *testing.T) {
	doc := validMinimalDoc()
	doc.SourceDescriptions[0].Name = ""
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Error(), "missing required 'name'")
}

func TestValidate_Rule2_SourceDescMissingURL(t *testing.T) {
	doc := validMinimalDoc()
	doc.SourceDescriptions[0].URL = ""
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Error(), "missing required 'url'")
}

func TestValidate_Rule2_WorkflowMissingSteps(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps = nil
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrEmptySteps) {
			found = true
		}
	}
	assert.True(t, found, "expected ErrEmptySteps")
}

// ---------------------------------------------------------------------------
// Rule 3: Unique IDs
// ---------------------------------------------------------------------------

func TestValidate_Rule3_DuplicateWorkflowId(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows = append(doc.Workflows, &high.Workflow{
		WorkflowId: "createPet",
		Steps: []*high.Step{
			{StepId: "s2", OperationId: "op2"},
		},
	})
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrDuplicateWorkflowId) {
			found = true
		}
	}
	assert.True(t, found, "expected ErrDuplicateWorkflowId")
}

func TestValidate_Rule3_DuplicateStepId(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps = append(doc.Workflows[0].Steps, &high.Step{
		StepId:      "addPet",
		OperationId: "addPetAgain",
	})
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrDuplicateStepId) {
			found = true
		}
	}
	assert.True(t, found, "expected ErrDuplicateStepId")
}

func TestValidate_Rule3_DuplicateSourceDescName(t *testing.T) {
	doc := validMinimalDoc()
	doc.SourceDescriptions = append(doc.SourceDescriptions, &high.SourceDescription{
		Name: "petStore",
		URL:  "https://example.com/other.yaml",
	})
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Error(), "duplicate sourceDescription name")
}

func TestValidate_Rule3_MissingWorkflowId(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].WorkflowId = ""
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrMissingWorkflowId) {
			found = true
		}
	}
	assert.True(t, found, "expected ErrMissingWorkflowId")
}

func TestValidate_Rule3_MissingStepId(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].StepId = ""
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrMissingStepId) {
			found = true
		}
	}
	assert.True(t, found, "expected ErrMissingStepId")
}

// ---------------------------------------------------------------------------
// Rule 4: Step mutual exclusivity
// ---------------------------------------------------------------------------

func TestValidate_Rule4_StepNoAction(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].OperationId = ""
	doc.Workflows[0].Steps[0].OperationPath = ""
	doc.Workflows[0].Steps[0].WorkflowId = ""
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrStepMutualExclusion) {
			found = true
		}
	}
	assert.True(t, found, "expected ErrStepMutualExclusion")
}

func TestValidate_Rule4_StepMultipleActions(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].OperationId = "addPet"
	doc.Workflows[0].Steps[0].OperationPath = "/pets"
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrStepMutualExclusion) {
			found = true
		}
	}
	assert.True(t, found, "expected ErrStepMutualExclusion for multiple actions")
}

func TestValidate_Rule4_StepAllThree(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].OperationId = "addPet"
	doc.Workflows[0].Steps[0].OperationPath = "/pets"
	doc.Workflows[0].Steps[0].WorkflowId = "someWorkflow"
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
}

func TestValidate_Rule4_StepOnlyOperationPath(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].OperationId = ""
	doc.Workflows[0].Steps[0].OperationPath = "{$sourceDescriptions.petStore}/pets"
	result := Validate(doc)
	assert.Nil(t, result)
}

func TestValidate_Rule4_StepOnlyWorkflowId(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].OperationId = ""
	doc.Workflows[0].Steps[0].WorkflowId = "createPet"
	result := Validate(doc)
	assert.Nil(t, result)
}

func TestValidate_Rule4_StepWorkflowIdUnresolved(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].OperationId = ""
	doc.Workflows[0].Steps[0].WorkflowId = "missingWorkflow"

	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())

	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrUnresolvedWorkflowRef) {
			found = true
			break
		}
	}
	assert.True(t, found, "expected ErrUnresolvedWorkflowRef for unresolved step workflowId")
}

func TestValidate_OperationLookup_NoAttachedDocsSkipsCheck(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].OperationId = "doesNotExistAnywhere"

	result := Validate(doc)
	assert.Nil(t, result)
}

func TestValidate_OperationLookup_OperationIDFound(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].OperationId = "findMe"
	doc.AddOpenAPISourceDocument(
		buildOpenAPISourceDoc("https://example.com/other.yaml", "/other", "get", "otherOp"),
		buildOpenAPISourceDoc("https://petstore.swagger.io/v2/swagger.json", "/pets", "post", "findMe"),
	)

	result := Validate(doc)
	assert.Nil(t, result)
}

func TestValidate_OperationLookup_OperationIDMissing(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].OperationId = "missingOp"
	doc.AddOpenAPISourceDocument(
		buildOpenAPISourceDoc("https://petstore.swagger.io/v2/swagger.json", "/pets", "post", "differentOp"),
	)

	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrUnresolvedOperationRef) {
			found = true
			break
		}
	}
	assert.True(t, found, "expected ErrUnresolvedOperationRef for missing operationId")
}

func TestValidate_OperationLookup_OperationPathFoundByMappedSource(t *testing.T) {
	doc := validMinimalDoc()
	doc.SourceDescriptions = append(doc.SourceDescriptions, &high.SourceDescription{
		Name: "other",
		URL:  "https://example.com/other.yaml",
		Type: "openapi",
	})
	doc.Workflows[0].Steps[0].OperationId = ""
	doc.Workflows[0].Steps[0].OperationPath = "{$sourceDescriptions.other.url}#/paths/~1orders/get"

	// Reversed attach order verifies URL mapping takes precedence over positional fallback.
	doc.AddOpenAPISourceDocument(
		buildOpenAPISourceDoc("https://example.com/other.yaml", "/orders", "get", "listOrders"),
		buildOpenAPISourceDoc("https://petstore.swagger.io/v2/swagger.json", "/pets", "post", "addPet"),
	)

	result := Validate(doc)
	assert.Nil(t, result)
}

func TestValidate_OperationLookup_OperationPathMissing(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].OperationId = ""
	doc.Workflows[0].Steps[0].OperationPath = "{$sourceDescriptions.petStore.url}#/paths/~1pets/post"
	doc.AddOpenAPISourceDocument(
		buildOpenAPISourceDoc("https://petstore.swagger.io/v2/swagger.json", "/pets", "get", "listPets"),
	)

	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrUnresolvedOperationRef) {
			found = true
			break
		}
	}
	assert.True(t, found, "expected ErrUnresolvedOperationRef for missing operationPath target")
}

func TestValidate_OperationLookup_MissingSourceMappingIsWarning(t *testing.T) {
	doc := validMinimalDoc()
	doc.SourceDescriptions = append(doc.SourceDescriptions, &high.SourceDescription{
		Name: "other",
		URL:  "https://example.com/other.yaml",
		Type: "openapi",
	})
	doc.Workflows[0].Steps[0].OperationId = ""
	doc.Workflows[0].Steps[0].OperationPath = "{$sourceDescriptions.other.url}#/paths/~1orders/get"
	doc.AddOpenAPISourceDocument(
		buildOpenAPISourceDoc("https://petstore.swagger.io/v2/swagger.json", "/pets", "post", "addPet"),
	)

	result := Validate(doc)
	require.NotNil(t, result)
	assert.False(t, result.HasErrors())
	assert.True(t, result.HasWarnings())
	assert.Contains(t, result.Warnings[0].Message, ErrOperationSourceMapping.Error())
}

// ---------------------------------------------------------------------------
// Rule 5: Parameter in validation
// ---------------------------------------------------------------------------

func TestValidate_Rule5_ValidParameterIn(t *testing.T) {
	validIns := []string{"path", "query", "header", "cookie"}
	for _, in := range validIns {
		doc := validMinimalDoc()
		doc.Workflows[0].Steps[0].Parameters = []*high.Parameter{
			{Name: "param1", In: in, Value: makeValueNode("val")},
		}
		result := Validate(doc)
		assert.Nil(t, result, "expected no errors for in=%q", in)
	}
}

func TestValidate_Rule5_InvalidParameterIn(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].Parameters = []*high.Parameter{
		{Name: "param1", In: "body", Value: makeValueNode("val")},
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrInvalidParameterIn) {
			found = true
		}
	}
	assert.True(t, found, "expected ErrInvalidParameterIn")
}

func TestValidate_Rule5_MissingParameterName(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].Parameters = []*high.Parameter{
		{Name: "", In: "header", Value: makeValueNode("val")},
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrMissingParameterName) {
			found = true
		}
	}
	assert.True(t, found, "expected ErrMissingParameterName")
}

func TestValidate_Rule5_MissingParameterValue(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].Parameters = []*high.Parameter{
		{Name: "param1", In: "header", Value: nil},
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrMissingParameterValue) {
			found = true
		}
	}
	assert.True(t, found, "expected ErrMissingParameterValue")
}

func TestValidate_Rule5_MissingParameterIn(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].Parameters = []*high.Parameter{
		{Name: "param1", In: "", Value: makeValueNode("val")},
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrMissingParameterIn) {
			found = true
		}
	}
	assert.True(t, found, "expected ErrMissingParameterIn")
}

// ---------------------------------------------------------------------------
// Rules 6-7: Action type and target validation
// ---------------------------------------------------------------------------

func TestValidate_Rule6_MissingActionName(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].OnSuccess = []*high.SuccessAction{
		{Name: "", Type: "end"},
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrMissingActionName) {
			found = true
		}
	}
	assert.True(t, found, "expected ErrMissingActionName")
}

func TestValidate_Rule6_MissingActionType(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].OnSuccess = []*high.SuccessAction{
		{Name: "action1", Type: ""},
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrMissingActionType) {
			found = true
		}
	}
	assert.True(t, found, "expected ErrMissingActionType")
}

func TestValidate_Rule6_InvalidSuccessActionType(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].OnSuccess = []*high.SuccessAction{
		{Name: "action1", Type: "retry"},
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrInvalidSuccessType) {
			found = true
		}
	}
	assert.True(t, found, "expected ErrInvalidSuccessType for 'retry' on success action")
}

func TestValidate_Rule6_ValidSuccessTypes(t *testing.T) {
	for _, tp := range []string{"end", "goto"} {
		doc := validMinimalDoc()
		var stepId string
		if tp == "goto" {
			stepId = "addPet"
		}
		doc.Workflows[0].Steps[0].OnSuccess = []*high.SuccessAction{
			{Name: "action1", Type: tp, StepId: stepId},
		}
		result := Validate(doc)
		assert.Nil(t, result, "expected no errors for type=%q", tp)
	}
}

func TestValidate_Rule6_InvalidFailureActionType(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].OnFailure = []*high.FailureAction{
		{Name: "action1", Type: "invalid"},
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrInvalidFailureType) {
			found = true
		}
	}
	assert.True(t, found, "expected ErrInvalidFailureType")
}

func TestValidate_Rule6_ValidFailureTypes(t *testing.T) {
	for _, tp := range []string{"end", "retry", "goto"} {
		doc := validMinimalDoc()
		var stepId string
		if tp == "goto" {
			stepId = "addPet"
		}
		doc.Workflows[0].Steps[0].OnFailure = []*high.FailureAction{
			{Name: "action1", Type: tp, StepId: stepId},
		}
		result := Validate(doc)
		assert.Nil(t, result, "expected no errors for failure type=%q", tp)
	}
}

func TestValidate_Rule7_ActionMutualExclusion(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].OnSuccess = []*high.SuccessAction{
		{Name: "action1", Type: "goto", WorkflowId: "otherWf", StepId: "addPet"},
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrActionMutualExclusion) {
			found = true
		}
	}
	assert.True(t, found, "expected ErrActionMutualExclusion")
}

func TestValidate_Rule7_GotoRequiresTarget(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].OnSuccess = []*high.SuccessAction{
		{Name: "action1", Type: "goto"},
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrGotoRequiresTarget) {
			found = true
		}
	}
	assert.True(t, found, "expected ErrGotoRequiresTarget")
}

func TestValidate_Rule7_StepIdNotInWorkflow(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].OnSuccess = []*high.SuccessAction{
		{Name: "action1", Type: "goto", StepId: "nonexistent"},
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrStepIdNotInWorkflow) {
			found = true
		}
	}
	assert.True(t, found, "expected ErrStepIdNotInWorkflow")
}

func TestValidate_Rule7_GotoValidStepId(t *testing.T) {
	doc := validMinimalDoc()
	// Add a second step that the goto references
	doc.Workflows[0].Steps = append(doc.Workflows[0].Steps, &high.Step{
		StepId:      "nextStep",
		OperationId: "nextOp",
	})
	doc.Workflows[0].Steps[0].OnSuccess = []*high.SuccessAction{
		{Name: "goToNext", Type: "goto", StepId: "nextStep"},
	}
	result := Validate(doc)
	assert.Nil(t, result)
}

func TestValidate_Rule7_GotoWorkflowIdUnresolved_SuccessAction(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].OnSuccess = []*high.SuccessAction{
		{Name: "goToMissingWorkflow", Type: "goto", WorkflowId: "missingWorkflow"},
	}

	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())

	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrUnresolvedWorkflowRef) {
			found = true
			break
		}
	}
	assert.True(t, found, "expected ErrUnresolvedWorkflowRef for unresolved success action workflowId")
}

func TestValidate_Rule7_GotoWorkflowIdUnresolved_FailureAction(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].OnFailure = []*high.FailureAction{
		{Name: "goToMissingWorkflow", Type: "goto", WorkflowId: "missingWorkflow"},
	}

	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())

	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrUnresolvedWorkflowRef) {
			found = true
			break
		}
	}
	assert.True(t, found, "expected ErrUnresolvedWorkflowRef for unresolved failure action workflowId")
}

func TestValidate_Rule7_FailureActionMutualExclusion(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].OnFailure = []*high.FailureAction{
		{Name: "action1", Type: "goto", WorkflowId: "otherWf", StepId: "addPet"},
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrActionMutualExclusion) {
			found = true
		}
	}
	assert.True(t, found, "expected ErrActionMutualExclusion on failure action")
}

// ---------------------------------------------------------------------------
// Rule 8: DependsOn validation
// ---------------------------------------------------------------------------

func TestValidate_Rule8_DependsOnValid(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows = append(doc.Workflows, &high.Workflow{
		WorkflowId: "secondWf",
		DependsOn:  []string{"createPet"},
		Steps: []*high.Step{
			{StepId: "s1", OperationId: "op1"},
		},
	})
	result := Validate(doc)
	assert.Nil(t, result)
}

func TestValidate_Rule8_DependsOnUnresolved(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].DependsOn = []string{"nonexistentWorkflow"}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrUnresolvedWorkflowRef) {
			found = true
		}
	}
	assert.True(t, found, "expected ErrUnresolvedWorkflowRef")
}

// ---------------------------------------------------------------------------
// Rule 9: Circular dependency detection
// ---------------------------------------------------------------------------

func TestValidate_Rule9_CircularDependency_Simple(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows = []*high.Workflow{
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
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrCircularDependency) {
			found = true
		}
	}
	assert.True(t, found, "expected ErrCircularDependency")
}

func TestValidate_Rule9_CircularDependency_Self(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].DependsOn = []string{"createPet"}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrCircularDependency) {
			found = true
		}
	}
	assert.True(t, found, "expected ErrCircularDependency for self-reference")
}

func TestValidate_Rule9_CircularDependency_ThreeWay(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows = []*high.Workflow{
		{
			WorkflowId: "wf1",
			DependsOn:  []string{"wf3"},
			Steps:      []*high.Step{{StepId: "s1", OperationId: "op1"}},
		},
		{
			WorkflowId: "wf2",
			DependsOn:  []string{"wf1"},
			Steps:      []*high.Step{{StepId: "s2", OperationId: "op2"}},
		},
		{
			WorkflowId: "wf3",
			DependsOn:  []string{"wf2"},
			Steps:      []*high.Step{{StepId: "s3", OperationId: "op3"}},
		},
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrCircularDependency) {
			found = true
		}
	}
	assert.True(t, found, "expected ErrCircularDependency for 3-way cycle")
}

func TestValidate_Rule9_NoCycle_DAG(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows = []*high.Workflow{
		{
			WorkflowId: "wf1",
			Steps:      []*high.Step{{StepId: "s1", OperationId: "op1"}},
		},
		{
			WorkflowId: "wf2",
			DependsOn:  []string{"wf1"},
			Steps:      []*high.Step{{StepId: "s2", OperationId: "op2"}},
		},
		{
			WorkflowId: "wf3",
			DependsOn:  []string{"wf1", "wf2"},
			Steps:      []*high.Step{{StepId: "s3", OperationId: "op3"}},
		},
	}
	result := Validate(doc)
	assert.Nil(t, result)
}

// ---------------------------------------------------------------------------
// Rule 10: Component key validation
// ---------------------------------------------------------------------------

func TestValidate_Rule10_ValidComponentKeys(t *testing.T) {
	doc := validMinimalDoc()
	params := orderedmap.New[string, *high.Parameter]()
	params.Set("valid-key_1.0", &high.Parameter{Name: "p", In: "header", Value: makeValueNode("v")})
	doc.Components = &high.Components{
		Parameters: params,
	}
	result := Validate(doc)
	assert.Nil(t, result)
}

func TestValidate_Rule10_InvalidComponentKey(t *testing.T) {
	doc := validMinimalDoc()
	params := orderedmap.New[string, *high.Parameter]()
	params.Set("invalid key!", &high.Parameter{Name: "p", In: "header", Value: makeValueNode("v")})
	doc.Components = &high.Components{
		Parameters: params,
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Error(), "component key")
}

func TestValidate_Rule10_InvalidInputKey(t *testing.T) {
	doc := validMinimalDoc()
	inputs := orderedmap.New[string, *yaml.Node]()
	inputs.Set("bad key!", &yaml.Node{Kind: yaml.ScalarNode, Value: "test"})
	doc.Components = &high.Components{
		Inputs: inputs,
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Error(), "component key")
}

func TestValidate_Rule10_InvalidSuccessActionKey(t *testing.T) {
	doc := validMinimalDoc()
	actions := orderedmap.New[string, *high.SuccessAction]()
	actions.Set("bad key!", &high.SuccessAction{Name: "a", Type: "end"})
	doc.Components = &high.Components{
		SuccessActions: actions,
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Error(), "component key")
}

func TestValidate_Rule10_InvalidFailureActionKey(t *testing.T) {
	doc := validMinimalDoc()
	actions := orderedmap.New[string, *high.FailureAction]()
	actions.Set("bad key!", &high.FailureAction{Name: "a", Type: "end"})
	doc.Components = &high.Components{
		FailureActions: actions,
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Error(), "component key")
}

// ---------------------------------------------------------------------------
// Rule 13: SourceDescription name format (warning)
// ---------------------------------------------------------------------------

func TestValidate_Rule13_SourceDescNameWarning(t *testing.T) {
	doc := validMinimalDoc()
	doc.SourceDescriptions[0].Name = "has spaces in name"
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasWarnings())
	assert.Contains(t, result.Warnings[0].Message, "should match")
}

func TestValidate_Rule13_SourceDescNameValid(t *testing.T) {
	doc := validMinimalDoc()
	doc.SourceDescriptions[0].Name = "valid_Name-123"
	result := Validate(doc)
	assert.Nil(t, result)
}

// ---------------------------------------------------------------------------
// Rule 13a: SourceDescription type validation
// ---------------------------------------------------------------------------

func TestValidate_Rule13a_ValidTypes(t *testing.T) {
	for _, tp := range []string{"openapi", "arazzo", ""} {
		doc := validMinimalDoc()
		doc.SourceDescriptions[0].Type = tp
		result := Validate(doc)
		assert.Nil(t, result, "expected no errors for type=%q", tp)
	}
}

func TestValidate_Rule13a_InvalidType(t *testing.T) {
	doc := validMinimalDoc()
	doc.SourceDescriptions[0].Type = "graphql"
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Error(), "unknown sourceDescription type")
}

// ---------------------------------------------------------------------------
// Valid document: no errors
// ---------------------------------------------------------------------------

func TestValidate_ValidDocument_NoErrors(t *testing.T) {
	doc := validMinimalDoc()
	result := Validate(doc)
	assert.Nil(t, result, "expected nil result for valid document")
}

func TestValidate_ValidDocument_Complex(t *testing.T) {
	params := orderedmap.New[string, *high.Parameter]()
	params.Set("sharedParam", &high.Parameter{Name: "shared", In: "header", Value: makeValueNode("v")})

	saMap := orderedmap.New[string, *high.SuccessAction]()
	saMap.Set("logAndEnd", &high.SuccessAction{Name: "logAndEnd", Type: "end"})

	faMap := orderedmap.New[string, *high.FailureAction]()
	faMap.Set("retryDefault", &high.FailureAction{Name: "retryDefault", Type: "retry"})

	doc := &high.Arazzo{
		Arazzo: "1.0.1",
		Info: &high.Info{
			Title:   "Complex Valid",
			Version: "2.0.0",
		},
		SourceDescriptions: []*high.SourceDescription{
			{Name: "api", URL: "https://example.com/api.yaml", Type: "openapi"},
			{Name: "subWorkflows", URL: "https://example.com/sub.arazzo.yaml", Type: "arazzo"},
		},
		Workflows: []*high.Workflow{
			{
				WorkflowId: "wf1",
				Steps: []*high.Step{
					{
						StepId:      "step1",
						OperationId: "listPets",
						Parameters: []*high.Parameter{
							{Name: "limit", In: "query", Value: makeValueNode("10")},
						},
						OnSuccess: []*high.SuccessAction{
							{Reference: "$components.successActions.logAndEnd"},
						},
						OnFailure: []*high.FailureAction{
							{Reference: "$components.failureActions.retryDefault"},
						},
					},
					{
						StepId:      "step2",
						OperationId: "getPet",
						Parameters: []*high.Parameter{
							{Reference: "$components.parameters.sharedParam"},
						},
					},
				},
			},
			{
				WorkflowId: "wf2",
				DependsOn:  []string{"wf1"},
				Steps: []*high.Step{
					{StepId: "s1", OperationId: "deletePet"},
				},
			},
		},
		Components: &high.Components{
			Parameters:     params,
			SuccessActions: saMap,
			FailureActions: faMap,
		},
	}

	result := Validate(doc)
	assert.Nil(t, result, "expected nil result for valid complex document")
}

// ---------------------------------------------------------------------------
// ValidationResult and ValidationError methods
// ---------------------------------------------------------------------------

func TestValidationResult_Error_Empty(t *testing.T) {
	r := &ValidationResult{}
	assert.Equal(t, "", r.Error())
}

func TestValidationResult_HasErrors_False(t *testing.T) {
	r := &ValidationResult{}
	assert.False(t, r.HasErrors())
}

func TestValidationResult_HasWarnings_False(t *testing.T) {
	r := &ValidationResult{}
	assert.False(t, r.HasWarnings())
}

func TestValidationResult_Unwrap(t *testing.T) {
	r := &ValidationResult{
		Errors: []*ValidationError{
			{Path: "a", Cause: ErrDuplicateWorkflowId},
			{Path: "b", Cause: ErrMissingStepId},
		},
	}
	assert.True(t, errors.Is(r, ErrDuplicateWorkflowId))
	assert.True(t, errors.Is(r, ErrMissingStepId))
	assert.False(t, errors.Is(r, ErrMissingInfo))
}

func TestValidationResult_Unwrap_Empty(t *testing.T) {
	r := &ValidationResult{}
	assert.Nil(t, r.Unwrap())
}

func TestValidationError_Error_WithLineInfo(t *testing.T) {
	e := &ValidationError{
		Path:   "workflows[0].steps[1]",
		Line:   10,
		Column: 5,
		Cause:  ErrMissingStepId,
	}
	s := e.Error()
	assert.Contains(t, s, "line 10")
	assert.Contains(t, s, "col 5")
	assert.Contains(t, s, "workflows[0].steps[1]")
}

func TestValidationError_Error_WithoutLineInfo(t *testing.T) {
	e := &ValidationError{
		Path:  "info.title",
		Cause: ErrMissingInfo,
	}
	s := e.Error()
	assert.Contains(t, s, "info.title")
	assert.NotContains(t, s, "line")
}

func TestValidationError_Unwrap(t *testing.T) {
	e := &ValidationError{Cause: ErrMissingStepId}
	assert.True(t, errors.Is(e, ErrMissingStepId))
}

func TestWarning_String_WithLineInfo(t *testing.T) {
	w := &Warning{
		Path:    "sourceDescriptions[0].name",
		Line:    5,
		Column:  3,
		Message: "should match pattern",
	}
	s := w.String()
	assert.Contains(t, s, "line 5")
	assert.Contains(t, s, "col 3")
}

func TestWarning_String_WithoutLineInfo(t *testing.T) {
	w := &Warning{
		Path:    "sourceDescriptions[0].name",
		Message: "should match pattern",
	}
	s := w.String()
	assert.Contains(t, s, "sourceDescriptions[0].name")
	assert.NotContains(t, s, "line")
}

// ---------------------------------------------------------------------------
// Reusable component reference validation
// ---------------------------------------------------------------------------

func TestValidate_ReusableParam_NoComponents(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].Parameters = []*high.Parameter{
		{Reference: "$components.parameters.missing"},
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrUnresolvedComponent) {
			found = true
		}
	}
	assert.True(t, found, "expected ErrUnresolvedComponent when no components defined")
}

func TestValidate_ReusableParam_MissingName(t *testing.T) {
	params := orderedmap.New[string, *high.Parameter]()
	params.Set("existing", &high.Parameter{Name: "p", In: "header", Value: makeValueNode("v")})

	doc := validMinimalDoc()
	doc.Components = &high.Components{Parameters: params}
	doc.Workflows[0].Steps[0].Parameters = []*high.Parameter{
		{Reference: "$components.parameters.missing"},
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrUnresolvedComponent) {
			found = true
		}
	}
	assert.True(t, found, "expected ErrUnresolvedComponent for missing parameter")
}

func TestValidate_ReusableParam_Valid(t *testing.T) {
	params := orderedmap.New[string, *high.Parameter]()
	params.Set("sharedParam", &high.Parameter{Name: "p", In: "header", Value: makeValueNode("v")})

	doc := validMinimalDoc()
	doc.Components = &high.Components{Parameters: params}
	doc.Workflows[0].Steps[0].Parameters = []*high.Parameter{
		{Reference: "$components.parameters.sharedParam"},
	}
	result := Validate(doc)
	assert.Nil(t, result)
}

func TestValidate_ReusableParam_InvalidPrefix(t *testing.T) {
	params := orderedmap.New[string, *high.Parameter]()
	params.Set("p", &high.Parameter{Name: "p", In: "header", Value: makeValueNode("v")})

	doc := validMinimalDoc()
	doc.Components = &high.Components{Parameters: params}
	doc.Workflows[0].Steps[0].Parameters = []*high.Parameter{
		{Reference: "$wrongPrefix.parameters.p"},
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Error(), "must start with")
}

func TestValidate_ReusableSuccessAction_Valid(t *testing.T) {
	saMap := orderedmap.New[string, *high.SuccessAction]()
	saMap.Set("logAndEnd", &high.SuccessAction{Name: "logAndEnd", Type: "end"})

	doc := validMinimalDoc()
	doc.Components = &high.Components{SuccessActions: saMap}
	doc.Workflows[0].Steps[0].OnSuccess = []*high.SuccessAction{
		{Reference: "$components.successActions.logAndEnd"},
	}
	result := Validate(doc)
	assert.Nil(t, result)
}

func TestValidate_ReusableSuccessAction_Missing(t *testing.T) {
	saMap := orderedmap.New[string, *high.SuccessAction]()
	saMap.Set("logAndEnd", &high.SuccessAction{Name: "logAndEnd", Type: "end"})

	doc := validMinimalDoc()
	doc.Components = &high.Components{SuccessActions: saMap}
	doc.Workflows[0].Steps[0].OnSuccess = []*high.SuccessAction{
		{Reference: "$components.successActions.nonexistent"},
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
}

func TestValidate_ReusableFailureAction_Valid(t *testing.T) {
	faMap := orderedmap.New[string, *high.FailureAction]()
	faMap.Set("retryDefault", &high.FailureAction{Name: "retryDefault", Type: "retry"})

	doc := validMinimalDoc()
	doc.Components = &high.Components{FailureActions: faMap}
	doc.Workflows[0].Steps[0].OnFailure = []*high.FailureAction{
		{Reference: "$components.failureActions.retryDefault"},
	}
	result := Validate(doc)
	assert.Nil(t, result)
}

func TestValidate_ReusableFailureAction_Missing(t *testing.T) {
	faMap := orderedmap.New[string, *high.FailureAction]()
	faMap.Set("retryDefault", &high.FailureAction{Name: "retryDefault", Type: "retry"})

	doc := validMinimalDoc()
	doc.Components = &high.Components{FailureActions: faMap}
	doc.Workflows[0].Steps[0].OnFailure = []*high.FailureAction{
		{Reference: "$components.failureActions.nonexistent"},
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
}

// ---------------------------------------------------------------------------
// Workflow-level success/failure actions
// ---------------------------------------------------------------------------

func TestValidate_WorkflowLevelActions_Valid(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps = append(doc.Workflows[0].Steps, &high.Step{
		StepId:      "step2",
		OperationId: "op2",
	})
	doc.Workflows[0].SuccessActions = []*high.SuccessAction{
		{Name: "done", Type: "end"},
	}
	doc.Workflows[0].FailureActions = []*high.FailureAction{
		{Name: "retryFirst", Type: "goto", StepId: "addPet"},
	}
	result := Validate(doc)
	assert.Nil(t, result)
}

func TestValidate_WorkflowLevelActions_InvalidStepRef(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].SuccessActions = []*high.SuccessAction{
		{Name: "gotoMissing", Type: "goto", StepId: "nonexistent"},
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
}

// ---------------------------------------------------------------------------
// Multiple validation errors at once
// ---------------------------------------------------------------------------

func TestValidate_MultipleErrors(t *testing.T) {
	doc := &high.Arazzo{
		Arazzo: "2.0.0",
		Info:   nil,
	}
	result := Validate(doc)
	require.NotNil(t, result)
	// Should have errors for: version, missing info, missing sourceDescriptions, missing workflows
	assert.True(t, len(result.Errors) >= 3, "expected at least 3 errors, got %d", len(result.Errors))
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestValidate_EarlyReturn_WhenRequiredFieldsMissing(t *testing.T) {
	// When info/sourceDescriptions/workflows are missing, validation returns early
	// without trying to validate workflows (which would nil pointer)
	doc := &high.Arazzo{
		Arazzo: "1.0.1",
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
}

func TestValidate_EmptyOutputKeyIsAccepted(t *testing.T) {
	// An output with a valid key regex should pass
	doc := validMinimalDoc()
	outputs := orderedmap.New[string, string]()
	outputs.Set("valid.key-1_0", "$steps.addPet.outputs.id")
	doc.Workflows[0].Outputs = outputs
	result := Validate(doc)
	assert.Nil(t, result)
}

func TestValidate_InvalidOutputKey(t *testing.T) {
	doc := validMinimalDoc()
	outputs := orderedmap.New[string, string]()
	outputs.Set("invalid key!", "$steps.addPet.outputs.id")
	doc.Workflows[0].Outputs = outputs
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Error(), "output key")
}

func TestValidate_StepInvalidOutputKey(t *testing.T) {
	doc := validMinimalDoc()
	outputs := orderedmap.New[string, string]()
	outputs.Set("bad key!", "$response.body#/id")
	doc.Workflows[0].Steps[0].Outputs = outputs
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Error(), "output key")
}

func TestValidate_DuplicateParameterNameIn(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].Parameters = []*high.Parameter{
		{Name: "token", In: "header", Value: makeValueNode("val1")},
		{Name: "token", In: "header", Value: makeValueNode("val2")},
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Error(), "duplicate parameter")
}

func TestValidate_DuplicateActionNames_Success(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].OnSuccess = []*high.SuccessAction{
		{Name: "sameName", Type: "end"},
		{Name: "sameName", Type: "end"},
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Error(), "duplicate action name")
}

func TestValidate_DuplicateActionNames_Failure(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].OnFailure = []*high.FailureAction{
		{Name: "sameName", Type: "end"},
		{Name: "sameName", Type: "end"},
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Error(), "duplicate action name")
}
