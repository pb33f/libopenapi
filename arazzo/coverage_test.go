// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/pb33f/libopenapi/arazzo/expression"
	high "github.com/pb33f/libopenapi/datamodel/high/arazzo"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func ptrFloat64(v float64) *float64 { return &v }
func ptrInt64(v int64) *int64       { return &v }

// ---------------------------------------------------------------------------
// Mock executor for engine tests
// ---------------------------------------------------------------------------

type mockExecutor struct {
	responses map[string]*ExecutionResponse
	err       error
}

func (m *mockExecutor) Execute(_ context.Context, req *ExecutionRequest) (*ExecutionResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	if resp, ok := m.responses[req.OperationID]; ok {
		return resp, nil
	}
	return &ExecutionResponse{StatusCode: 200}, nil
}

// ===========================================================================
// criterion.go tests
// ===========================================================================

// ---------------------------------------------------------------------------
// EvaluateCriterion - all branches
// ---------------------------------------------------------------------------

func TestEvaluateCriterion_SimpleType(t *testing.T) {
	c := &high.Criterion{Condition: "$statusCode == 200"}
	ok, err := EvaluateCriterion(c, &expression.Context{StatusCode: 200})
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestEvaluateCriterion_RegexType(t *testing.T) {
	c := &high.Criterion{
		Condition: "^2\\d{2}$",
		Type:      "regex",
		Context:   "$statusCode",
	}
	ok, err := EvaluateCriterion(c, &expression.Context{StatusCode: 200})
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestEvaluateCriterion_JSONPathType(t *testing.T) {
	c := &high.Criterion{
		Condition: "$.status",
		ExpressionType: &high.CriterionExpressionType{
			Type: "jsonpath",
		},
		Context: "$statusCode",
	}
	ok, err := EvaluateCriterion(c, &expression.Context{StatusCode: 200})
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestEvaluateCriterion_XPathType(t *testing.T) {
	c := &high.Criterion{
		Condition: "//status",
		ExpressionType: &high.CriterionExpressionType{
			Type: "xpath",
		},
		Context: "$statusCode",
	}
	_, err := EvaluateCriterion(c, &expression.Context{StatusCode: 200})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "xpath")
}

func TestEvaluateCriterion_UnknownType(t *testing.T) {
	c := &high.Criterion{
		Condition: "test",
		Type:      "unknown-type",
	}
	_, err := EvaluateCriterion(c, &expression.Context{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown criterion type")
}

// ---------------------------------------------------------------------------
// evaluateSimpleCriterion - with and without context
// ---------------------------------------------------------------------------

func TestEvaluateSimpleCriterion_WithContext(t *testing.T) {
	c := &high.Criterion{
		Context:   "$statusCode",
		Condition: "200",
	}
	ok, err := EvaluateCriterion(c, &expression.Context{StatusCode: 200})
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestEvaluateSimpleCriterion_WithContext_NoMatch(t *testing.T) {
	c := &high.Criterion{
		Context:   "$statusCode",
		Condition: "404",
	}
	ok, err := EvaluateCriterion(c, &expression.Context{StatusCode: 200})
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestEvaluateSimpleCriterion_WithContext_EvalError(t *testing.T) {
	c := &high.Criterion{
		Context:   "$invalidExpr",
		Condition: "200",
	}
	_, err := EvaluateCriterion(c, &expression.Context{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to evaluate context expression")
}

func TestEvaluateSimpleCriterion_WithoutContext(t *testing.T) {
	c := &high.Criterion{
		Condition: "$statusCode == 200",
	}
	ok, err := EvaluateCriterion(c, &expression.Context{StatusCode: 200})
	require.NoError(t, err)
	assert.True(t, ok)
}

// ---------------------------------------------------------------------------
// evaluateSimpleCondition
// ---------------------------------------------------------------------------

func TestEvaluateSimpleCondition_MatchingStringValue(t *testing.T) {
	ok, err := evaluateSimpleCondition("hello", "hello")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestEvaluateSimpleCondition_NonMatchingStringValue(t *testing.T) {
	ok, err := evaluateSimpleCondition("hello", "world")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestEvaluateSimpleCondition_NumericValue(t *testing.T) {
	ok, err := evaluateSimpleCondition("200", 200)
	require.NoError(t, err)
	assert.True(t, ok)
}

// ---------------------------------------------------------------------------
// evaluateSimpleConditionString
// ---------------------------------------------------------------------------

func TestEvaluateSimpleConditionString_EmptyString(t *testing.T) {
	ok, err := evaluateSimpleConditionString("", nil, nil)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestEvaluateSimpleConditionString_WhitespaceOnly(t *testing.T) {
	ok, err := evaluateSimpleConditionString("   ", nil, nil)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestEvaluateSimpleConditionString_BooleanTrue(t *testing.T) {
	ok, err := evaluateSimpleConditionString("true", nil, nil)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestEvaluateSimpleConditionString_BooleanFalse(t *testing.T) {
	ok, err := evaluateSimpleConditionString("false", nil, nil)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestEvaluateSimpleConditionString_ExpressionWithOperator(t *testing.T) {
	ctx := &expression.Context{StatusCode: 200}
	ok, err := evaluateSimpleConditionString("$statusCode == 200", ctx, nil)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestEvaluateSimpleConditionString_ExpressionNotEqual(t *testing.T) {
	ctx := &expression.Context{StatusCode: 404}
	ok, err := evaluateSimpleConditionString("$statusCode != 200", ctx, nil)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestEvaluateSimpleConditionString_SingleExpressionBoolean(t *testing.T) {
	// A single expression that evaluates to a boolean
	ctx := &expression.Context{
		Inputs: map[string]any{"enabled": true},
	}
	ok, err := evaluateSimpleConditionString("$inputs.enabled", ctx, nil)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestEvaluateSimpleConditionString_SingleExpressionNonBoolean(t *testing.T) {
	ctx := &expression.Context{StatusCode: 200}
	_, err := evaluateSimpleConditionString("$statusCode", ctx, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "did not evaluate to a boolean")
}

func TestEvaluateSimpleConditionString_SingleExpressionError(t *testing.T) {
	ctx := &expression.Context{}
	_, err := evaluateSimpleConditionString("$invalidExpr", ctx, nil)
	require.Error(t, err)
}

func TestEvaluateSimpleConditionString_LeftOperandError(t *testing.T) {
	ctx := &expression.Context{}
	_, err := evaluateSimpleConditionString("$invalidExpr == 200", ctx, nil)
	require.Error(t, err)
}

func TestEvaluateSimpleConditionString_RightOperandError(t *testing.T) {
	ctx := &expression.Context{StatusCode: 200}
	_, err := evaluateSimpleConditionString("$statusCode == $invalidExpr", ctx, nil)
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// splitSimpleCondition - all operators
// ---------------------------------------------------------------------------

func TestSplitSimpleCondition_EqualEqual(t *testing.T) {
	l, op, r, found := splitSimpleCondition("a == b")
	assert.True(t, found)
	assert.Equal(t, "a", l)
	assert.Equal(t, "==", op)
	assert.Equal(t, "b", r)
}

func TestSplitSimpleCondition_NotEqual(t *testing.T) {
	l, op, r, found := splitSimpleCondition("a != b")
	assert.True(t, found)
	assert.Equal(t, "a", l)
	assert.Equal(t, "!=", op)
	assert.Equal(t, "b", r)
}

func TestSplitSimpleCondition_GreaterEqual(t *testing.T) {
	l, op, r, found := splitSimpleCondition("a >= b")
	assert.True(t, found)
	assert.Equal(t, "a", l)
	assert.Equal(t, ">=", op)
	assert.Equal(t, "b", r)
}

func TestSplitSimpleCondition_LessEqual(t *testing.T) {
	l, op, r, found := splitSimpleCondition("a <= b")
	assert.True(t, found)
	assert.Equal(t, "a", l)
	assert.Equal(t, "<=", op)
	assert.Equal(t, "b", r)
}

func TestSplitSimpleCondition_GreaterThan(t *testing.T) {
	l, op, r, found := splitSimpleCondition("a > b")
	assert.True(t, found)
	assert.Equal(t, "a", l)
	assert.Equal(t, ">", op)
	assert.Equal(t, "b", r)
}

func TestSplitSimpleCondition_LessThan(t *testing.T) {
	l, op, r, found := splitSimpleCondition("a < b")
	assert.True(t, found)
	assert.Equal(t, "a", l)
	assert.Equal(t, "<", op)
	assert.Equal(t, "b", r)
}

func TestSplitSimpleCondition_MissingLeftOperand(t *testing.T) {
	_, _, _, found := splitSimpleCondition("== b")
	assert.False(t, found)
}

func TestSplitSimpleCondition_MissingRightOperand(t *testing.T) {
	_, _, _, found := splitSimpleCondition("a ==")
	assert.False(t, found)
}

func TestSplitSimpleCondition_NoOperator(t *testing.T) {
	_, _, _, found := splitSimpleCondition("just a string")
	assert.False(t, found)
}

func TestSplitSimpleCondition_OperatorInsideJSONPointer(t *testing.T) {
	l, op, r, found := splitSimpleCondition("$response.body#/data/>=threshold == true")
	assert.True(t, found)
	assert.Equal(t, "$response.body#/data/>=threshold", l)
	assert.Equal(t, "==", op)
	assert.Equal(t, "true", r)
}

func TestSplitSimpleCondition_NormalExpressionWithOperator(t *testing.T) {
	l, op, r, found := splitSimpleCondition("$statusCode == 200")
	assert.True(t, found)
	assert.Equal(t, "$statusCode", l)
	assert.Equal(t, "==", op)
	assert.Equal(t, "200", r)
}

func TestSplitSimpleCondition_ExpressionWithComparison(t *testing.T) {
	l, op, r, found := splitSimpleCondition("$statusCode >= 400")
	assert.True(t, found)
	assert.Equal(t, "$statusCode", l)
	assert.Equal(t, ">=", op)
	assert.Equal(t, "400", r)
}

func TestSplitSimpleCondition_BareExpressionNoOperator(t *testing.T) {
	_, _, _, found := splitSimpleCondition("$response.body#/success")
	assert.False(t, found)
}

// ---------------------------------------------------------------------------
// evaluateSimpleOperand
// ---------------------------------------------------------------------------

func TestEvaluateSimpleOperand_EmptyString(t *testing.T) {
	val, err := evaluateSimpleOperand("", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "", val)
}

func TestEvaluateSimpleOperand_ExpressionPrefix(t *testing.T) {
	ctx := &expression.Context{StatusCode: 200}
	val, err := evaluateSimpleOperand("$statusCode", ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, 200, val)
}

func TestEvaluateSimpleOperand_DoubleQuotedString(t *testing.T) {
	val, err := evaluateSimpleOperand("\"hello\"", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "hello", val)
}

func TestEvaluateSimpleOperand_SingleQuotedString(t *testing.T) {
	val, err := evaluateSimpleOperand("'world'", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "world", val)
}

func TestEvaluateSimpleOperand_BooleanTrue(t *testing.T) {
	val, err := evaluateSimpleOperand("true", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, true, val)
}

func TestEvaluateSimpleOperand_BooleanFalse(t *testing.T) {
	val, err := evaluateSimpleOperand("false", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, false, val)
}

func TestEvaluateSimpleOperand_Integer(t *testing.T) {
	val, err := evaluateSimpleOperand("42", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(42), val)
}

func TestEvaluateSimpleOperand_NegativeInteger(t *testing.T) {
	val, err := evaluateSimpleOperand("-5", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(-5), val)
}

func TestEvaluateSimpleOperand_Float(t *testing.T) {
	val, err := evaluateSimpleOperand("3.14", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 3.14, val)
}

func TestEvaluateSimpleOperand_PlainString(t *testing.T) {
	val, err := evaluateSimpleOperand("hello", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "hello", val)
}

func TestEvaluateSimpleOperand_WhitespaceTrimmmed(t *testing.T) {
	val, err := evaluateSimpleOperand("  42  ", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(42), val)
}

// ---------------------------------------------------------------------------
// compareSimpleValues - numeric comparison
// ---------------------------------------------------------------------------

func TestCompareSimpleValues_NumericEqual(t *testing.T) {
	ok, err := compareSimpleValues(int64(200), int64(200), "==")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestCompareSimpleValues_NumericNotEqual(t *testing.T) {
	ok, err := compareSimpleValues(int64(200), int64(404), "!=")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestCompareSimpleValues_NumericGreaterThan(t *testing.T) {
	ok, err := compareSimpleValues(int64(500), int64(200), ">")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestCompareSimpleValues_NumericLessThan(t *testing.T) {
	ok, err := compareSimpleValues(int64(200), int64(500), "<")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestCompareSimpleValues_NumericGreaterEqual(t *testing.T) {
	ok, err := compareSimpleValues(int64(200), int64(200), ">=")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestCompareSimpleValues_NumericLessEqual(t *testing.T) {
	ok, err := compareSimpleValues(int64(200), int64(200), "<=")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestCompareSimpleValues_NumericGreaterEqual_Greater(t *testing.T) {
	ok, err := compareSimpleValues(int64(300), int64(200), ">=")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestCompareSimpleValues_NumericLessEqual_Less(t *testing.T) {
	ok, err := compareSimpleValues(int64(100), int64(200), "<=")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestCompareSimpleValues_FloatComparison(t *testing.T) {
	ok, err := compareSimpleValues(3.14, 3.14, "==")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestCompareSimpleValues_MixedIntFloat(t *testing.T) {
	ok, err := compareSimpleValues(int64(3), 3.0, "==")
	require.NoError(t, err)
	assert.True(t, ok)
}

// ---------------------------------------------------------------------------
// compareSimpleValues - string comparison
// ---------------------------------------------------------------------------

func TestCompareSimpleValues_StringEqual(t *testing.T) {
	ok, err := compareSimpleValues("hello", "hello", "==")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestCompareSimpleValues_StringNotEqual(t *testing.T) {
	ok, err := compareSimpleValues("hello", "world", "!=")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestCompareSimpleValues_StringGreaterThan(t *testing.T) {
	ok, err := compareSimpleValues("b", "a", ">")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestCompareSimpleValues_StringLessThan(t *testing.T) {
	ok, err := compareSimpleValues("a", "b", "<")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestCompareSimpleValues_StringGreaterEqual(t *testing.T) {
	ok, err := compareSimpleValues("b", "a", ">=")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestCompareSimpleValues_StringLessEqual(t *testing.T) {
	ok, err := compareSimpleValues("a", "b", "<=")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestCompareSimpleValues_UnsupportedOperator(t *testing.T) {
	_, err := compareSimpleValues("a", "b", "~=")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported operator")
}

// ---------------------------------------------------------------------------
// numericValue - all numeric types
// ---------------------------------------------------------------------------

func TestNumericValue_Int(t *testing.T) {
	v, ok := numericValue(int(42))
	assert.True(t, ok)
	assert.Equal(t, float64(42), v)
}

func TestNumericValue_Int8(t *testing.T) {
	v, ok := numericValue(int8(8))
	assert.True(t, ok)
	assert.Equal(t, float64(8), v)
}

func TestNumericValue_Int16(t *testing.T) {
	v, ok := numericValue(int16(16))
	assert.True(t, ok)
	assert.Equal(t, float64(16), v)
}

func TestNumericValue_Int32(t *testing.T) {
	v, ok := numericValue(int32(32))
	assert.True(t, ok)
	assert.Equal(t, float64(32), v)
}

func TestNumericValue_Int64(t *testing.T) {
	v, ok := numericValue(int64(64))
	assert.True(t, ok)
	assert.Equal(t, float64(64), v)
}

func TestNumericValue_Uint(t *testing.T) {
	v, ok := numericValue(uint(42))
	assert.True(t, ok)
	assert.Equal(t, float64(42), v)
}

func TestNumericValue_Uint8(t *testing.T) {
	v, ok := numericValue(uint8(8))
	assert.True(t, ok)
	assert.Equal(t, float64(8), v)
}

func TestNumericValue_Uint16(t *testing.T) {
	v, ok := numericValue(uint16(16))
	assert.True(t, ok)
	assert.Equal(t, float64(16), v)
}

func TestNumericValue_Uint32(t *testing.T) {
	v, ok := numericValue(uint32(32))
	assert.True(t, ok)
	assert.Equal(t, float64(32), v)
}

func TestNumericValue_Uint64(t *testing.T) {
	v, ok := numericValue(uint64(64))
	assert.True(t, ok)
	assert.Equal(t, float64(64), v)
}

func TestNumericValue_Float32(t *testing.T) {
	v, ok := numericValue(float32(3.14))
	assert.True(t, ok)
	assert.InDelta(t, float64(3.14), v, 0.001)
}

func TestNumericValue_Float64(t *testing.T) {
	v, ok := numericValue(float64(3.14))
	assert.True(t, ok)
	assert.Equal(t, 3.14, v)
}

func TestNumericValue_String_NotNumeric(t *testing.T) {
	_, ok := numericValue("not a number")
	assert.False(t, ok)
}

func TestNumericValue_Bool_NotNumeric(t *testing.T) {
	_, ok := numericValue(true)
	assert.False(t, ok)
}

// ---------------------------------------------------------------------------
// evaluateRegexCriterion
// ---------------------------------------------------------------------------

func TestEvaluateRegexCriterion_NoContext(t *testing.T) {
	c := &high.Criterion{
		Condition: "^2\\d{2}$",
		Type:      "regex",
	}
	_, err := EvaluateCriterion(c, &expression.Context{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "regex criterion requires a context expression")
}

func TestEvaluateRegexCriterion_ValidMatch(t *testing.T) {
	c := &high.Criterion{
		Condition: "^2\\d{2}$",
		Type:      "regex",
		Context:   "$statusCode",
	}
	ok, err := EvaluateCriterion(c, &expression.Context{StatusCode: 201})
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestEvaluateRegexCriterion_NoMatch(t *testing.T) {
	c := &high.Criterion{
		Condition: "^2\\d{2}$",
		Type:      "regex",
		Context:   "$statusCode",
	}
	ok, err := EvaluateCriterion(c, &expression.Context{StatusCode: 404})
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestEvaluateRegexCriterion_InvalidRegex(t *testing.T) {
	c := &high.Criterion{
		Condition: "[invalid",
		Type:      "regex",
		Context:   "$statusCode",
	}
	_, err := EvaluateCriterion(c, &expression.Context{StatusCode: 200})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid regex pattern")
}

func TestEvaluateRegexCriterion_ContextEvalError(t *testing.T) {
	c := &high.Criterion{
		Condition: ".*",
		Type:      "regex",
		Context:   "$invalidExpr",
	}
	_, err := EvaluateCriterion(c, &expression.Context{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to evaluate context expression")
}

// ---------------------------------------------------------------------------
// evaluateJSONPathCriterion
// ---------------------------------------------------------------------------

func TestEvaluateJSONPathCriterion_NoContext(t *testing.T) {
	c := &high.Criterion{
		Condition: "$.status",
		ExpressionType: &high.CriterionExpressionType{
			Type: "jsonpath",
		},
	}
	_, err := EvaluateCriterion(c, &expression.Context{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "jsonpath criterion requires a context expression")
}

func TestEvaluateJSONPathCriterion_ContextEvalError(t *testing.T) {
	c := &high.Criterion{
		Condition: "$.status",
		ExpressionType: &high.CriterionExpressionType{
			Type: "jsonpath",
		},
		Context: "$invalidExpr",
	}
	_, err := EvaluateCriterion(c, &expression.Context{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to evaluate context expression")
}

func TestEvaluateJSONPathCriterion_NotImplemented(t *testing.T) {
	c := &high.Criterion{
		Condition: "$.status",
		ExpressionType: &high.CriterionExpressionType{
			Type: "jsonpath",
		},
		Context: "$statusCode",
	}
	ok, err := EvaluateCriterion(c, &expression.Context{StatusCode: 200})
	require.NoError(t, err)
	assert.False(t, ok)
}

// ===========================================================================
// engine.go tests
// ===========================================================================

// ---------------------------------------------------------------------------
// NewEngineWithConfig
// ---------------------------------------------------------------------------

func TestNewEngineWithConfig_WithConfig(t *testing.T) {
	doc := &high.Arazzo{Workflows: []*high.Workflow{}}
	config := &EngineConfig{RetainResponseBodies: true}
	engine := NewEngineWithConfig(doc, nil, nil, config)
	require.NotNil(t, engine)
	assert.True(t, engine.config.RetainResponseBodies)
}

func TestNewEngineWithConfig_NilConfig(t *testing.T) {
	doc := &high.Arazzo{Workflows: []*high.Workflow{}}
	engine := NewEngineWithConfig(doc, nil, nil, nil)
	require.NotNil(t, engine)
	// Default config should be used
	assert.False(t, engine.config.RetainResponseBodies)
}

func TestNewEngine_WithSources(t *testing.T) {
	doc := &high.Arazzo{Workflows: []*high.Workflow{}}
	sources := []*ResolvedSource{
		{Name: "api", URL: "https://example.com/api.yaml"},
		{Name: "flows", URL: "https://example.com/flows.yaml"},
	}
	engine := NewEngine(doc, nil, sources)
	require.NotNil(t, engine)
	assert.Len(t, engine.sources, 2)
	assert.NotNil(t, engine.sources["api"])
	assert.NotNil(t, engine.sources["flows"])
}

// ---------------------------------------------------------------------------
// RunWorkflow
// ---------------------------------------------------------------------------

func TestRunWorkflow_SingleWorkflow(t *testing.T) {
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{
				WorkflowId: "wf1",
				Steps: []*high.Step{
					{StepId: "s1", OperationId: "op1"},
				},
			},
		},
	}
	executor := &mockExecutor{
		responses: map[string]*ExecutionResponse{
			"op1": {StatusCode: 200},
		},
	}
	engine := NewEngine(doc, executor, nil)
	result, err := engine.RunWorkflow(context.Background(), "wf1", nil)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "wf1", result.WorkflowId)
	require.Len(t, result.Steps, 1)
	assert.Equal(t, 200, result.Steps[0].StatusCode)
}

func TestRunWorkflow_NotFound(t *testing.T) {
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{},
	}
	engine := NewEngine(doc, nil, nil)
	_, err := engine.RunWorkflow(context.Background(), "nonexistent", nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnresolvedWorkflowRef))
}

func TestRunWorkflow_CircularDetection(t *testing.T) {
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{
				WorkflowId: "wf1",
				Steps: []*high.Step{
					{StepId: "s1", WorkflowId: "wf1"}, // self-reference via step
				},
			},
		},
	}
	engine := NewEngine(doc, nil, nil)
	result, err := engine.RunWorkflow(context.Background(), "wf1", nil)
	// The step attempts to run wf1 again, triggering circular detection
	require.NoError(t, err) // The outer run succeeds
	// But the step fails due to circular detection
	require.Len(t, result.Steps, 1)
	assert.False(t, result.Steps[0].Success)
	assert.True(t, errors.Is(result.Steps[0].Error, ErrCircularDependency))
}

func TestRunWorkflow_MaxDepth(t *testing.T) {
	// Create a chain of workflows that exceeds max depth
	workflows := make([]*high.Workflow, maxWorkflowDepth+2)
	for i := range workflows {
		wfId := fmt.Sprintf("wf%d", i)
		nextWfId := fmt.Sprintf("wf%d", i+1)
		if i == len(workflows)-1 {
			workflows[i] = &high.Workflow{
				WorkflowId: wfId,
				Steps: []*high.Step{
					{StepId: "s", OperationId: "op"},
				},
			}
		} else {
			workflows[i] = &high.Workflow{
				WorkflowId: wfId,
				Steps: []*high.Step{
					{StepId: "s", WorkflowId: nextWfId},
				},
			}
		}
	}
	doc := &high.Arazzo{Workflows: workflows}
	engine := NewEngine(doc, &mockExecutor{}, nil)
	result, err := engine.RunWorkflow(context.Background(), "wf0", nil)
	// One of the nested calls should fail due to max depth
	require.NoError(t, err)
	assert.False(t, result.Success)
}

func TestRunWorkflow_ContextCancellation(t *testing.T) {
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{
				WorkflowId: "wf1",
				Steps: []*high.Step{
					{StepId: "s1", OperationId: "op1"},
					{StepId: "s2", OperationId: "op2"},
				},
			},
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	engine := NewEngine(doc, &mockExecutor{}, nil)
	result, err := engine.RunWorkflow(ctx, "wf1", nil)
	require.NoError(t, err)
	assert.False(t, result.Success)
}

// ---------------------------------------------------------------------------
// RunAll
// ---------------------------------------------------------------------------

func TestRunAll_ContextCancellation(t *testing.T) {
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{
				WorkflowId: "wf1",
				Steps: []*high.Step{
					{StepId: "s1", OperationId: "op1"},
				},
			},
			{
				WorkflowId: "wf2",
				Steps: []*high.Step{
					{StepId: "s2", OperationId: "op2"},
				},
			},
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	engine := NewEngine(doc, &mockExecutor{}, nil)
	_, err := engine.RunAll(ctx, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestRunAll_MultipleWorkflowsWithDependencies(t *testing.T) {
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{
				WorkflowId: "wf1",
				Steps: []*high.Step{
					{StepId: "s1", OperationId: "op1"},
				},
			},
			{
				WorkflowId: "wf2",
				DependsOn:  []string{"wf1"},
				Steps: []*high.Step{
					{StepId: "s2", OperationId: "op2"},
				},
			},
			{
				WorkflowId: "wf3",
				DependsOn:  []string{"wf1", "wf2"},
				Steps: []*high.Step{
					{StepId: "s3", OperationId: "op3"},
				},
			},
		},
	}
	engine := NewEngine(doc, &mockExecutor{}, nil)
	result, err := engine.RunAll(context.Background(), nil)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Len(t, result.Workflows, 3)
}

func TestRunAll_WorkflowFailure(t *testing.T) {
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{
				WorkflowId: "wf1",
				Steps: []*high.Step{
					{StepId: "s1", OperationId: "op1"},
				},
			},
		},
	}
	executor := &mockExecutor{err: fmt.Errorf("executor failure")}
	engine := NewEngine(doc, executor, nil)
	result, err := engine.RunAll(context.Background(), nil)
	require.NoError(t, err)
	assert.False(t, result.Success)
}

func TestRunAll_WithInputs(t *testing.T) {
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{
				WorkflowId: "wf1",
				Steps: []*high.Step{
					{StepId: "s1", OperationId: "op1"},
				},
			},
		},
	}
	inputs := map[string]map[string]any{
		"wf1": {"key": "value"},
	}
	engine := NewEngine(doc, &mockExecutor{}, nil)
	result, err := engine.RunAll(context.Background(), inputs)
	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestRunAll_CircularDependencies(t *testing.T) {
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{
				WorkflowId: "wf1",
				DependsOn:  []string{"wf2"},
				Steps: []*high.Step{
					{StepId: "s1", OperationId: "op1"},
				},
			},
			{
				WorkflowId: "wf2",
				DependsOn:  []string{"wf1"},
				Steps: []*high.Step{
					{StepId: "s2", OperationId: "op2"},
				},
			},
		},
	}
	engine := NewEngine(doc, &mockExecutor{}, nil)
	_, err := engine.RunAll(context.Background(), nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCircularDependency))
}

// ---------------------------------------------------------------------------
// executeStep
// ---------------------------------------------------------------------------

func TestExecuteStep_WithWorkflowReference(t *testing.T) {
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{
				WorkflowId: "main",
				Steps: []*high.Step{
					{StepId: "callSub", WorkflowId: "sub"},
				},
			},
			{
				WorkflowId: "sub",
				Steps: []*high.Step{
					{StepId: "s1", OperationId: "op1"},
				},
			},
		},
	}
	engine := NewEngine(doc, &mockExecutor{}, nil)
	result, err := engine.RunWorkflow(context.Background(), "main", nil)
	require.NoError(t, err)
	assert.True(t, result.Success)
	require.Len(t, result.Steps, 1)
	assert.Equal(t, "callSub", result.Steps[0].StepId)
}

func TestExecuteStep_WithExecutor(t *testing.T) {
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{
				WorkflowId: "wf1",
				Steps: []*high.Step{
					{StepId: "s1", OperationId: "op1"},
				},
			},
		},
	}
	executor := &mockExecutor{
		responses: map[string]*ExecutionResponse{
			"op1": {StatusCode: 201, Headers: map[string][]string{"X-Test": {"val"}}},
		},
	}
	engine := NewEngine(doc, executor, nil)
	result, err := engine.RunWorkflow(context.Background(), "wf1", nil)
	require.NoError(t, err)
	require.Len(t, result.Steps, 1)
	assert.Equal(t, 201, result.Steps[0].StatusCode)
}

func TestExecuteStep_WithoutExecutor(t *testing.T) {
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{
				WorkflowId: "wf1",
				Steps: []*high.Step{
					{StepId: "s1", OperationId: "op1"},
				},
			},
		},
	}
	engine := NewEngine(doc, nil, nil) // nil executor
	result, err := engine.RunWorkflow(context.Background(), "wf1", nil)
	require.NoError(t, err)
	assert.False(t, result.Success)
	require.Len(t, result.Steps, 1)
	assert.Equal(t, 0, result.Steps[0].StatusCode)
	require.Error(t, result.Steps[0].Error)
	assert.ErrorIs(t, result.Steps[0].Error, ErrExecutorNotConfigured)
}

func TestExecuteStep_ExecutorError(t *testing.T) {
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{
				WorkflowId: "wf1",
				Steps: []*high.Step{
					{StepId: "s1", OperationId: "op1"},
				},
			},
		},
	}
	executor := &mockExecutor{err: fmt.Errorf("network failure")}
	engine := NewEngine(doc, executor, nil)
	result, err := engine.RunWorkflow(context.Background(), "wf1", nil)
	require.NoError(t, err)
	assert.False(t, result.Success)
	require.Len(t, result.Steps, 1)
	assert.False(t, result.Steps[0].Success)
	assert.Contains(t, result.Steps[0].Error.Error(), "network failure")
}

func TestExecuteStep_WithOperationPath(t *testing.T) {
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{
				WorkflowId: "wf1",
				Steps: []*high.Step{
					{StepId: "s1", OperationPath: "/pets"},
				},
			},
		},
	}
	executor := &mockExecutor{}
	engine := NewEngine(doc, executor, nil)
	result, err := engine.RunWorkflow(context.Background(), "wf1", nil)
	require.NoError(t, err)
	assert.True(t, result.Success)
}

// ---------------------------------------------------------------------------
// parseExpression - cache hit and miss
// ---------------------------------------------------------------------------

func TestParseExpression_CacheMiss(t *testing.T) {
	doc := &high.Arazzo{Workflows: []*high.Workflow{}}
	engine := NewEngine(doc, nil, nil)
	expr, err := engine.parseExpression("$statusCode")
	require.NoError(t, err)
	assert.Equal(t, expression.StatusCode, expr.Type)
}

func TestParseExpression_CacheHit(t *testing.T) {
	doc := &high.Arazzo{Workflows: []*high.Workflow{}}
	engine := NewEngine(doc, nil, nil)
	// First call populates cache
	expr1, err1 := engine.parseExpression("$statusCode")
	require.NoError(t, err1)
	// Second call should hit cache
	expr2, err2 := engine.parseExpression("$statusCode")
	require.NoError(t, err2)
	assert.Equal(t, expr1, expr2)
}

func TestParseExpression_InvalidExpression(t *testing.T) {
	doc := &high.Arazzo{Workflows: []*high.Workflow{}}
	engine := NewEngine(doc, nil, nil)
	_, err := engine.parseExpression("not-an-expression")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// topologicalSort
// ---------------------------------------------------------------------------

func TestTopologicalSort_NoDependencies(t *testing.T) {
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{WorkflowId: "wf1", Steps: []*high.Step{{StepId: "s1", OperationId: "op1"}}},
			{WorkflowId: "wf2", Steps: []*high.Step{{StepId: "s2", OperationId: "op2"}}},
		},
	}
	engine := NewEngine(doc, nil, nil)
	order, err := engine.topologicalSort()
	require.NoError(t, err)
	assert.Len(t, order, 2)
}

func TestTopologicalSort_WithDependencies(t *testing.T) {
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{WorkflowId: "wf1", Steps: []*high.Step{{StepId: "s1", OperationId: "op1"}}},
			{WorkflowId: "wf2", DependsOn: []string{"wf1"}, Steps: []*high.Step{{StepId: "s2", OperationId: "op2"}}},
			{WorkflowId: "wf3", DependsOn: []string{"wf2"}, Steps: []*high.Step{{StepId: "s3", OperationId: "op3"}}},
		},
	}
	engine := NewEngine(doc, nil, nil)
	order, err := engine.topologicalSort()
	require.NoError(t, err)
	require.Len(t, order, 3)
	// wf1 must come before wf2, wf2 before wf3
	wf1Idx, wf2Idx, wf3Idx := -1, -1, -1
	for i, id := range order {
		switch id {
		case "wf1":
			wf1Idx = i
		case "wf2":
			wf2Idx = i
		case "wf3":
			wf3Idx = i
		}
	}
	assert.True(t, wf1Idx < wf2Idx)
	assert.True(t, wf2Idx < wf3Idx)
}

func TestTopologicalSort_CircularDependencies(t *testing.T) {
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{WorkflowId: "wf1", DependsOn: []string{"wf2"}, Steps: []*high.Step{{StepId: "s1", OperationId: "op1"}}},
			{WorkflowId: "wf2", DependsOn: []string{"wf1"}, Steps: []*high.Step{{StepId: "s2", OperationId: "op2"}}},
		},
	}
	engine := NewEngine(doc, nil, nil)
	_, err := engine.topologicalSort()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCircularDependency))
}

func TestTopologicalSort_DiamondDependency(t *testing.T) {
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{WorkflowId: "wf1", Steps: []*high.Step{{StepId: "s1", OperationId: "op1"}}},
			{WorkflowId: "wf2", DependsOn: []string{"wf1"}, Steps: []*high.Step{{StepId: "s2", OperationId: "op2"}}},
			{WorkflowId: "wf3", DependsOn: []string{"wf1"}, Steps: []*high.Step{{StepId: "s3", OperationId: "op3"}}},
			{WorkflowId: "wf4", DependsOn: []string{"wf2", "wf3"}, Steps: []*high.Step{{StepId: "s4", OperationId: "op4"}}},
		},
	}
	engine := NewEngine(doc, nil, nil)
	order, err := engine.topologicalSort()
	require.NoError(t, err)
	assert.Len(t, order, 4)
	// wf1 must come first, wf4 must come last
	assert.Equal(t, "wf1", order[0])
	assert.Equal(t, "wf4", order[3])
}

// ---------------------------------------------------------------------------
// RunAll - dependency failure propagation
// ---------------------------------------------------------------------------

func TestRunAll_NilDocument(t *testing.T) {
	engine := NewEngine(nil, &mockExecutor{}, nil)
	result, err := engine.RunAll(context.Background(), nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Empty(t, result.Workflows)
}

func TestRunAll_DependencyFailurePropagates(t *testing.T) {
	// wf1 fails via executor error, wf2 depends on wf1 => wf2 should be skipped
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{
				WorkflowId: "wf1",
				Steps: []*high.Step{
					{StepId: "s1", OperationId: "op1"},
				},
			},
			{
				WorkflowId: "wf2",
				DependsOn:  []string{"wf1"},
				Steps: []*high.Step{
					{StepId: "s2", OperationId: "op2"},
				},
			},
		},
	}
	executor := &mockExecutor{err: fmt.Errorf("executor error")}
	engine := NewEngine(doc, executor, nil)
	result, err := engine.RunAll(context.Background(), nil)
	require.NoError(t, err)
	assert.False(t, result.Success)
	require.Len(t, result.Workflows, 2)
	// wf2 should have been skipped due to dependency failure
	assert.False(t, result.Workflows[1].Success)
	assert.Contains(t, result.Workflows[1].Error.Error(), "dependency")
}

func TestRunAll_DependencyNotExecuted(t *testing.T) {
	// If a workflow depends on a workflow that wasn't executed (not in results), it should fail
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{
				WorkflowId: "wf1",
				Steps: []*high.Step{
					{StepId: "s1", OperationId: "op1"},
				},
			},
			{
				WorkflowId: "wf2",
				DependsOn:  []string{"wf1"},
				Steps: []*high.Step{
					{StepId: "s2", OperationId: "op2"},
				},
			},
		},
	}
	// wf1 will succeed, wf2 depends on wf1 - should work normally
	engine := NewEngine(doc, &mockExecutor{}, nil)
	result, err := engine.RunAll(context.Background(), nil)
	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestRunAll_WorkflowExecError(t *testing.T) {
	// Simulate a workflow that returns an error from runWorkflow (not a step failure)
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{
				WorkflowId: "wf1",
				Steps:      nil, // empty steps should still work
			},
		},
	}
	engine := NewEngine(doc, &mockExecutor{}, nil)
	result, err := engine.RunAll(context.Background(), nil)
	require.NoError(t, err)
	// Workflow with no steps still succeeds (empty loop)
	assert.True(t, result.Success)
}

func TestRunAll_RunWorkflowReturnsError(t *testing.T) {
	// When the topological sort includes workflow IDs from DependsOn that
	// don't exist in the document, runWorkflow returns an error.
	// topologicalSort adds DependsOn IDs to inDegree even if they don't exist
	// as actual workflows. So runWorkflow("missingDep") would return
	// ErrUnresolvedWorkflowRef - triggering the execErr != nil branch.
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{
				WorkflowId: "wf1",
				DependsOn:  []string{"missingDep"},
				Steps: []*high.Step{
					{StepId: "s1", OperationId: "op1"},
				},
			},
		},
	}
	engine := NewEngine(doc, &mockExecutor{}, nil)
	result, err := engine.RunAll(context.Background(), nil)
	require.NoError(t, err)
	assert.False(t, result.Success)
	// missingDep should have been attempted via runWorkflow and failed
	require.True(t, len(result.Workflows) >= 1)
	// One workflow result should have ErrUnresolvedWorkflowRef
	foundUnresolved := false
	for _, wfr := range result.Workflows {
		if wfr.Error != nil && errors.Is(wfr.Error, ErrUnresolvedWorkflowRef) {
			foundUnresolved = true
		}
	}
	assert.True(t, foundUnresolved, "expected at least one workflow with ErrUnresolvedWorkflowRef")
}

func TestRunAll_WorkflowStepFailure_NotSuccess(t *testing.T) {
	// A workflow whose steps fail but runWorkflow returns no error - result.Success = false
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{
				WorkflowId: "wf1",
				Steps: []*high.Step{
					{StepId: "s1", OperationId: "fail-op"},
				},
			},
			{
				WorkflowId: "wf2",
				DependsOn:  []string{"wf1"},
				Steps: []*high.Step{
					{StepId: "s2", OperationId: "op2"},
				},
			},
		},
	}
	executor := &mockExecutor{err: fmt.Errorf("fail")}
	engine := NewEngine(doc, executor, nil)
	result, err := engine.RunAll(context.Background(), nil)
	require.NoError(t, err)
	assert.False(t, result.Success)
}

// ---------------------------------------------------------------------------
// dependencyExecutionError
// ---------------------------------------------------------------------------

func TestDependencyExecutionError_NoDeps(t *testing.T) {
	wf := &high.Workflow{WorkflowId: "wf1"}
	err := dependencyExecutionError(wf, map[string]*WorkflowResult{})
	assert.NoError(t, err)
}

func TestDependencyExecutionError_DepNotFound(t *testing.T) {
	wf := &high.Workflow{WorkflowId: "wf2", DependsOn: []string{"wf1"}}
	err := dependencyExecutionError(wf, map[string]*WorkflowResult{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnresolvedWorkflowRef))
}

func TestDependencyExecutionError_DepFailedWithError(t *testing.T) {
	wf := &high.Workflow{WorkflowId: "wf2", DependsOn: []string{"wf1"}}
	results := map[string]*WorkflowResult{
		"wf1": {WorkflowId: "wf1", Success: false, Error: fmt.Errorf("boom")},
	}
	err := dependencyExecutionError(wf, results)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dependency")
	assert.Contains(t, err.Error(), "boom")
}

func TestDependencyExecutionError_DepFailedWithoutError(t *testing.T) {
	wf := &high.Workflow{WorkflowId: "wf2", DependsOn: []string{"wf1"}}
	results := map[string]*WorkflowResult{
		"wf1": {WorkflowId: "wf1", Success: false, Error: nil},
	}
	err := dependencyExecutionError(wf, results)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dependency")
}

func TestDependencyExecutionError_DepSucceeded(t *testing.T) {
	wf := &high.Workflow{WorkflowId: "wf2", DependsOn: []string{"wf1"}}
	results := map[string]*WorkflowResult{
		"wf1": {WorkflowId: "wf1", Success: true},
	}
	err := dependencyExecutionError(wf, results)
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// runWorkflow - step failure with nil error wraps into "step X failed"
// ---------------------------------------------------------------------------

func TestRunWorkflow_StepFailure_NilError(t *testing.T) {
	// A step that references a sub-workflow that fails without an explicit error
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{
			{
				WorkflowId: "main",
				Steps: []*high.Step{
					{StepId: "s1", WorkflowId: "sub"},
				},
			},
			{
				WorkflowId: "sub",
				Steps: []*high.Step{
					{StepId: "s2", OperationId: "op-fail"},
				},
			},
		},
	}
	executor := &mockExecutor{err: fmt.Errorf("fail")}
	engine := NewEngine(doc, executor, nil)
	result, err := engine.RunWorkflow(context.Background(), "main", nil)
	require.NoError(t, err)
	assert.False(t, result.Success)
}

// ===========================================================================
// resolve.go tests
// ===========================================================================

// ---------------------------------------------------------------------------
// ResolveSources
// ---------------------------------------------------------------------------

func TestResolveSources_NilDoc(t *testing.T) {
	_, err := ResolveSources(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil arazzo document")
}

func TestResolveSources_NilConfig(t *testing.T) {
	doc := &high.Arazzo{
		SourceDescriptions: []*high.SourceDescription{
			{Name: "api", URL: "https://example.com/api.yaml", Type: "openapi"},
		},
	}
	// nil config should have defaults applied, but no factory => error
	_, err := ResolveSources(doc, nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrSourceDescLoadFailed))
}

func TestResolveSources_TooManySources(t *testing.T) {
	descs := make([]*high.SourceDescription, 51)
	for i := range descs {
		descs[i] = &high.SourceDescription{
			Name: fmt.Sprintf("sd%d", i),
			URL:  fmt.Sprintf("https://example.com/%d.yaml", i),
			Type: "openapi",
		}
	}
	doc := &high.Arazzo{SourceDescriptions: descs}
	config := &ResolveConfig{MaxSources: 50}
	_, err := ResolveSources(doc, config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too many source descriptions")
}

func TestResolveSources_NilSourceDescription(t *testing.T) {
	doc := &high.Arazzo{
		SourceDescriptions: []*high.SourceDescription{nil},
	}
	_, err := ResolveSources(doc, &ResolveConfig{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrSourceDescLoadFailed))
	assert.Contains(t, err.Error(), "source description is nil")
}

func TestResolveSources_FactoryError(t *testing.T) {
	doc := &high.Arazzo{
		SourceDescriptions: []*high.SourceDescription{
			{Name: "api", URL: "https://example.com/api.yaml", Type: "openapi"},
		},
	}
	config := &ResolveConfig{
		HTTPHandler: func(_ string) ([]byte, error) {
			return []byte("content"), nil
		},
		OpenAPIFactory: func(_ string, _ []byte) (any, error) {
			return nil, fmt.Errorf("parse failed")
		},
	}
	_, err := ResolveSources(doc, config)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrSourceDescLoadFailed))
	assert.Contains(t, err.Error(), "parse failed")
}

