// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package expression

import (
	"testing"

	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

// Arazzo 1.0.1 section 4.7 defines a general production, "$components." name, where
// name = *( CHAR ) and so may contain dots. Arazzo 1.1.0 replaced it with
// components-reference = component-type "." component-name, restricted to three types.
// A reference that is legal in 1.0 must not be reported as invalid when the caller
// states that the document is 1.0.
func TestParseWithVersion_ComponentsGeneralProductionIsArazzo10Only(t *testing.T) {
	const input = "$components.inputs.someInput"

	expr, err := ParseWithVersion(input, Arazzo10)
	require.NoError(t, err, "1.0.1 permits the general $components. name production")
	assert.Equal(t, Components, expr.Type)
	assert.Equal(t, "inputs", expr.Name, "component type is carried in Name")
	assert.Equal(t, "someInput", expr.Tail, "remaining path is carried in Tail")
	assert.Equal(t, input, expr.Raw)

	_, err = ParseWithVersion(input, Arazzo11)
	require.Error(t, err, "1.1.0 removed the general production")
	assert.Contains(t, err.Error(), "unknown component type")
}

func TestParseWithVersion_Arazzo10GeneralNamedProductions(t *testing.T) {
	tests := []struct {
		input    string
		exprType ExpressionType
		name     string
		tail     string
	}{
		{input: "$steps.someStep", exprType: Steps, name: "someStep"},
		{input: "$steps.someStep.arbitrary.tail", exprType: Steps, name: "someStep", tail: "arbitrary.tail"},
		{input: "$workflows.someWorkflow", exprType: Workflows, name: "someWorkflow"},
		{input: "$sourceDescriptions.petstore", exprType: SourceDescriptions, name: "petstore"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			expr, err := ParseWithVersion(test.input, Arazzo10)
			require.NoError(t, err)
			assert.Equal(t, test.exprType, expr.Type)
			assert.Equal(t, test.name, expr.Name)
			assert.Equal(t, test.tail, expr.Tail)

			_, err = ParseWithVersion(test.input, Arazzo11)
			require.Error(t, err)
		})
	}
}

// The version-free entry points must keep their existing 1.1 behavior.
func TestParse_DefaultsToArazzo11Grammar(t *testing.T) {
	_, err := Parse("$components.inputs.someInput")
	require.Error(t, err)

	require.Error(t, Validate("$components.inputs.someInput"))
	require.NoError(t, ValidateWithVersion("$components.inputs.someInput", Arazzo10))
}

// The three component types addressable in 1.1 must keep their specific expression types
// under both grammars, so 1.0 documents are not quietly downgraded to the general form.
func TestParseWithVersion_SpecificComponentTypesUnchangedAcrossVersions(t *testing.T) {
	tests := []struct {
		input    string
		expected ExpressionType
	}{
		{"$components.parameters.token", ComponentParameters},
		{"$components.successActions.retry", ComponentSuccessActions},
		{"$components.failureActions.bail", ComponentFailureActions},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			for _, version := range []SpecVersion{Arazzo10, Arazzo11} {
				expr, err := ParseWithVersion(test.input, version)
				require.NoError(t, err)
				assert.Equal(t, test.expected, expr.Type)
				assert.Empty(t, expr.Tail, "specific forms carry the name in Name, not Tail")
			}
		})
	}
}

// The general 1.0 form still has to be well formed; it is not an escape hatch for
// arbitrary text.
func TestParseWithVersion_Arazzo10GeneralFormStillValidated(t *testing.T) {
	tests := []string{
		"$components.",
		"$components.inputs.",
		"$components.bad type.name",
		"$components.inputs.bad name",
		"$steps.",
		"$steps..outputs.value",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := ParseWithVersion(input, Arazzo10)
			assert.Error(t, err)
		})
	}
}

// The 1.0.1 production is "$components." name, where name = *( CHAR ). A second dot is
// not required, so a single-segment reference is legal 1.0 and must not be rejected.
// 1.1.0 always requires component-type "." component-name.
func TestParseWithVersion_Arazzo10AcceptsSingleSegmentComponentReference(t *testing.T) {
	const input = "$components.someName"

	expr, err := ParseWithVersion(input, Arazzo10)
	require.NoError(t, err)
	assert.Equal(t, Components, expr.Type)
	assert.Equal(t, "someName", expr.Name)
	assert.Empty(t, expr.Tail)

	_, err = ParseWithVersion(input, Arazzo11)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid component reference")
}

// Embedded expressions must honor the selected grammar too, since they delegate to the
// same single-expression parser.
func TestParseEmbeddedWithVersion_HonorsArazzo10Grammar(t *testing.T) {
	const input = "prefix {$components.inputs.someInput} suffix"

	tokens, err := ParseEmbeddedWithVersion(input, Arazzo10)
	require.NoError(t, err)
	require.Len(t, tokens, 3)
	assert.Equal(t, "prefix ", tokens[0].Literal)
	require.True(t, tokens[1].IsExpression)
	assert.Equal(t, Components, tokens[1].Expression.Type)
	assert.Equal(t, " suffix", tokens[2].Literal)

	_, err = ParseEmbeddedWithVersion(input, Arazzo11)
	assert.Error(t, err)
}

// The evaluator has always been able to resolve component inputs; before the grammar was
// version-aware that branch was unreachable because the parser rejected the expression.
func TestEvaluate_Arazzo10ComponentInputsResolves(t *testing.T) {
	expr, err := ParseWithVersion("$components.inputs.someInput", Arazzo10)
	require.NoError(t, err)

	ctx := &Context{
		Components: &ComponentsContext{
			Inputs: map[string]any{"someInput": "resolved-value"},
		},
	}

	value, err := Evaluate(expr, ctx)
	require.NoError(t, err)
	assert.Equal(t, "resolved-value", value)
}

// A missing component input must report the input name rather than fall through to the
// generic unknown-component-type error.
func TestEvaluate_Arazzo10ComponentInputsMissingName(t *testing.T) {
	expr, err := ParseWithVersion("$components.inputs.absent", Arazzo10)
	require.NoError(t, err)

	ctx := &Context{Components: &ComponentsContext{Inputs: map[string]any{}}}
	_, err = Evaluate(expr, ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `component input "absent" not found`)
}

// A component type with no backing context must report that, not a parse failure.
func TestEvaluate_Arazzo10UnknownComponentTypeAtEvaluation(t *testing.T) {
	expr, err := ParseWithVersion("$components.somethingElse.name", Arazzo10)
	require.NoError(t, err, "1.0 grammar accepts any component type syntactically")

	_, err = Evaluate(expr, &Context{Components: &ComponentsContext{}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown component type")
}
