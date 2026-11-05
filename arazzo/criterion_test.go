// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/arazzo/expression"
	high "github.com/pb33f/libopenapi/datamodel/high/arazzo"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestEvaluateCriterion_SimpleCondition_StatusCodeComparison(t *testing.T) {
	criterion := &high.Criterion{
		Condition: "$statusCode == 200",
	}

	ok, err := EvaluateCriterion(criterion, &expression.Context{StatusCode: 200})
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = EvaluateCriterion(criterion, &expression.Context{StatusCode: 500})
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestEvaluateCriterion_SimpleCondition_StringComparison(t *testing.T) {
	criterion := &high.Criterion{
		Condition: "$method == \"POST\"",
	}

	ok, err := EvaluateCriterion(criterion, &expression.Context{Method: "POST"})
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestEvaluateCriterion_SimpleCondition_BooleanOperators(t *testing.T) {
	criterion := &high.Criterion{
		Condition: `$statusCode == 204 || ($statusCode == 200 && ($response.body#/challengeName == 'NEW_PASSWORD_REQUIRED' || $response.body#/challengeName == 'SOFTWARE_TOKEN_MFA'))`,
	}

	ok, err := EvaluateCriterion(criterion, &expression.Context{
		StatusCode:   200,
		ResponseBody: yamlMapping(t, map[string]any{"challengeName": "NEW_PASSWORD_REQUIRED"}),
	})
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = EvaluateCriterion(criterion, &expression.Context{
		StatusCode:   200,
		ResponseBody: yamlMapping(t, map[string]any{"challengeName": "MFA_SETUP"}),
	})
	require.NoError(t, err)
	assert.False(t, ok, "unsupported challenge must fail")
}

func yamlMapping(t *testing.T, v any) *yaml.Node {
	t.Helper()
	b, err := yaml.Marshal(v)
	require.NoError(t, err)
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal(b, &node))
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return node.Content[0]
	}
	return &node
}

// An explicit JSON null in a body has no ordering and is falsy as a bare operand.
func TestEvaluateCriterion_NullBodyValueComparisons(t *testing.T) {
	body := yamlMapping(t, map[string]any{"count": nil})
	ctx := &expression.Context{StatusCode: 200, ResponseBody: body}

	ok, err := EvaluateCriterion(&high.Criterion{Condition: "$response.body#/count > 0"}, ctx)
	require.NoError(t, err)
	assert.False(t, ok)

	ok, err = EvaluateCriterion(&high.Criterion{Condition: "$response.body#/count != 0"}, ctx)
	require.NoError(t, err)
	assert.True(t, ok)

	// A bare null operand is falsy, leaving the rest of the condition to decide.
	ok, err = EvaluateCriterion(&high.Criterion{Condition: "$response.body#/count"}, ctx)
	require.NoError(t, err)
	assert.False(t, ok)

	ok, err = EvaluateCriterion(&high.Criterion{Condition: "$response.body#/count || $statusCode == 200"}, ctx)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestCompareSimpleValues_Nil(t *testing.T) {
	ok, err := compareSimpleValues(nil, nil, "==")
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = compareSimpleValues(nil, nil, "!=")
	require.NoError(t, err)
	assert.False(t, ok)

	ok, err = compareSimpleValues(nil, int64(1), ">")
	require.NoError(t, err)
	assert.False(t, ok)

	_, err = compareSimpleValues(nil, int64(1), "~=")
	assert.Error(t, err)
}

// A present but non-boolean bare operand is still an authoring error.
func TestEvaluateCriterion_NonBooleanBareOperandErrors(t *testing.T) {
	_, err := EvaluateCriterion(&high.Criterion{Condition: "$statusCode"},
		&expression.Context{StatusCode: 204})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "did not evaluate to a boolean")
}

func TestEvaluateCriterion_SimpleConditionSyntaxErrors(t *testing.T) {
	cases := []struct {
		condition string
		errText   string
	}{
		{"$statusCode == 'abc", "unterminated string"},
		{"$statusCode == 204 )", `unexpected token ")"`},
		{"($statusCode == 204", "missing closing parenthesis"},
		{"&& $statusCode == 204", `unexpected token "&&"`},
		{"$statusCode ==", "unexpected token"},
		// Non-optional errors still propagate from every position in the parse.
		{"($steps.nope.outputs.x == 1)", `step "nope" not found`},
		{"$statusCode == 204 || $steps.nope.outputs.x == 1", `step "nope" not found`},
		{"$statusCode == 204 && $steps.nope.outputs.x == 1", `step "nope" not found`},
		{"$steps.nope.outputs.x == 1 || $statusCode == 204", `step "nope" not found`},
		{"$statusCode == $steps.nope.outputs.x", `step "nope" not found`},
	}
	for _, tc := range cases {
		t.Run(tc.condition, func(t *testing.T) {
			_, err := EvaluateCriterion(&high.Criterion{Condition: tc.condition},
				&expression.Context{StatusCode: 204, Steps: map[string]*expression.StepContext{}})
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errText)
		})
	}
}

func TestTokenizeSimpleCondition_TrailingWhitespace(t *testing.T) {
	tokens, err := tokenizeSimpleCondition("$statusCode == 204 \t")
	require.NoError(t, err)
	require.Len(t, tokens, 3)
	assert.Equal(t, tokOperand, tokens[0].kind)
	assert.Equal(t, tokEq, tokens[1].kind)
	assert.Equal(t, "204", tokens[2].raw)
}

func TestIsBoolOpStart(t *testing.T) {
	assert.False(t, isBoolOpStart(""))
	assert.False(t, isBoolOpStart("abc"))
	for _, s := range []string{"&&", "||", "==", "!=", ">=", "<=", ">", "<"} {
		assert.True(t, isBoolOpStart(s+" x"), s)
	}
}