func TestResolveSources_DefaultTypeIsOpenAPI(t *testing.T) {
	doc := &high.Arazzo{
		SourceDescriptions: []*high.SourceDescription{
			{Name: "api", URL: "https://example.com/api.yaml"}, // no Type
		},
	}
	config := &ResolveConfig{
		HTTPHandler: func(_ string) ([]byte, error) {
			return []byte("content"), nil
		},
		OpenAPIFactory: func(u string, b []byte) (any, error) {
			return "doc", nil
		},
	}
	resolved, err := ResolveSources(doc, config)
	require.NoError(t, err)
	require.Len(t, resolved, 1)
	assert.Equal(t, "openapi", resolved[0].Type)
}

// ---------------------------------------------------------------------------
// parseAndResolveSourceURL
// ---------------------------------------------------------------------------

func TestParseAndResolveSourceURL_EmptyURL(t *testing.T) {
	_, err := parseAndResolveSourceURL("", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing source URL")
}

func TestParseAndResolveSourceURL_AbsoluteURL(t *testing.T) {
	u, err := parseAndResolveSourceURL("https://example.com/api.yaml", "")
	require.NoError(t, err)
	assert.Equal(t, "https", u.Scheme)
	assert.Equal(t, "example.com", u.Host)
}

func TestParseAndResolveSourceURL_RelativeWithBase(t *testing.T) {
	u, err := parseAndResolveSourceURL("api.yaml", "https://example.com/specs/")
	require.NoError(t, err)
	assert.Equal(t, "https", u.Scheme)
	assert.Contains(t, u.Path, "api.yaml")
}

func TestParseAndResolveSourceURL_RelativeWithoutBase(t *testing.T) {
	u, err := parseAndResolveSourceURL("api.yaml", "")
	require.NoError(t, err)
	assert.Equal(t, "file", u.Scheme)
	assert.Equal(t, "api.yaml", u.Path)
}

func TestParseAndResolveSourceURL_SchemelessDefaultsToFile(t *testing.T) {
	u, err := parseAndResolveSourceURL("/some/path/api.yaml", "")
	require.NoError(t, err)
	assert.Equal(t, "file", u.Scheme)
}

func TestParseAndResolveSourceURL_InvalidBaseURL(t *testing.T) {
	_, err := parseAndResolveSourceURL("api.yaml", "://invalid-base")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

// ---------------------------------------------------------------------------
// validateSourceURL
// ---------------------------------------------------------------------------

func TestValidateSourceURL_AllowedScheme(t *testing.T) {
	config := &ResolveConfig{AllowedSchemes: []string{"https"}}
	u := mustParseURL("https://example.com/api.yaml")
	err := validateSourceURL(u, config)
	assert.NoError(t, err)
}

func TestValidateSourceURL_BlockedScheme(t *testing.T) {
	config := &ResolveConfig{AllowedSchemes: []string{"https"}}
	u := mustParseURL("ftp://example.com/api.yaml")
	err := validateSourceURL(u, config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "scheme")
}

func TestValidateSourceURL_AllowedHost(t *testing.T) {
	config := &ResolveConfig{
		AllowedSchemes: []string{"https"},
		AllowedHosts:   []string{"example.com"},
	}
	u := mustParseURL("https://example.com/api.yaml")
	err := validateSourceURL(u, config)
	assert.NoError(t, err)
}

func TestValidateSourceURL_BlockedHost(t *testing.T) {
	config := &ResolveConfig{
		AllowedSchemes: []string{"https"},
		AllowedHosts:   []string{"allowed.com"},
	}
	u := mustParseURL("https://blocked.com/api.yaml")
	err := validateSourceURL(u, config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "host")
}

func TestValidateSourceURL_FileSchemeSkipsHostCheck(t *testing.T) {
	config := &ResolveConfig{
		AllowedSchemes: []string{"file"},
		AllowedHosts:   []string{"specific-host.com"},
	}
	u := mustParseURL("file:///some/path/api.yaml")
	err := validateSourceURL(u, config)
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// fetchSourceBytes
// ---------------------------------------------------------------------------

func TestFetchSourceBytes_UnsupportedScheme(t *testing.T) {
	u := mustParseURL("ftp://example.com/api.yaml")
	config := &ResolveConfig{MaxBodySize: 10 * 1024 * 1024}
	_, _, err := fetchSourceBytes(u, config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported source scheme")
}

func TestFetchSourceBytes_HTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("http-content"))
	}))
	defer server.Close()

	u := mustParseURL(server.URL + "/api.yaml")
	config := &ResolveConfig{MaxBodySize: 1024, Timeout: 5e9}
	b, resolvedURL, err := fetchSourceBytes(u, config)
	require.NoError(t, err)
	assert.Equal(t, "http-content", string(b))
	assert.Contains(t, resolvedURL, server.URL)
}

