// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package expression

import (
	"errors"
	"strings"
	"testing"

	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestParseArazzo11ConformanceCorpus(t *testing.T) {
	tests := []struct {
		input       string
		exprType    ExpressionType
		name        string
		property    string
		tail        string
		jsonPointer string
	}{
		{"$self", Self, "", "", "", ""},
		{"$request.payload#/items/0", RequestPayload, "", "", "", "/items/0"},
		{"$response.payload", ResponsePayload, "", "", "", ""},
		{"$message.header.X-Correlation-ID", MessageHeader, "", "X-Correlation-ID", "", ""},
		{"$message.query.", MessageQuery, "", "", "", ""},
		{"$message.path.correlation", MessagePath, "", "correlation", "", ""},
		{"$message.body#/event", MessageBody, "", "", "", "/event"},
		{"$message.payload#/order~1id", MessagePayload, "", "", "", "/order~1id"},
		{"$inputs.user.profile#/name", Inputs, "user.profile", "", "", "/name"},
		{"$outputs.result#/", Outputs, "result", "", "", "/"},
		{"$steps.create-order.outputs.order.id#/value", Steps, "create-order", "", "outputs.order.id", "/value"},
		{"$workflows.checkout.inputs.user.id#/value", Workflows, "checkout", "", "inputs.user.id", "/value"},
		{"$workflows.checkout.outputs.receipt#/items/0", Workflows, "checkout", "", "outputs.receipt", "/items/0"},
		{"$sourceDescriptions.petstore.get/pet.by.id", SourceDescriptions, "petstore", "", "get/pet.by.id", ""},
		{`$sourceDescriptions.petstore.get\u0020pet`, SourceDescriptions, "petstore", "", `get\u0020pet`, ""},
		{"$components.parameters.rate.limit", ComponentParameters, "rate.limit", "", "", ""},
		{"$components.successActions.retry", ComponentSuccessActions, "retry", "", "", ""},
		{"$components.failureActions.abort.now", ComponentFailureActions, "abort.now", "", "", ""},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			expr, err := Parse(test.input)
			require.NoError(t, err)
			assert.Equal(t, test.exprType, expr.Type)
			assert.Equal(t, test.input, expr.Raw)
			assert.Equal(t, test.name, expr.Name)
			assert.Equal(t, test.property, expr.Property)
			assert.Equal(t, test.tail, expr.Tail)
			assert.Equal(t, test.jsonPointer, expr.JSONPointer)
		})
	}
}

