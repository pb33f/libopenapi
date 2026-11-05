// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"

	"github.com/pb33f/libopenapi/arazzo/expression"
	high "github.com/pb33f/libopenapi/datamodel/high/arazzo"
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

func TestEvaluateCriterion_SimpleCondition_NumberAndBooleanLiterals(t *testing.T) {
	var body yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(`{"zero": 0, "one": 1, "yes": true, "no": false}`), &body))
	ctx := &expression.Context{ResponseBody: &body}

	for condition, want := range map[string]bool{
		"$response.body#/zero == 0":     true,
		"$response.body#/one == 1":      true,
		"$response.body#/one != 1":      false,
		"$response.body#/one > 0":       true,
		"$response.body#/yes == true":   true,
		"$response.body#/no == false":   true,
		"$response.body#/yes == 1":      false,
		"$response.body#/no == 0":       false,
		"$response.body#/one == true":   false,
		"$response.body#/zero == false": false,
	} {
		ok, err := EvaluateCriterion(&high.Criterion{Condition: condition}, ctx)
		require.NoError(t, err, condition)
		assert.Equal(t, want, ok, condition)
	}
}

func TestEvaluateCriterion_SimpleCondition_BooleanOperators(t *testing.T) {
	criterion := &high.Criterion{
		Condition: `$statusCode == 204 || ($statusCode == 200 && ($response.body#/challengeName == 'NEW_PASSWORD_REQUIRED' || $response.body#/challengeName == 'SOFTWARE_TOKEN_MFA'))`,
	}

	ok, err := EvaluateCriterion(criterion, &expression.Context{StatusCode: 204})
	require.NoError(t, err)
	assert.True(t, ok, "204 with an empty body should pass")

	ok, err = EvaluateCriterion(criterion, &expression.Context{
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

// A missing runtime value resolves to null. Null has no ordering, so every
// ordering comparison against it must fail; only == and != are meaningful.
func TestEvaluateCriterion_MissingValueComparisons(t *testing.T) {
	cases := []struct {
		condition string
		want      bool
	}{
		{"$response.body#/count > 0", false},
		{"$response.body#/count >= 0", false},
		{"$response.body#/count < 5", false},
		{"$response.body#/count <= 0", false},
		{"$response.body#/count == 0", false},
		{"$response.body#/count != 0", true},
		{"$response.body#/name == ''", false},
		{"$response.header.X-Count > 0", false},
		{"$statusCode == 204 && $response.body#/count > 0", false},
		{"$statusCode == 204 || $response.body#/count > 0", true},
		{"$response.body#/count > 0 || $statusCode == 204", true},
	}
	for _, tc := range cases {
		t.Run(tc.condition, func(t *testing.T) {
			ok, err := EvaluateCriterion(&high.Criterion{Condition: tc.condition},
				&expression.Context{StatusCode: 204})
			require.NoError(t, err)
			assert.Equal(t, tc.want, ok)
		})
	}
}

// Every kind of absent response or request value is null, whether the source
// map is nil or present without the key, and whether a JSON Pointer misses a
// key, overruns an array or walks into a scalar.
func TestEvaluateCriterion_AbsentRuntimeValuesAreNull(t *testing.T) {
	body := yamlMapping(t, map[string]any{"items": []any{}, "data": nil})
	noHeaders := &expression.Context{StatusCode: 200, ResponseBody: body}
	otherHeaders := &expression.Context{
		StatusCode:     200,
		ResponseBody:   body,
		RequestHeaders: map[string]string{"Other": "1"},
		RequestQuery:   map[string]string{"other": "1"},
		RequestPath:    map[string]string{"other": "1"},
	}
	// A null operand is unequal to any literal and falsy as a bare operand.
	var conditions []string
	for _, operand := range []string{
		"$response.body#/items/0/id",
		"$response.body#/data/id",
		"$request.header.X",
		"$request.query.x",
		"$request.path.x",
	} {
		conditions = append(conditions,
			operand+" != 0",
			"$statusCode == 200 && "+operand+" != 0",
			operand+" || $statusCode == 200",
		)
	}
	for name, ctx := range map[string]*expression.Context{"no sources": noHeaders, "other keys": otherHeaders} {
		for _, condition := range conditions {
			t.Run(name+"/"+condition, func(t *testing.T) {
				ok, err := EvaluateCriterion(&high.Criterion{Condition: condition}, ctx)
				require.NoError(t, err)
				assert.True(t, ok)
			})
		}
	}
}

// Only request and response values are optional. A pointer miss inside a step
// output, an absent payload, or a nil context is still an error.
func TestEvaluateCriterion_NonExchangeMissingValuesError(t *testing.T) {
	ctx := &expression.Context{
		StatusCode: 200,
		Steps: map[string]*expression.StepContext{
			"s": {Outputs: map[string]any{"body": map[string]any{}}},
		},
	}
	cases := []struct {
		condition string
		ctx       *expression.Context
		errText   string
	}{
		{"$steps.s.outputs.body#/missing == 1", ctx, `JSON pointer segment "missing" not found`},
		{"$response.payload == 1", ctx, "no response payload available"},
		{"$response.header.X == 1", nil, "nil context"},
	}
	for _, tc := range cases {
		t.Run(tc.condition, func(t *testing.T) {
			_, err := EvaluateCriterion(&high.Criterion{Condition: tc.condition}, tc.ctx)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errText)
		})
	}
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

// A missing value used as a bare boolean operand is falsy rather than an error,
// so the rest of the condition still decides the outcome.
func TestEvaluateCriterion_MissingBareOperandIsFalse(t *testing.T) {
	cases := []struct {
		condition string
		want      bool
	}{
		{"$response.body#/flag", false},
		{"$response.header.X-Flag", false},
		{"$statusCode == 204 || $response.body#/flag", true},
		{"$response.body#/flag || $statusCode == 204", true},
		{"$statusCode == 204 && $response.body#/flag", false},
		{"($response.body#/flag) || $statusCode == 204", true},
	}
	for _, tc := range cases {
		t.Run(tc.condition, func(t *testing.T) {
			ok, err := EvaluateCriterion(&high.Criterion{Condition: tc.condition},
				&expression.Context{StatusCode: 204})
			require.NoError(t, err)
			assert.Equal(t, tc.want, ok)
		})
	}
}

// && and || stop once the left side decides the result. Each right side here
// would fail if evaluated: an unknown step, or a non-boolean bare operand.
func TestEvaluateCriterion_ShortCircuit(t *testing.T) {
	cases := []struct {
		condition string
		want      bool
	}{
		{"$statusCode == 204 || $steps.nope.outputs.x == 1", true},
		{"$statusCode == 200 && $steps.nope.outputs.x == 1", false},
		{"true || $statusCode", true},
		{"false && $statusCode", false},
		{"$statusCode == 200 || $statusCode == 204 || $steps.nope.outputs.x == 1", true},
		{"$statusCode == 200 && $steps.nope.outputs.x == 1 || $statusCode == 204", true},
		{"$statusCode == 204 || ($steps.nope.outputs.x == 1 && $statusCode)", true},
		{"($statusCode == 200 && $statusCode) || $method == 'GET'", true},
	}
	for _, tc := range cases {
		t.Run(tc.condition, func(t *testing.T) {
			ok, err := EvaluateCriterion(&high.Criterion{Condition: tc.condition},
				&expression.Context{StatusCode: 204, Method: "GET", Steps: map[string]*expression.StepContext{}})
			require.NoError(t, err)
			assert.Equal(t, tc.want, ok)
		})
	}
}

// Bare true and false take the fast path; other spellings are not booleans.
func TestEvaluateCriterion_BareBooleanLiterals(t *testing.T) {
	for condition, want := range map[string]bool{"true": true, "false": false, "  true\t": true, "\nfalse ": false} {
		ok, err := EvaluateCriterion(&high.Criterion{Condition: condition}, &expression.Context{})
		require.NoError(t, err, condition)
		assert.Equal(t, want, ok, condition)
	}
	for _, condition := range []string{"TRUE", "True", "t", "1", "0"} {
		_, err := EvaluateCriterion(&high.Criterion{Condition: condition}, &expression.Context{})
		assert.ErrorContains(t, err, "did not evaluate to a boolean", condition)
	}
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
		// A syntax error after ||, after && or inside a group propagates out.
		{"$statusCode == 204 ||", "unexpected token"},
		{"$statusCode == 204 &&", "unexpected token"},
		{"($statusCode == 204 &&", "unexpected token"},
		// Non-optional errors still propagate from every position in the parse.
		{"($steps.nope.outputs.x == 1)", `step "nope" not found`},
		{"$statusCode == 200 || $steps.nope.outputs.x == 1", `step "nope" not found`},
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

// Comparison operator characters are legal in JSON Pointer reference tokens,
// so they belong to the runtime expression rather than splitting the condition.
func TestEvaluateCriterion_OperatorCharsInsideJSONPointer(t *testing.T) {
	body := yamlMapping(t, map[string]any{"a>b": 1, "x==y": "ok", "p<=q": 3})
	ctx := &expression.Context{StatusCode: 200, ResponseBody: body}

	cases := []struct {
		condition string
		want      bool
	}{
		{"$response.body#/a>b == 1", true},
		{"$response.body#/a>b == 2", false},
		{"$response.body#/x==y == 'ok'", true},
		{"$response.body#/p<=q > 2", true},
		{"$statusCode == 200 && $response.body#/a>b == 1", true},
	}
	for _, tc := range cases {
		t.Run(tc.condition, func(t *testing.T) {
			ok, err := EvaluateCriterion(&high.Criterion{Condition: tc.condition}, ctx)
			require.NoError(t, err)
			assert.Equal(t, tc.want, ok)
		})
	}
}

func TestCompileSimpleCondition_CachesTreeAndErrors(t *testing.T) {
	caches := newCriterionCaches()

	first, err := compileSimpleCondition("$statusCode == 200", caches)
	require.NoError(t, err)
	second, err := compileSimpleCondition("$statusCode == 200", caches)
	require.NoError(t, err)
	assert.Same(t, first, second, "a repeated condition must reuse the compiled tree")

	_, err = compileSimpleCondition("$statusCode == 'abc", caches)
	require.ErrorContains(t, err, "unterminated string")
	_, err = compileSimpleCondition("$statusCode == 'abc", caches)
	require.ErrorContains(t, err, "unterminated string")
	assert.Len(t, caches.simple, 2, "syntax errors are cached alongside compiled trees")

	// Without caches every call compiles a fresh tree.
	a, err := compileSimpleCondition("$statusCode == 200", nil)
	require.NoError(t, err)
	b, err := compileSimpleCondition("$statusCode == 200", nil)
	require.NoError(t, err)
	assert.NotSame(t, a, b)
}

// The cached tree holds only literals and expression text, never runtime values,
// so reusing it across contexts must not leak a previous evaluation's result.
func TestEvaluateCriterion_CachedSimpleConditionUsesCurrentContext(t *testing.T) {
	caches := newCriterionCaches()
	criterion := &high.Criterion{Condition: "$statusCode == 200 && ($method == 'POST' || $method == 'PUT')"}

	cases := []struct {
		ctx  *expression.Context
		want bool
	}{
		{&expression.Context{StatusCode: 200, Method: "POST"}, true},
		{&expression.Context{StatusCode: 500, Method: "POST"}, false},
		{&expression.Context{StatusCode: 200, Method: "PUT"}, true},
		{&expression.Context{StatusCode: 200, Method: "GET"}, false},
	}
	for _, tc := range cases {
		ok, err := evaluateCriterionImpl(criterion, tc.ctx, caches)
		require.NoError(t, err)
		assert.Equal(t, tc.want, ok, "%d %s", tc.ctx.StatusCode, tc.ctx.Method)
	}
	assert.Len(t, caches.simple, 1)
}

func TestEvaluateCriterion_SimpleConditionGroupsAndLiterals(t *testing.T) {
	ctx := &expression.Context{StatusCode: 200}
	cases := []struct {
		condition string
		want      bool
	}{
		// A parenthesised expression is a boolean operand of the outer comparison.
		{"($statusCode == 200) == true", true},
		{"($statusCode == 500) != false", false},
		{"(true)", true},
		{"((false) || ($statusCode >= 200 && $statusCode < 300))", true},
		// Literals compare by their pre-converted types.
		{"1 == 1.0", true},
		{"2 > 10", false},
		{"false == false", true},
	}
	for _, tc := range cases {
		t.Run(tc.condition, func(t *testing.T) {
			for _, caches := range []*criterionCaches{nil, newCriterionCaches()} {
				ok, err := evaluateCriterionImpl(&high.Criterion{Condition: tc.condition}, ctx, caches)
				require.NoError(t, err)
				assert.Equal(t, tc.want, ok)
			}
		})
	}
}

// Conditions compile before any operand is evaluated, so a syntax error wins
// over a runtime error from an operand earlier in the string.
func TestEvaluateCriterion_SyntaxErrorReportedBeforeEvaluation(t *testing.T) {
	_, err := EvaluateCriterion(&high.Criterion{Condition: "$steps.nope.outputs.x == 1 )"},
		&expression.Context{Steps: map[string]*expression.StepContext{}})
	require.ErrorContains(t, err, `unexpected token ")"`)
}

func TestEvaluateCriterion_ParenthesisedNonBooleanErrors(t *testing.T) {
	_, err := EvaluateCriterion(&high.Criterion{Condition: "($statusCode) == true"},
		&expression.Context{StatusCode: 200})
	require.ErrorContains(t, err, "did not evaluate to a boolean")
}

// A runtime expression runs to whitespace, so a ')' it does not balance itself
// closes an open group, while parentheses inside a pointer stay in the pointer.
func TestEvaluateCriterion_RuntimeOperandParentheses(t *testing.T) {
	body := yamlMapping(t, map[string]any{"flag": true, "f(x)": 1, "g)": 2})
	ctx := &expression.Context{StatusCode: 200, ResponseBody: body}

	cases := []struct {
		condition string
		want      bool
	}{
		{"($response.body#/flag)", true},
		{"(($response.body#/flag))", true},
		{"($statusCode == 500 || $response.body#/flag)", true},
		{"$response.body#/f(x) == 1", true},
		{"($response.body#/f(x) == 1)", true},
		{"(($response.body#/f(x) == 1))", true},
		// Outside any group a trailing ')' belongs to the pointer.
		{"$response.body#/g) == 2", true},
	}
	for _, tc := range cases {
		t.Run(tc.condition, func(t *testing.T) {
			ok, err := EvaluateCriterion(&high.Criterion{Condition: tc.condition}, ctx)
			require.NoError(t, err)
			assert.Equal(t, tc.want, ok)
		})
	}
}

func TestEvaluateCriterion_NonASCIIOperands(t *testing.T) {
	// "Å" is C3 85 and "à" is C3 A0; the continuation bytes must not split tokens.
	body := yamlMapping(t, map[string]any{"Åsa": "à"})
	ctx := &expression.Context{ResponseBody: body}

	for condition, want := range map[string]bool{
		"$response.body#/Åsa == 'à'": true,
		"$response.body#/Åsa == à":   true,
		"Å == à":                     false,
		"Å == Å":                     true,
	} {
		ok, err := EvaluateCriterion(&high.Criterion{Condition: condition}, ctx)
		require.NoError(t, err, condition)
		assert.Equal(t, want, ok, condition)
	}
}

func TestTokenizeSimpleCondition_RuntimeOperandEndsAtWhitespace(t *testing.T) {
	cases := map[string][]string{
		"$response.body#/a>b == 1":      {"$response.body#/a>b", "==", "1"},
		"$response.header.X&&Y != 'z'":  {"$response.header.X&&Y", "!=", "'z'"},
		"($response.body#/flag)":        {"(", "$response.body#/flag", ")"},
		"(($response.body#/f(x)))":      {"(", "(", "$response.body#/f(x)", ")", ")"},
		"$response.body#/g)":            {"$response.body#/g)"},
		"$statusCode == 1&&$statusCode": {"$statusCode", "==", "1", "&&", "$statusCode"},
		"$statusCode\t==\n200":          {"$statusCode", "==", "200"},
	}
	for input, want := range cases {
		tokens, err := tokenizeSimpleCondition(input)
		require.NoError(t, err, input)
		got := make([]string, len(tokens))
		for i, tok := range tokens {
			got[i] = tok.raw
		}
		assert.Equal(t, want, got, input)
	}
}

func TestIsASCIISpace(t *testing.T) {
	for _, c := range []byte(" \t\n\r\v\f") {
		assert.True(t, isASCIISpace(c), "%q", c)
	}
	for _, c := range []byte{'a', '$', ')', 0x85, 0xA0} {
		assert.False(t, isASCIISpace(c), "%q", c)
	}
}

// A runtime expression runs to whitespace, so an operator written directly
// after one is part of the expression: `$statusCode==200` is a single operand
// and fails to parse, as on main. A literal stops at an operator, so the
// operator may follow it without a space.
func TestEvaluateCriterion_OperatorsWithoutSpaces(t *testing.T) {
	ctx := &expression.Context{StatusCode: 200}

	cases := []struct {
		condition string
		want      bool
	}{
		{"$statusCode ==200", true},
		{"200==$statusCode", true},
		{"201!=$statusCode", true},
		{"$statusCode == 200&&true", true},
		{"$statusCode == 201||true", true},
	}
	for _, tc := range cases {
		t.Run(tc.condition, func(t *testing.T) {
			ok, err := EvaluateCriterion(&high.Criterion{Condition: tc.condition}, ctx)
			require.NoError(t, err)
			assert.Equal(t, tc.want, ok)
		})
	}

	errorCases := []struct {
		condition string
		errText   string
	}{
		// The whole string is one runtime expression.
		{"$statusCode==200", `unknown expression: "$statusCode==200"`},
		{"$statusCode&&true", `unknown expression: "$statusCode&&true"`},
		// The expression is `$statusCode==`; `200` is left over.
		{"$statusCode== 200", `unexpected token "200"`},
	}
	for _, tc := range errorCases {
		t.Run(tc.condition, func(t *testing.T) {
			_, err := EvaluateCriterion(&high.Criterion{Condition: tc.condition}, ctx)
			require.ErrorContains(t, err, tc.errText)
		})
	}
}

// A runtime expression runs to whitespace. Trailing ')' characters it does not
// balance itself close open groups first, one per open group; any left over
// stay in the pointer. Inside a group, a pointer ending in an unbalanced ')'
// is therefore only reachable as the group's last operand, with the closing
// ')' written directly after it. The body holds both "h" and "h)" so each case
// shows which key was read.
func TestEvaluateCriterion_PointerEndingInParen(t *testing.T) {
	body := yamlMapping(t, map[string]any{"g)": 2, "h": true, "h)": false})
	ctx := &expression.Context{StatusCode: 200, ResponseBody: body}

	cases := []struct {
		condition string
		want      bool
	}{
		// Outside any group the ')' belongs to the pointer.
		{"$response.body#/g) == 2", true},
		{"$response.body#/h) || $statusCode == 404", false},
		// One open group takes one trailing ')', so the pointer is /h.
		{"($response.body#/h)", true},
		{"(($response.body#/h))", true},
		// A ')' beyond the open groups stays in the pointer, so it is /h).
		{"($response.body#/h))", false},
		{"(($response.body#/h)))", false},
		{"(2 == $response.body#/g))", true},
		{"($statusCode == 200 && 2 == $response.body#/g))", true},
		{"($response.body#/h)) || $statusCode == 200", true},
	}
	for _, tc := range cases {
		t.Run(tc.condition, func(t *testing.T) {
			ok, err := EvaluateCriterion(&high.Criterion{Condition: tc.condition}, ctx)
			require.NoError(t, err)
			assert.Equal(t, tc.want, ok)
		})
	}

	// Followed by more of the group, the pointer's ')' closes the group instead,
	// leaving the group's real ')' unmatched.
	_, err := EvaluateCriterion(&high.Criterion{Condition: "($response.body#/g) == 2 )"}, ctx)
	require.ErrorContains(t, err, `unexpected token ")"`)
}