func TestFetchSourceBytes_File(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "api.yaml")
	require.NoError(t, os.WriteFile(filePath, []byte("file-content"), 0o600))

	u := &url.URL{Scheme: "file", Path: filepath.ToSlash(filePath)}
	config := &ResolveConfig{MaxBodySize: 1024}
	b, resolvedURL, err := fetchSourceBytes(u, config)
	require.NoError(t, err)
	assert.Equal(t, "file-content", string(b))
	assert.Contains(t, resolvedURL, "file://")
}

func TestFetchSourceBytes_FileError(t *testing.T) {
	u := mustParseURL("file:///nonexistent/path/file.yaml")
	config := &ResolveConfig{MaxBodySize: 1024}
	_, _, err := fetchSourceBytes(u, config)
	require.Error(t, err)
}

func TestFetchSourceBytes_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer server.Close()

	u := mustParseURL(server.URL + "/api.yaml")
	config := &ResolveConfig{MaxBodySize: 1024, Timeout: 5e9}
	_, _, err := fetchSourceBytes(u, config)
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// fetchHTTPSourceBytes
// ---------------------------------------------------------------------------

func TestFetchHTTPSourceBytes_CustomHandler(t *testing.T) {
	config := &ResolveConfig{
		MaxBodySize: 1024,
		HTTPHandler: func(url string) ([]byte, error) {
			return []byte("response body"), nil
		},
	}
	b, err := fetchHTTPSourceBytes("https://example.com/api.yaml", config)
	require.NoError(t, err)
	assert.Equal(t, "response body", string(b))
}

