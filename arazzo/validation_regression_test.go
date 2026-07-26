// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"context"
	"testing"

	libopenapi "github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/arazzo/expression"
	high "github.com/pb33f/libopenapi/datamodel/high/arazzo"
	lowmodel "github.com/pb33f/libopenapi/datamodel/low"
	low "github.com/pb33f/libopenapi/datamodel/low/arazzo"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestValidate_ExpressionTypeDiagnosticsUseNarrowValueNode(t *testing.T) {
	documentYAML := `arazzo: 1.0.1
info:
  title: diagnostic
  version: 1.0.0
sourceDescriptions:
  - name: source
    url: https://example.com/openapi.yaml
workflows:
  - workflowId: workflow
    steps:
      - stepId: step
        operationId: operation
        successCriteria:
          - condition: $.id
            context: $response.body
            type:
              type: jsonpath
              version: unsupported
`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(documentYAML), &root))
	var lowDocument low.Arazzo
	require.NoError(t, lowmodel.BuildModel(root.Content[0], &lowDocument))
	require.NoError(t, lowDocument.Build(context.Background(), nil, root.Content[0], nil))
	document := high.NewArazzo(&lowDocument)

	expressionType := document.Workflows[0].Steps[0].SuccessCriteria[0].ExpressionType.GoLow()
	require.NotNil(t, expressionType.Version.ValueNode)

	result := Validate(document)
	require.NotNil(t, result)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "workflows[0].steps[0].successCriteria[0].type.version", result.Errors[0].Path)
	assert.Equal(t, expressionType.Version.ValueNode.Line, result.Errors[0].Line)
	assert.Equal(t, expressionType.Version.ValueNode.Column, result.Errors[0].Column)
	assert.Greater(t, result.Errors[0].Line, 0)
}

func TestValidate_ParameterContexts(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*high.Arazzo)
		errorIs   error
	}{
		{
			name: "operation requires in",
			configure: func(document *high.Arazzo) {
				document.Workflows[0].Steps[0].Parameters = []*high.Parameter{{
					Name:  "id",
					Value: makeValueNode("1"),
				}}
			},
			errorIs: ErrMissingParameterIn,
		},
		{
			name: "workflow forbids in",
			configure: func(document *high.Arazzo) {
				document.Workflows[0].Parameters = []*high.Parameter{{
					Name:  "id",
					In:    "query",
					Value: makeValueNode("1"),
				}}
			},
			errorIs: ErrParameterInNotAllowed,
		},
		{
			name: "workflow-target step forbids in",
			configure: func(document *high.Arazzo) {
				document.Workflows = append(document.Workflows, &high.Workflow{
					WorkflowId: "child",
					Steps:      []*high.Step{{StepId: "child-step", OperationId: "get"}},
				})
				step := document.Workflows[0].Steps[0]
				step.OperationId = ""
				step.WorkflowId = "child"
				step.Parameters = []*high.Parameter{{
					Name:  "id",
					In:    "query",
					Value: makeValueNode("1"),
				}}
			},
			errorIs: ErrParameterInNotAllowed,
		},
		{
			name: "success action forbids in",
			configure: func(document *high.Arazzo) {
				document.Workflows[0].SuccessActions = []*high.SuccessAction{{
					Name: "done",
					Type: "end",
					Parameters: []*high.Parameter{{
						Name:  "id",
						In:    "query",
						Value: makeValueNode("1"),
					}},
				}}
			},
			errorIs: ErrParameterInNotAllowed,
		},
		{
			name: "failure action forbids in",
			configure: func(document *high.Arazzo) {
				document.Workflows[0].FailureActions = []*high.FailureAction{{
					Name: "done",
					Type: "end",
					Parameters: []*high.Parameter{{
						Name:  "id",
						In:    "query",
						Value: makeValueNode("1"),
					}},
				}}
			},
			errorIs: ErrParameterInNotAllowed,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document := validMinimalDoc()
			test.configure(document)
			result := Validate(document)
			require.NotNil(t, result)
			require.True(t, result.HasErrors())
			assert.ErrorIs(t, result, test.errorIs)
		})
	}
}

func TestValidate_ValidWorkflowAndActionParametersWithoutIn(t *testing.T) {
	document := validMinimalDoc()
	parameter := func(name string) *high.Parameter {
		return &high.Parameter{Name: name, Value: makeValueNode("1")}
	}
	document.Workflows[0].Parameters = []*high.Parameter{parameter("workflow")}
	document.Workflows[0].SuccessActions = []*high.SuccessAction{{
		Name:       "success",
		Type:       "end",
		Parameters: []*high.Parameter{parameter("success")},
	}}
	document.Workflows[0].FailureActions = []*high.FailureAction{{
		Name:       "failure",
		Type:       "end",
		Parameters: []*high.Parameter{parameter("failure")},
	}}
	assert.Nil(t, Validate(document))
}

func TestValidate_ParameterDuplicateIdentityUsesContext(t *testing.T) {
	document := validMinimalDoc()
	document.Workflows[0].Steps[0].Parameters = []*high.Parameter{
		{Name: "id", In: "query", Value: makeValueNode("1")},
		{Name: "id", In: "header", Value: makeValueNode("2")},
	}
	assert.Nil(t, Validate(document))

	document.Workflows[0].Steps[0].Parameters = append(
		document.Workflows[0].Steps[0].Parameters,
		&high.Parameter{Name: "id", In: "query", Value: makeValueNode("3")},
	)
	result := Validate(document)
	require.NotNil(t, result)
	assert.Contains(t, result.Error(), `duplicate parameter (name="id", in="query")`)

	document = validMinimalDoc()
	document.Workflows[0].Parameters = []*high.Parameter{
		{Name: "id", Value: makeValueNode("1")},
		{Name: "id", Value: makeValueNode("2")},
	}
	result = Validate(document)
	require.NotNil(t, result)
	assert.Contains(t, result.Error(), `duplicate parameter name "id"`)
}