func TestParseArazzo11ConformanceBoundaries(t *testing.T) {
	tests := []string{
		"$self.value",
		"$message.header.",
		"$message.header.Bad Header",
		"$message.payload#not-a-pointer",
		"$request.query.{bad}",
		"$request.path.\"bad\"",
		"$response.payload#/bad~2escape",
		"$response.body#/bad{brace}",
		"$inputs.bad+name",
		"$inputs.value#relative",
		"$outputs.\xff",
		"$steps.step.with-dot.outputs.value",
		"$steps.step.outputs.bad+output",
		"$steps.step.outputs.value#relative",
		"$workflows.flow.with-dot.inputs.value",
		"$workflows.flow.parameters.value",
		"$workflows.flow.outputs.bad+name",
		"$sourceDescriptions.bad+name.operation",
		"$sourceDescriptions.source.",
		"$sourceDescriptions.source.bad{reference}",
		`$sourceDescriptions.source.bad"reference`,
		`$sourceDescriptions.source.bad\`,
		`$sourceDescriptions.source.bad\q`,
		`$sourceDescriptions.source.bad\u123`,
		`$sourceDescriptions.source.bad\u12xz`,
		"$sourceDescriptions.source.bad\x01reference",
		"$components.inputs.value",
		"$components.parameters.bad+name",
		"$components.successActions.",
	}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := Parse(input)
			assert.Error(t, err)
		})
	}
}

func TestParseArazzo11HelpersBoundaries(t *testing.T) {
	assert.Error(t, validateCharSequence("", false))
	assert.NoError(t, validateCharSequence("", true))
	assert.NoError(t, validateCharSequence(`escaped\/value`, false))
	assert.Error(t, validateCharSequence("\xff", false))
	assert.Error(t, validateJSONPointer("/\xff"))
	assert.Error(t, validateJSONPointer("relative"))
	assert.NoError(t, validateJSONPointer("/valid/~0/~1"))
	assert.Error(t, validateJSONPointer("/trailing~"))
	assert.Error(t, validateJSONPointer("/brace}"))
	assert.False(t, isIdentifier("é", false))
	assert.False(t, isHex4("123"))
	assert.False(t, isHex4("12xz"))
	assert.True(t, isHex4("09aF"))
}

func TestParseEmbeddedArazzo11Boundaries(t *testing.T) {
	_, err := ParseEmbedded("\xff")
	assert.Error(t, err)

	_, err = ParseEmbedded("literal}")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected closing brace")

	_, err = ParseEmbedded("literal} then {$self}")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected closing brace")
}

func TestValidateHasZeroAllocationsOnValidExpression(t *testing.T) {
	allocations := testing.AllocsPerRun(1000, func() {
		if err := Validate("$steps.create.outputs.order#/id"); err != nil {
			panic(err)
		}
	})
	assert.Equal(t, float64(0), allocations)
}

func TestResolveLocalArazzo11(t *testing.T) {
	symbols := &LocalSymbols{
		HasSelf:                 true,
		Inputs:                  SymbolSet{"input": {}, "exact.nested": {}},
		Outputs:                 SymbolSet{"output": {}},
		Steps:                   map[string]StepSymbols{"step": {Outputs: SymbolSet{"result": {}}}},
		Workflows:               map[string]WorkflowSymbols{"flow": {Inputs: SymbolSet{"input": {}}, Outputs: SymbolSet{"output": {}}}},
		SourceDescriptions:      SymbolSet{"source": {}},
		ComponentParameters:     SymbolSet{"parameter": {}},
		ComponentSuccessActions: SymbolSet{"success": {}},
		ComponentFailureActions: SymbolSet{"failure": {}},
	}
	valid := []string{
		"$self",
		"$inputs.input",
		"$inputs.input.child",
		"$inputs.exact.nested",
		"$outputs.output",
		"$outputs.output.child",
		"$steps.step.outputs.result",
		"$steps.step.outputs.result.child",
		"$workflows.flow.inputs.input",
		"$workflows.flow.inputs.input.child",
		"$workflows.flow.outputs.output",
		"$workflows.flow.outputs.output.child",
		"$sourceDescriptions.source.operation",
		"$components.parameters.parameter",
		"$components.successActions.success",
		"$components.failureActions.failure",
		"$response.body#/id",
	}
	for _, input := range valid {
		assert.NoError(t, ValidateLocal(input, symbols), input)
	}

	assert.Error(t, ResolveLocal(Expression{}, nil))
	assert.Error(t, ValidateLocal("not-an-expression", symbols))

	missing := []Expression{
		{Raw: "$self", Type: Self},
		{Raw: "$inputs.missing", Type: Inputs, Name: "missing"},
		{Raw: "$outputs.missing", Type: Outputs, Name: "missing"},
		{Raw: "$steps.missing.outputs.result", Type: Steps, Name: "missing", Tail: "outputs.result"},
		{Raw: "$steps.step", Type: Steps, Name: "step"},
		{Raw: "$steps.step.inputs.value", Type: Steps, Name: "step", Tail: "inputs.value"},
		{Raw: "$steps.step.outputs.missing", Type: Steps, Name: "step", Tail: "outputs.missing"},
		{Raw: "$workflows.missing.inputs.input", Type: Workflows, Name: "missing", Tail: "inputs.input"},
		{Raw: "$workflows.flow", Type: Workflows, Name: "flow"},
		{Raw: "$workflows.flow.parameters.input", Type: Workflows, Name: "flow", Tail: "parameters.input"},
		{Raw: "$workflows.flow.inputs.missing", Type: Workflows, Name: "flow", Tail: "inputs.missing"},
		{Raw: "$workflows.flow.outputs.missing", Type: Workflows, Name: "flow", Tail: "outputs.missing"},
		{Raw: "$sourceDescriptions.missing.op", Type: SourceDescriptions, Name: "missing"},
		{Raw: "$components.parameters.missing", Type: ComponentParameters, Name: "missing"},
		{Raw: "$components.successActions.missing", Type: ComponentSuccessActions, Name: "missing"},
		{Raw: "$components.failureActions.missing", Type: ComponentFailureActions, Name: "missing"},
	}
	missingSymbols := *symbols
	missingSymbols.HasSelf = false
	for _, expr := range missing {
		err := ResolveLocal(expr, &missingSymbols)
		require.Error(t, err, expr.Raw)
		assert.True(t, errors.Is(err, ErrLocalSymbolNotFound))
		var symbolErr *LocalSymbolError
		require.True(t, errors.As(err, &symbolErr))
		assert.NotEmpty(t, symbolErr.Error())
		assert.Equal(t, ErrLocalSymbolNotFound, symbolErr.Unwrap())
	}

	assert.Equal(t, []any{"", "", false}, tailParts("missing"))
	assert.Equal(t, []any{"", "", false}, tailParts("outputs."))
	assert.Equal(t, []any{"outputs", "value", true}, tailParts("outputs.value"))
	assert.False(t, hasSymbolOrPrefix(SymbolSet{"other": {}}, "missing.child"))
}

func tailParts(tail string) []any {
	field, name, ok := splitReferenceTail(tail)
	return []any{field, name, ok}
}

func FuzzParseArazzo11NeverPanics(f *testing.F) {
	for _, seed := range []string{
		"$self",
		"$message.payload#/id",
		"$steps.step.outputs.value#/0",
		"https://{$inputs.host}/{$outputs.path}",
		"{",
		"}",
		"\xff",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		_, _ = Parse(input)
		_, _ = ParseEmbedded(input)
	})
}

func BenchmarkValidateRuntimeExpression(b *testing.B) {
	for b.Loop() {
		_ = Validate("$steps.create.outputs.order#/items/0/id")
	}
}

func BenchmarkParseEmbeddedRuntimeExpressions(b *testing.B) {
	input := strings.Repeat("prefix {$inputs.value} suffix ", 32)
	b.ResetTimer()
	for b.Loop() {
		_, _ = ParseEmbedded(input)
	}
}