func TestFetchHTTPSourceBytes_CustomHandler_ExceedsMax(t *testing.T) {
	config := &ResolveConfig{
		MaxBodySize: 5, // very small
		HTTPHandler: func(url string) ([]byte, error) {
			return []byte("this is too long"), nil
		},
	}
	_, err := fetchHTTPSourceBytes("https://example.com/api.yaml", config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds max size")
}

func TestFetchHTTPSourceBytes_CustomHandler_Error(t *testing.T) {
	config := &ResolveConfig{
		MaxBodySize: 1024,
		HTTPHandler: func(url string) ([]byte, error) {
			return nil, fmt.Errorf("handler error")
		},
	}
	_, err := fetchHTTPSourceBytes("https://example.com/api.yaml", config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "handler error")
}

func TestFetchHTTPSourceBytes_RealHTTP_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("openapi: 3.1.0"))
	}))
	defer server.Close()

	config := &ResolveConfig{
		MaxBodySize: 1024,
		Timeout:     5e9, // 5 seconds
	}
	b, err := fetchHTTPSourceBytes(server.URL, config)
	require.NoError(t, err)
	assert.Equal(t, "openapi: 3.1.0", string(b))
}

func TestFetchHTTPSourceBytes_RealHTTP_StatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer server.Close()

	config := &ResolveConfig{
		MaxBodySize: 1024,
		Timeout:     5e9,
	}
	_, err := fetchHTTPSourceBytes(server.URL, config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status code 500")
}