func TestValidate_ReusableParameterFollowsUsageContext(t *testing.T) {
	componentParameters := orderedmap.New[string, *high.Parameter]()
	componentParameters.Set("operation", &high.Parameter{
		Name:  "id",
		In:    "query",
		Value: makeValueNode("1"),
	})
	componentParameters.Set("workflow", &high.Parameter{
		Name:  "id",
		Value: makeValueNode("1"),
	})
	reusable := func(name string) *high.Parameter {
		return &high.Parameter{Reference: "$components.parameters." + name}
	}

	document := validMinimalDoc()
	document.Components = &high.Components{Parameters: componentParameters}
	document.Workflows[0].Steps[0].Parameters = []*high.Parameter{reusable("operation")}
	document.Workflows[0].Parameters = []*high.Parameter{reusable("workflow")}
	assert.Nil(t, Validate(document))

	document.Workflows[0].Steps[0].Parameters = []*high.Parameter{reusable("workflow")}
	document.Workflows[0].Parameters = []*high.Parameter{reusable("operation")}
	result := Validate(document)
	require.NotNil(t, result)
	assert.ErrorIs(t, result, ErrMissingParameterIn)
	assert.ErrorIs(t, result, ErrParameterInNotAllowed)
}

func TestValidate_ExpressionTypeOfficialPairs(t *testing.T) {
	valid := []high.ExpressionType{
		{Type: "jsonpath", Version: "rfc9535"},
		{Type: "jsonpath", Version: "draft-goessner-dispatch-jsonpath-00"},
		{Type: "xpath", Version: "xpath-31"},
		{Type: "jsonpointer", Version: "rfc6901"},
	}
	for _, expressionType := range valid {
		document := validMinimalDoc()
		value := expressionType
		document.Workflows[0].Steps[0].SuccessCriteria = []*high.Criterion{{
			Context:        "$response.body",
			Condition:      "$.id",
			ExpressionType: &value,
		}}
		assert.Nil(t, Validate(document), "%s/%s", value.Type, value.Version)
	}

	invalid := []high.ExpressionType{
		{Type: "jsonpointer", Version: "wrong"},
		{Type: "unknown", Version: "1"},
	}
	for _, expressionType := range invalid {
		document := validMinimalDoc()
		value := expressionType
		document.Workflows[0].Steps[0].SuccessCriteria = []*high.Criterion{{
			Context:        "$response.body",
			Condition:      "$.id",
			ExpressionType: &value,
		}}
		result := Validate(document)
		require.NotNil(t, result)
		assert.True(t, result.HasErrors())
	}
}

func TestEvaluateJSONPathCriterion_DialectDispatch(t *testing.T) {
	body := &yaml.Node{}
	require.NoError(t, body.Encode(map[string]any{"paths": map[string]any{"get": true}}))
	context := &expression.Context{ResponseBody: body}

	rfc := &high.Criterion{
		Context:   "$response.body",
		Condition: `$.paths[?(@property == 'get')]`,
		ExpressionType: &high.ExpressionType{
			Type:    "jsonpath",
			Version: "rfc9535",
		},
	}
	_, err := EvaluateCriterion(rfc, context)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid jsonpath")

	legacy := &high.Criterion{
		Context:   "$response.body",
		Condition: "$.paths",
		ExpressionType: &high.ExpressionType{
			Type:    "jsonpath",
			Version: "draft-goessner-dispatch-jsonpath-00",
		},
	}
	_, err = EvaluateCriterion(legacy, context)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnsupportedExpressionDialect)
}

func TestCompileCriterionJSONPath_DialectIsPartOfCacheKey(t *testing.T) {
	caches := newCriterionCaches()
	rfcPath, err := compileCriterionJSONPath("$.paths", criterionJSONPathRFC9535, caches)
	require.NoError(t, err)
	require.NotNil(t, rfcPath)

	legacyPath, err := compileCriterionJSONPath("$.paths", criterionJSONPathLegacy, caches)
	assert.Nil(t, legacyPath)
	assert.ErrorIs(t, err, ErrUnsupportedExpressionDialect)
	assert.Len(t, caches.jsonPath, 2)

	_, err = compileCriterionJSONPath("$.paths", criterionJSONPathDialect("unknown"), caches)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown JSONPath dialect")
	assert.Len(t, caches.jsonPath, 3)
}

// Arazzo 1.0.1 section 4.7 defines a general "$components." name production that Arazzo
// 1.1.0 removed. arazzo.Validate is scoped to 1.0 documents, so a criterion context using
// that production must not be reported as an invalid expression.
func TestValidate_Arazzo10AcceptsGeneralComponentsReference(t *testing.T) {
	spec := []byte(`arazzo: 1.0.1
info:
  title: components reference
  version: 1.0.0
sourceDescriptions:
  - name: api
    url: ./api.yaml
    type: openapi
components:
  inputs:
    someInput:
      type: string
workflows:
  - workflowId: wf
    steps:
      - stepId: s1
        operationId: op
        successCriteria:
          - condition: $.id
            context: $components.inputs.someInput
            type: jsonpath
`)

	doc, err := libopenapi.NewArazzoDocument(spec)
	require.NoError(t, err)

	result := Validate(doc)
	if result != nil {
		for _, e := range result.Errors {
			assert.NotContains(t, e.Error(), "expression",
				"a legal 1.0 components reference must not be an expression error")
		}
	}
}