func TestFetchHTTPSourceBytes_RealHTTP_BodyExceedsMax(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("this is a very long response body"))
	}))
	defer server.Close()

	config := &ResolveConfig{
		MaxBodySize: 5,
		Timeout:     5e9,
	}
	_, err := fetchHTTPSourceBytes(server.URL, config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds max size")
}

// ---------------------------------------------------------------------------
// readFileWithLimit
// ---------------------------------------------------------------------------

func TestReadFileWithLimit_Normal(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.yaml")
	require.NoError(t, os.WriteFile(path, []byte("openapi: 3.1.0"), 0o600))

	b, err := readFileWithLimit(path, 1024)
	require.NoError(t, err)
	assert.Equal(t, "openapi: 3.1.0", string(b))
}

func TestReadFileWithLimit_FileTooLarge(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "large.yaml")
	require.NoError(t, os.WriteFile(path, []byte("this is too much data"), 0o600))

	_, err := readFileWithLimit(path, 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds max size")
}

func TestReadFileWithLimit_MissingFile(t *testing.T) {
	_, err := readFileWithLimit("/nonexistent/path/file.yaml", 1024)
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// resolveFilePath
// ---------------------------------------------------------------------------

func TestResolveFilePath_AbsolutePath_NoRoots(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.yaml")
	require.NoError(t, os.WriteFile(path, []byte("content"), 0o600))

	resolved, err := resolveFilePath(path, nil)
	require.NoError(t, err)
	assert.Equal(t, path, resolved)
}

func TestResolveFilePath_RelativePath_NoRoots(t *testing.T) {
	// With no roots, relative paths resolve from cwd
	resolved, err := resolveFilePath("test.yaml", nil)
	require.NoError(t, err)
	assert.True(t, filepath.IsAbs(resolved))
}

func TestResolveFilePath_RelativeWithRoots_Found(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "specs", "api.yaml")
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "specs"), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("content"), 0o600))

	resolved, err := resolveFilePath("specs/api.yaml", []string{tmpDir})
	require.NoError(t, err)
	assert.Contains(t, resolved, "api.yaml")
}

func TestResolveFilePath_RelativeWithRoots_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := resolveFilePath("nonexistent.yaml", []string{tmpDir})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found within configured roots")
}

func TestResolveFilePath_AbsoluteOutsideRoots(t *testing.T) {
	tmpDir := t.TempDir()
	otherDir := t.TempDir()
	path := filepath.Join(otherDir, "test.yaml")
	require.NoError(t, os.WriteFile(path, []byte("content"), 0o600))

	_, err := resolveFilePath(path, []string{tmpDir})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "outside configured roots")
}

func TestResolveFilePath_AbsoluteInsideRoots(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "api.yaml")
	require.NoError(t, os.WriteFile(path, []byte("content"), 0o600))

	resolved, err := resolveFilePath(path, []string{tmpDir})
	require.NoError(t, err)
	assert.Equal(t, path, resolved)
}

// ---------------------------------------------------------------------------
// isPathWithinRoots
// ---------------------------------------------------------------------------

func TestIsPathWithinRoots_InsideRoot(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "sub", "file.yaml")
	assert.True(t, isPathWithinRoots(path, []string{tmpDir}))
}

func TestIsPathWithinRoots_OutsideRoot(t *testing.T) {
	tmpDir := t.TempDir()
	otherDir := t.TempDir()
	path := filepath.Join(otherDir, "file.yaml")
	assert.False(t, isPathWithinRoots(path, []string{tmpDir}))
}

func TestIsPathWithinRoots_ExactRoot(t *testing.T) {
	tmpDir := t.TempDir()
	assert.True(t, isPathWithinRoots(tmpDir, []string{tmpDir}))
}

func TestIsPathWithinRoots_MultipleRoots(t *testing.T) {
	root1 := t.TempDir()
	root2 := t.TempDir()
	path := filepath.Join(root2, "file.yaml")
	assert.True(t, isPathWithinRoots(path, []string{root1, root2}))
}

func TestIsPathWithinRoots_ParentTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	// Try to go up from the root using ../
	path := filepath.Join(tmpDir, "..", "escape.yaml")
	assert.False(t, isPathWithinRoots(path, []string{tmpDir}))
}

func TestResolveFilePath_RelativeTraversalBlocked(t *testing.T) {
	tmpDir := t.TempDir()
	// Try to traverse outside root with ../
	_, err := resolveFilePath("../../etc/passwd", []string{tmpDir})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found within configured roots")
}

func TestResolveFilePath_RelativeMultipleRoots_FirstMissingSecondHas(t *testing.T) {
	root1 := t.TempDir()
	root2 := t.TempDir()
	path := filepath.Join(root2, "found.yaml")
	require.NoError(t, os.WriteFile(path, []byte("content"), 0o600))

	resolved, err := resolveFilePath("found.yaml", []string{root1, root2})
	require.NoError(t, err)
	assert.Contains(t, resolved, "found.yaml")
}

func TestResolveFilePath_EncodedPath(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "my api.yaml")
	require.NoError(t, os.WriteFile(path, []byte("content"), 0o600))

	// URL-encoded space
	resolved, err := resolveFilePath(filepath.Join(tmpDir, "my%20api.yaml"), nil)
	require.NoError(t, err)
	assert.Contains(t, resolved, "my api.yaml")
}

// ---------------------------------------------------------------------------
// factoryForType
// ---------------------------------------------------------------------------

func TestFactoryForType_OpenAPIWithFactory(t *testing.T) {
	config := &ResolveConfig{
		OpenAPIFactory: func(u string, b []byte) (any, error) {
			return "openapi-doc", nil
		},
	}
	factory, err := factoryForType("openapi", config)
	require.NoError(t, err)
	require.NotNil(t, factory)
	doc, err := factory("url", nil)
	require.NoError(t, err)
	assert.Equal(t, "openapi-doc", doc)
}

func TestFactoryForType_ArazzoWithFactory(t *testing.T) {
	config := &ResolveConfig{
		ArazzoFactory: func(u string, b []byte) (any, error) {
			return "arazzo-doc", nil
		},
	}
	factory, err := factoryForType("arazzo", config)
	require.NoError(t, err)
	require.NotNil(t, factory)
	doc, err := factory("url", nil)
	require.NoError(t, err)
	assert.Equal(t, "arazzo-doc", doc)
}

func TestFactoryForType_OpenAPINilFactory(t *testing.T) {
	config := &ResolveConfig{}
	_, err := factoryForType("openapi", config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no OpenAPIFactory configured")
}

func TestFactoryForType_ArazzoNilFactory(t *testing.T) {
	config := &ResolveConfig{}
	_, err := factoryForType("arazzo", config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no ArazzoFactory configured")
}

func TestFactoryForType_UnknownType(t *testing.T) {
	config := &ResolveConfig{}
	_, err := factoryForType("graphql", config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown source type")
}

// ---------------------------------------------------------------------------
// containsFold
// ---------------------------------------------------------------------------

func TestContainsFold_MatchFound(t *testing.T) {
	assert.True(t, containsFold([]string{"http", "https", "file"}, "HTTPS"))
}

func TestContainsFold_NoMatch(t *testing.T) {
	assert.False(t, containsFold([]string{"http", "https", "file"}, "ftp"))
}

func TestContainsFold_CaseInsensitive(t *testing.T) {
	assert.True(t, containsFold([]string{"HTTP", "HTTPS"}, "http"))
}

func TestContainsFold_EmptySlice(t *testing.T) {
	assert.False(t, containsFold(nil, "http"))
}

// ---------------------------------------------------------------------------
// Full integration test: ResolveSources with httptest
// ---------------------------------------------------------------------------

func TestResolveSources_HTTPTest_Integration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("openapi: 3.1.0"))
	}))
	defer server.Close()

	doc := &high.Arazzo{
		SourceDescriptions: []*high.SourceDescription{
			{Name: "api", URL: server.URL + "/api.yaml", Type: "openapi"},
		},
	}
	config := &ResolveConfig{
		OpenAPIFactory: func(u string, b []byte) (any, error) {
			return string(b), nil
		},
	}
	resolved, err := ResolveSources(doc, config)
	require.NoError(t, err)
	require.Len(t, resolved, 1)
	assert.Equal(t, "openapi: 3.1.0", resolved[0].Document)
}

func TestResolveSources_FileSource_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "api.yaml")
	require.NoError(t, os.WriteFile(filePath, []byte("openapi: 3.1.0"), 0o600))

	doc := &high.Arazzo{
		SourceDescriptions: []*high.SourceDescription{
			{Name: "local", URL: filePath, Type: "openapi"},
		},
	}
	config := &ResolveConfig{
		OpenAPIFactory: func(u string, b []byte) (any, error) {
			return string(b), nil
		},
	}
	resolved, err := ResolveSources(doc, config)
	require.NoError(t, err)
	require.Len(t, resolved, 1)
	assert.Equal(t, "openapi: 3.1.0", resolved[0].Document)
}

func TestResolveSources_URLValidationFails(t *testing.T) {
	doc := &high.Arazzo{
		SourceDescriptions: []*high.SourceDescription{
			{Name: "api", URL: "ftp://example.com/api.yaml", Type: "openapi"},
		},
	}
	config := &ResolveConfig{
		AllowedSchemes: []string{"https", "http"},
	}
	_, err := ResolveSources(doc, config)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrSourceDescLoadFailed))
}

// ===========================================================================
// validation.go tests
// ===========================================================================

// ---------------------------------------------------------------------------
// validateCriterion
// ---------------------------------------------------------------------------

func TestValidateCriterion_MissingCondition(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].SuccessCriteria = []*high.Criterion{
		{Condition: ""},
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrMissingCondition) {
			found = true
		}
	}
	assert.True(t, found, "expected ErrMissingCondition")
}

func TestValidateCriterion_NonSimpleType_MissingContext(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].SuccessCriteria = []*high.Criterion{
		{
			Condition: "^2\\d{2}$",
			Type:      "regex",
		},
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Error(), "context is required")
}

func TestValidateCriterion_ExpressionType(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].SuccessCriteria = []*high.Criterion{
		{
			Condition: "$.status",
			ExpressionType: &high.CriterionExpressionType{
				Type: "jsonpath",
			},
			Context: "$statusCode",
		},
	}
	result := Validate(doc)
	assert.Nil(t, result)
}

func TestValidateCriterion_InvalidContextExpression(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].SuccessCriteria = []*high.Criterion{
		{
			Condition: "^2\\d{2}$",
			Type:      "regex",
			Context:   "invalid-not-an-expression",
		},
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrInvalidExpression) {
			found = true
		}
	}
	assert.True(t, found, "expected ErrInvalidExpression")
}

func TestValidateCriterion_ValidContextExpression(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].SuccessCriteria = []*high.Criterion{
		{
			Condition: "^2\\d{2}$",
			Type:      "regex",
			Context:   "$statusCode",
		},
	}
	result := Validate(doc)
	assert.Nil(t, result)
}

// ---------------------------------------------------------------------------
// validateCriterionExpressionType
// ---------------------------------------------------------------------------

func TestValidateCriterionExpressionType_MissingType(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].SuccessCriteria = []*high.Criterion{
		{
			Condition: "$.status",
			ExpressionType: &high.CriterionExpressionType{
				Type: "",
			},
			Context: "$statusCode",
		},
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Error(), "missing required 'type'")
}

func TestValidateCriterionExpressionType_JSONPathValidVersion(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].SuccessCriteria = []*high.Criterion{
		{
			Condition: "$.status",
			ExpressionType: &high.CriterionExpressionType{
				Type:    "jsonpath",
				Version: "draft-goessner-dispatch-jsonpath-00",
			},
			Context: "$statusCode",
		},
	}
	result := Validate(doc)
	assert.Nil(t, result)
}

func TestValidateCriterionExpressionType_JSONPathInvalidVersion(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].SuccessCriteria = []*high.Criterion{
		{
			Condition: "$.status",
			ExpressionType: &high.CriterionExpressionType{
				Type:    "jsonpath",
				Version: "invalid-version",
			},
			Context: "$statusCode",
		},
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Error(), "unknown jsonpath version")
}

func TestValidateCriterionExpressionType_XPathValidVersions(t *testing.T) {
	for _, version := range []string{"xpath-30", "xpath-20", "xpath-10"} {
		doc := validMinimalDoc()
		doc.Workflows[0].Steps[0].SuccessCriteria = []*high.Criterion{
			{
				Condition: "//status",
				ExpressionType: &high.CriterionExpressionType{
					Type:    "xpath",
					Version: version,
				},
				Context: "$statusCode",
			},
		}
		result := Validate(doc)
		assert.Nil(t, result, "expected no errors for xpath version %q", version)
	}
}

func TestValidateCriterionExpressionType_XPathInvalidVersion(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].SuccessCriteria = []*high.Criterion{
		{
			Condition: "//status",
			ExpressionType: &high.CriterionExpressionType{
				Type:    "xpath",
				Version: "xpath-99",
			},
			Context: "$statusCode",
		},
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Error(), "unknown xpath version")
}

func TestValidateCriterionExpressionType_JSONPathEmptyVersion(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].SuccessCriteria = []*high.Criterion{
		{
			Condition: "$.status",
			ExpressionType: &high.CriterionExpressionType{
				Type:    "jsonpath",
				Version: "", // empty version is valid
			},
			Context: "$statusCode",
		},
	}
	result := Validate(doc)
	assert.Nil(t, result)
}

func TestValidateCriterionExpressionType_XPathEmptyVersion(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].SuccessCriteria = []*high.Criterion{
		{
			Condition: "//status",
			ExpressionType: &high.CriterionExpressionType{
				Type:    "xpath",
				Version: "", // empty version is valid
			},
			Context: "$statusCode",
		},
	}
	result := Validate(doc)
	assert.Nil(t, result)
}

// ---------------------------------------------------------------------------
// validateFailureActions - workflowId resolving to unknown workflow
// ---------------------------------------------------------------------------

func TestValidateFailureActions_WorkflowIdUnknown(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].OnFailure = []*high.FailureAction{
		{Name: "retryOther", Type: "goto", WorkflowId: "unknownWorkflow"},
	}
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

func TestValidateFailureActions_StepIdNotInWorkflow(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].OnFailure = []*high.FailureAction{
		{Name: "gotoMissing", Type: "goto", StepId: "nonexistentStep"},
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

func TestValidateFailureActions_GotoRequiresTarget(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].OnFailure = []*high.FailureAction{
		{Name: "badGoto", Type: "goto"},
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

func TestValidateFailureActions_RetryAfterNegative(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].OnFailure = []*high.FailureAction{
		{Name: "badRetry", Type: "retry", RetryAfter: ptrFloat64(-1)},
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Error(), "retryAfter must be non-negative")
}

func TestValidateFailureActions_RetryLimitNegative(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].OnFailure = []*high.FailureAction{
		{Name: "badRetry", Type: "retry", RetryLimit: ptrInt64(-1)},
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Error(), "retryLimit must be non-negative")
}

// ---------------------------------------------------------------------------
// validateComponentReference
// ---------------------------------------------------------------------------

func TestValidateComponentReference_FailureActionsRef_NoComponents(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].OnFailure = []*high.FailureAction{
		{Reference: "$components.failureActions.retryDefault"},
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
	assert.True(t, found, "expected ErrUnresolvedComponent")
}

func TestValidateComponentReference_SuccessActions_NilMap(t *testing.T) {
	doc := validMinimalDoc()
	doc.Components = &high.Components{} // no SuccessActions map
	doc.Workflows[0].Steps[0].OnSuccess = []*high.SuccessAction{
		{Reference: "$components.successActions.logAndEnd"},
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
}

func TestValidateComponentReference_FailureActions_NilMap(t *testing.T) {
	doc := validMinimalDoc()
	doc.Components = &high.Components{} // no FailureActions map
	doc.Workflows[0].Steps[0].OnFailure = []*high.FailureAction{
		{Reference: "$components.failureActions.retryDefault"},
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
}

func TestValidateComponentReference_Parameters_NilMap(t *testing.T) {
	doc := validMinimalDoc()
	doc.Components = &high.Components{} // no Parameters map
	doc.Workflows[0].Steps[0].Parameters = []*high.Parameter{
		{Reference: "$components.parameters.token"},
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
}

func TestValidateComponentReference_EmptyComponentName(t *testing.T) {
	params := orderedmap.New[string, *high.Parameter]()
	params.Set("p", &high.Parameter{Name: "p", In: "header", Value: &yaml.Node{Kind: yaml.ScalarNode, Value: "v"}})

	doc := validMinimalDoc()
	doc.Components = &high.Components{Parameters: params}
	doc.Workflows[0].Steps[0].Parameters = []*high.Parameter{
		{Reference: "$components.parameters."},
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Error(), "empty component name")
}

// ---------------------------------------------------------------------------
// validateFailureActions - missing name and missing type
// ---------------------------------------------------------------------------

func TestValidateFailureActions_MissingName(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].OnFailure = []*high.FailureAction{
		{Name: "", Type: "end"},
	}
	result := Validate(doc)
	require.NotNil(t, result)
	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrMissingActionName) {
			found = true
		}
	}
	assert.True(t, found, "expected ErrMissingActionName on failure action")
}

func TestValidateFailureActions_MissingType(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].OnFailure = []*high.FailureAction{
		{Name: "action1", Type: ""},
	}
	result := Validate(doc)
	require.NotNil(t, result)
	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrMissingActionType) {
			found = true
		}
	}
	assert.True(t, found, "expected ErrMissingActionType on failure action")
}

// ---------------------------------------------------------------------------
// Workflow-level failure actions
// ---------------------------------------------------------------------------

func TestValidate_WorkflowLevelFailureActions_UnresolvedWorkflow(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].FailureActions = []*high.FailureAction{
		{Name: "gotoMissing", Type: "goto", WorkflowId: "unknownWorkflow"},
	}
	result := Validate(doc)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	found := false
	for _, e := range result.Errors {
		if errors.Is(e.Cause, ErrUnresolvedWorkflowRef) {
			found = true
		}
	}
	assert.True(t, found)
}

// ---------------------------------------------------------------------------
// Criterion type "simple" with context: covers simple path in validateCriterion
// ---------------------------------------------------------------------------

func TestValidateCriterion_SimpleTypeNoContextOK(t *testing.T) {
	doc := validMinimalDoc()
	doc.Workflows[0].Steps[0].SuccessCriteria = []*high.Criterion{
		{
			Condition: "$statusCode == 200",
			// No Type set and no Context set: simple type, context not required
		},
	}
	result := Validate(doc)
	assert.Nil(t, result)
}

// ===========================================================================
// errors.go - additional coverage
// ===========================================================================

func TestValidationResult_Error_WithMultipleErrors(t *testing.T) {
	r := &ValidationResult{
		Errors: []*ValidationError{
			{Path: "a", Cause: fmt.Errorf("error1")},
			{Path: "b", Cause: fmt.Errorf("error2")},
		},
	}
	s := r.Error()
	assert.Contains(t, s, "error1")
	assert.Contains(t, s, "error2")
	assert.Contains(t, s, "; ")
}

func TestValidationResult_HasErrors_True(t *testing.T) {
	r := &ValidationResult{
		Errors: []*ValidationError{{Path: "a", Cause: fmt.Errorf("err")}},
	}
	assert.True(t, r.HasErrors())
}

func TestValidationResult_HasWarnings_True(t *testing.T) {
	r := &ValidationResult{
		Warnings: []*Warning{{Path: "a", Message: "warn"}},
	}
	assert.True(t, r.HasWarnings())
}

// ===========================================================================
// ===========================================================================
// setJSONPointerValue / applyPayloadReplacements
// ===========================================================================

func TestSetJSONPointerValue_Simple(t *testing.T) {
	root := map[string]any{"name": "old"}
	err := setJSONPointerValue(root, "/name", "new")
	require.NoError(t, err)
	assert.Equal(t, "new", root["name"])
}

func TestSetJSONPointerValue_Nested(t *testing.T) {
	root := map[string]any{"user": map[string]any{"name": "old"}}
	err := setJSONPointerValue(root, "/user/name", "new")
	require.NoError(t, err)
	assert.Equal(t, "new", root["user"].(map[string]any)["name"])
}

func TestSetJSONPointerValue_IntermediateCreation(t *testing.T) {
	root := map[string]any{}
	err := setJSONPointerValue(root, "/a/b", "value")
	require.NoError(t, err)
	assert.Equal(t, "value", root["a"].(map[string]any)["b"])
}

func TestSetJSONPointerValue_EmptyPointer(t *testing.T) {
	root := map[string]any{}
	err := setJSONPointerValue(root, "", "x")
	assert.Error(t, err)
}

func TestSetJSONPointerValue_NoLeadingSlash(t *testing.T) {
	root := map[string]any{}
	err := setJSONPointerValue(root, "name", "x")
	assert.Error(t, err)
}

func TestSetJSONPointerValue_EscapedSegments(t *testing.T) {
	root := map[string]any{}
	err := setJSONPointerValue(root, "/a~1b", "value")
	require.NoError(t, err)
	assert.Equal(t, "value", root["a/b"])
}

func TestApplyPayloadReplacements_NonMapPayload(t *testing.T) {
	engine := &Engine{config: &EngineConfig{}}
	_, err := engine.applyPayloadReplacements("not a map", nil, nil, "step1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "non-object")
}

func TestApplyPayloadReplacements_EmptyReplacements(t *testing.T) {
	engine := &Engine{config: &EngineConfig{}}
	result, err := engine.applyPayloadReplacements(map[string]any{"a": 1}, nil, nil, "step1")
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"a": 1}, result)
}

// ===========================================================================
// $url and $method in expression context
// ===========================================================================

func TestExecuteStep_URLAndMethod(t *testing.T) {
	executor := &captureExecutor{
		response: &ExecutionResponse{
			StatusCode: 200,
			URL:        "https://api.example.com/pets/123",
			Method:     "GET",
		},
	}
	doc := &high.Arazzo{
		Arazzo: "1.0.1",
		Workflows: []*high.Workflow{
			{
				WorkflowId: "test",
				Steps: []*high.Step{
					{StepId: "s1", OperationId: "getPet"},
				},
			},
		},
	}
	engine := NewEngine(doc, executor, nil)
	result, err := engine.RunWorkflow(context.Background(), "test", nil)
	require.NoError(t, err)
	require.True(t, result.Success)
}

// Helper
// ===========================================================================

func mustParseURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return u
}
