// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package expression

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// Parse() -- every ExpressionType
// ---------------------------------------------------------------------------

func TestParse_URL(t *testing.T) {
	expr, err := Parse("$url")
	assert.NoError(t, err)
	assert.Equal(t, URL, expr.Type)
	assert.Equal(t, "$url", expr.Raw)
}

func TestParse_Method(t *testing.T) {
	expr, err := Parse("$method")
	assert.NoError(t, err)
	assert.Equal(t, Method, expr.Type)
	assert.Equal(t, "$method", expr.Raw)
}

func TestParse_StatusCode(t *testing.T) {
	expr, err := Parse("$statusCode")
	assert.NoError(t, err)
	assert.Equal(t, StatusCode, expr.Type)
	assert.Equal(t, "$statusCode", expr.Raw)
}

func TestParse_RequestHeader(t *testing.T) {
	expr, err := Parse("$request.header.X-Api-Key")
	assert.NoError(t, err)
	assert.Equal(t, RequestHeader, expr.Type)
	assert.Equal(t, "X-Api-Key", expr.Property)
}

func TestParse_RequestQuery(t *testing.T) {
	expr, err := Parse("$request.query.page")
	assert.NoError(t, err)
	assert.Equal(t, RequestQuery, expr.Type)
	assert.Equal(t, "page", expr.Property)
}

func TestParse_RequestPath(t *testing.T) {
	expr, err := Parse("$request.path.petId")
	assert.NoError(t, err)
	assert.Equal(t, RequestPath, expr.Type)
	assert.Equal(t, "petId", expr.Property)
}

func TestParse_RequestBody_NoPointer(t *testing.T) {
	expr, err := Parse("$request.body")
	assert.NoError(t, err)
	assert.Equal(t, RequestBody, expr.Type)
	assert.Empty(t, expr.JSONPointer)
}

func TestParse_RequestBody_WithPointer(t *testing.T) {
	expr, err := Parse("$request.body#/name")
	assert.NoError(t, err)
	assert.Equal(t, RequestBody, expr.Type)
	assert.Equal(t, "/name", expr.JSONPointer)
}

func TestParse_RequestBody_DeepPointer(t *testing.T) {
	expr, err := Parse("$request.body#/data/0/id")
	assert.NoError(t, err)
	assert.Equal(t, RequestBody, expr.Type)
	assert.Equal(t, "/data/0/id", expr.JSONPointer)
}

func TestParse_ResponseHeader(t *testing.T) {
	expr, err := Parse("$response.header.Content-Type")
	assert.NoError(t, err)
	assert.Equal(t, ResponseHeader, expr.Type)
	assert.Equal(t, "Content-Type", expr.Property)
}

func TestParse_ResponseQuery(t *testing.T) {
	expr, err := Parse("$response.query.token")
	assert.NoError(t, err)
	assert.Equal(t, ResponseQuery, expr.Type)
	assert.Equal(t, "token", expr.Property)
}

func TestParse_ResponsePath(t *testing.T) {
	expr, err := Parse("$response.path.userId")
	assert.NoError(t, err)
	assert.Equal(t, ResponsePath, expr.Type)
	assert.Equal(t, "userId", expr.Property)
}

func TestParse_ResponseBody_WithPointer(t *testing.T) {
	expr, err := Parse("$response.body#/results/0")
	assert.NoError(t, err)
	assert.Equal(t, ResponseBody, expr.Type)
	assert.Equal(t, "/results/0", expr.JSONPointer)
}

func TestParse_ResponseBody_NoPointer(t *testing.T) {
	expr, err := Parse("$response.body")
	assert.NoError(t, err)
	assert.Equal(t, ResponseBody, expr.Type)
	assert.Empty(t, expr.JSONPointer)
}

func TestParse_Inputs(t *testing.T) {
	expr, err := Parse("$inputs.petId")
	assert.NoError(t, err)
	assert.Equal(t, Inputs, expr.Type)
	assert.Equal(t, "petId", expr.Name)
}

func TestParse_Outputs(t *testing.T) {
	expr, err := Parse("$outputs.result")
	assert.NoError(t, err)
	assert.Equal(t, Outputs, expr.Type)
	assert.Equal(t, "result", expr.Name)
}

func TestParse_Steps_WithTail(t *testing.T) {
	expr, err := Parse("$steps.getPet.outputs.petId")
	assert.NoError(t, err)
	assert.Equal(t, Steps, expr.Type)
	assert.Equal(t, "getPet", expr.Name)
	assert.Equal(t, "outputs.petId", expr.Tail)
}

func TestParse_Steps_NoTail(t *testing.T) {
	expr, err := Parse("$steps.myStep")
	assert.NoError(t, err)
	assert.Equal(t, Steps, expr.Type)
	assert.Equal(t, "myStep", expr.Name)
	assert.Empty(t, expr.Tail)
}

func TestParse_Workflows(t *testing.T) {
	expr, err := Parse("$workflows.getUser.outputs.name")
	assert.NoError(t, err)
	assert.Equal(t, Workflows, expr.Type)
	assert.Equal(t, "getUser", expr.Name)
	assert.Equal(t, "outputs.name", expr.Tail)
}

func TestParse_Workflows_NoTail(t *testing.T) {
	expr, err := Parse("$workflows.myFlow")
	assert.NoError(t, err)
	assert.Equal(t, Workflows, expr.Type)
	assert.Equal(t, "myFlow", expr.Name)
	assert.Empty(t, expr.Tail)
}

func TestParse_SourceDescriptions(t *testing.T) {
	expr, err := Parse("$sourceDescriptions.petStore.url")
	assert.NoError(t, err)
	assert.Equal(t, SourceDescriptions, expr.Type)
	assert.Equal(t, "petStore", expr.Name)
	assert.Equal(t, "url", expr.Tail)
}

func TestParse_SourceDescriptions_NoTail(t *testing.T) {
	expr, err := Parse("$sourceDescriptions.petStore")
	assert.NoError(t, err)
	assert.Equal(t, SourceDescriptions, expr.Type)
	assert.Equal(t, "petStore", expr.Name)
	assert.Empty(t, expr.Tail)
}

func TestParse_ComponentParameters(t *testing.T) {
	expr, err := Parse("$components.parameters.myParam")
	assert.NoError(t, err)
	assert.Equal(t, ComponentParameters, expr.Type)
	assert.Equal(t, "myParam", expr.Name)
}

func TestParse_Components_General(t *testing.T) {
	expr, err := Parse("$components.inputs.someInput")
	assert.NoError(t, err)
	assert.Equal(t, Components, expr.Type)
	assert.Equal(t, "inputs", expr.Name)
	assert.Equal(t, "someInput", expr.Tail)
}

func TestParse_Components_SuccessActions(t *testing.T) {
	expr, err := Parse("$components.successActions.retry")
	assert.NoError(t, err)
	assert.Equal(t, Components, expr.Type)
	assert.Equal(t, "successActions", expr.Name)
	assert.Equal(t, "retry", expr.Tail)
}

func TestParse_Components_NoTail(t *testing.T) {
	expr, err := Parse("$components.schemas")
	assert.NoError(t, err)
	assert.Equal(t, Components, expr.Type)
	assert.Equal(t, "schemas", expr.Name)
	assert.Empty(t, expr.Tail)
}

// ---------------------------------------------------------------------------
// Parse() -- error cases
// ---------------------------------------------------------------------------

func TestParse_Error_Empty(t *testing.T) {
	_, err := Parse("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty expression")
}

func TestParse_Error_NoDollar(t *testing.T) {
	_, err := Parse("url")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must start with '$'")
}

func TestParse_Error_JustDollar(t *testing.T) {
	_, err := Parse("$")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "incomplete expression")
}

func TestParse_Error_UnknownPrefix(t *testing.T) {
	_, err := Parse("$x")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown expression prefix")
}

func TestParse_Error_IncompleteRequest(t *testing.T) {
	_, err := Parse("$request.")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "incomplete source expression")
}

func TestParse_Error_IncompleteResponse(t *testing.T) {
	_, err := Parse("$response.")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "incomplete source expression")
}

func TestParse_RequestBody_EmptyPointer(t *testing.T) {
	// $request.body# has an empty pointer string after the #
	expr, err := Parse("$request.body#")
	assert.NoError(t, err)
	assert.Equal(t, RequestBody, expr.Type)
	assert.Empty(t, expr.JSONPointer)
}

func TestParse_Error_EmptyInputsName(t *testing.T) {
	_, err := Parse("$inputs.")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty name")
}

func TestParse_Error_EmptyOutputsName(t *testing.T) {
	_, err := Parse("$outputs.")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty name")
}

func TestParse_Error_EmptyStepsName(t *testing.T) {
	_, err := Parse("$steps.")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty name")
}

func TestParse_Error_EmptyWorkflowsName(t *testing.T) {
	_, err := Parse("$workflows.")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty name")
}

func TestParse_Error_EmptySourceDescriptionsName(t *testing.T) {
	_, err := Parse("$sourceDescriptions.")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty name")
}

func TestParse_Error_EmptyNamedIdentifier(t *testing.T) {
	cases := []string{
		"$steps..outputs.id",
		"$workflows..outputs.id",
		"$sourceDescriptions..url",
	}

	for _, tc := range cases {
		_, err := Parse(tc)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty name")
	}
}

func TestParse_Error_EmptyComponentsName(t *testing.T) {
	_, err := Parse("$components.")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty name")
}

func TestParse_Error_EmptyComponentParametersName(t *testing.T) {
	_, err := Parse("$components.parameters.")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty parameter name")
}

func TestParse_Error_EmptyHeaderName(t *testing.T) {
	_, err := Parse("$request.header.")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty header name")
}

func TestParse_Error_EmptyQueryName(t *testing.T) {
	_, err := Parse("$request.query.")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty query name")
}

func TestParse_Error_EmptyPathName(t *testing.T) {
	_, err := Parse("$request.path.")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty path name")
}

func TestParse_Error_InvalidHeaderTchar(t *testing.T) {
	// Space is not a valid tchar
	_, err := Parse("$request.header.X Api Key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid character")
}

func TestParse_Error_InvalidHeaderTchar_Tab(t *testing.T) {
	_, err := Parse("$request.header.X\tKey")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid character")
}

func TestParse_Error_InvalidHeaderTchar_HighByte(t *testing.T) {
	// Bytes >= 128 are not valid tchars
	_, err := Parse("$request.header.X\x80Key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid character")
}

func TestParse_Error_UnknownUrl(t *testing.T) {
	_, err := Parse("$urls")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown expression")
}

func TestParse_Error_UnknownMethod(t *testing.T) {
	_, err := Parse("$methods")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown expression")
}

func TestParse_Error_UnknownStatusCode(t *testing.T) {
	_, err := Parse("$statusCodes")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown expression")
}

func TestParse_Error_UnknownInputs(t *testing.T) {
	_, err := Parse("$input.foo")
	assert.Error(t, err)
}

func TestParse_Error_UnknownOutputs(t *testing.T) {
	_, err := Parse("$output.foo")
	assert.Error(t, err)
}

func TestParse_Error_UnknownWorkflows(t *testing.T) {
	_, err := Parse("$workflow.foo")
	assert.Error(t, err)
}

func TestParse_Error_UnknownComponents(t *testing.T) {
	_, err := Parse("$component.foo")
	assert.Error(t, err)
}

func TestParse_Error_RequestNoSource(t *testing.T) {
	// "$request." followed by unrecognized source
	_, err := Parse("$request.cookie.foo")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown source type")
}

func TestParse_Error_ResponseNoSource(t *testing.T) {
	_, err := Parse("$response.cookie.bar")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown source type")
}

// ---------------------------------------------------------------------------
// tchar validation -- boundary characters
// ---------------------------------------------------------------------------

func TestTchar_ValidSpecials(t *testing.T) {
	// All special tchars: ! # $ % & ' * + - . ^ _ ` | ~
	specials := "!#$%&'*+-.^_`|~"
	for _, c := range specials {
		assert.True(t, isTchar(byte(c)), "expected %q to be a valid tchar", string(c))
	}
}

func TestTchar_ValidAlpha(t *testing.T) {
	for c := byte('a'); c <= 'z'; c++ {
		assert.True(t, isTchar(c))
	}
	for c := byte('A'); c <= 'Z'; c++ {
		assert.True(t, isTchar(c))
	}
}

func TestTchar_ValidDigit(t *testing.T) {
	for c := byte('0'); c <= '9'; c++ {
		assert.True(t, isTchar(c))
	}
}

func TestTchar_InvalidControls(t *testing.T) {
	// NUL, TAB, CR, LF, space
	for _, c := range []byte{0, 9, 10, 13, 32} {
		assert.False(t, isTchar(c), "expected %d to not be a valid tchar", c)
	}
}

func TestTchar_InvalidSeparators(t *testing.T) {
	// ( ) < > @ , ; : \ " / [ ] ? = { }
	for _, c := range "()<>@,;:\\\"/[]?={}" {
		assert.False(t, isTchar(byte(c)), "expected %q to not be a valid tchar", string(c))
	}
}

func TestTchar_HighByte(t *testing.T) {
	// Bytes >= 128 should return false
	assert.False(t, isTchar(128))
	assert.False(t, isTchar(255))
}

// ---------------------------------------------------------------------------
// Parse() -- header names with valid tchar special characters
// ---------------------------------------------------------------------------

func TestParse_RequestHeader_WithHyphen(t *testing.T) {
	expr, err := Parse("$request.header.X-Forwarded-For")
	assert.NoError(t, err)
	assert.Equal(t, RequestHeader, expr.Type)
	assert.Equal(t, "X-Forwarded-For", expr.Property)
}

func TestParse_RequestHeader_WithSpecialChars(t *testing.T) {
	expr, err := Parse("$request.header.X_Custom!Header")
	assert.NoError(t, err)
	assert.Equal(t, RequestHeader, expr.Type)
	assert.Equal(t, "X_Custom!Header", expr.Property)
}

func TestParse_ResponseHeader_Validation(t *testing.T) {
	// Space is not a valid tchar for response headers too
	_, err := Parse("$response.header.Bad Header")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid character")
}

// ---------------------------------------------------------------------------
// ParseEmbedded()
// ---------------------------------------------------------------------------

func TestParseEmbedded_PlainText(t *testing.T) {
	tokens, err := ParseEmbedded("plain text")
	assert.NoError(t, err)
	assert.Len(t, tokens, 1)
	assert.False(t, tokens[0].IsExpression)
	assert.Equal(t, "plain text", tokens[0].Literal)
}

func TestParseEmbedded_SingleExpression(t *testing.T) {
	tokens, err := ParseEmbedded("{$url}")
	assert.NoError(t, err)
	assert.Len(t, tokens, 1)
	assert.True(t, tokens[0].IsExpression)
	assert.Equal(t, URL, tokens[0].Expression.Type)
}

func TestParseEmbedded_Mixed(t *testing.T) {
	tokens, err := ParseEmbedded("ID: {$inputs.id} done")
	assert.NoError(t, err)
	assert.Len(t, tokens, 3)

	assert.False(t, tokens[0].IsExpression)
	assert.Equal(t, "ID: ", tokens[0].Literal)

	assert.True(t, tokens[1].IsExpression)
	assert.Equal(t, Inputs, tokens[1].Expression.Type)
	assert.Equal(t, "id", tokens[1].Expression.Name)

	assert.False(t, tokens[2].IsExpression)
	assert.Equal(t, " done", tokens[2].Literal)
}

func TestParseEmbedded_Multiple(t *testing.T) {
	tokens, err := ParseEmbedded("{$method} {$url}")
	assert.NoError(t, err)
	assert.Len(t, tokens, 3)

	assert.True(t, tokens[0].IsExpression)
	assert.Equal(t, Method, tokens[0].Expression.Type)

	assert.False(t, tokens[1].IsExpression)
	assert.Equal(t, " ", tokens[1].Literal)

	assert.True(t, tokens[2].IsExpression)
	assert.Equal(t, URL, tokens[2].Expression.Type)
}

func TestParseEmbedded_UnclosedBrace(t *testing.T) {
	_, err := ParseEmbedded("{$url")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unclosed expression brace")
}

func TestParseEmbedded_EmptyInput(t *testing.T) {
	tokens, err := ParseEmbedded("")
	assert.NoError(t, err)
	assert.Nil(t, tokens)
}

func TestParseEmbedded_InvalidExpressionInBraces(t *testing.T) {
	_, err := ParseEmbedded("{notAnExpression}")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid embedded expression")
}

func TestParseEmbedded_MultipleExpressionsMixed(t *testing.T) {
	tokens, err := ParseEmbedded("start {$method} middle {$statusCode} end")
	assert.NoError(t, err)
	assert.Len(t, tokens, 5)

	assert.False(t, tokens[0].IsExpression)
	assert.Equal(t, "start ", tokens[0].Literal)

	assert.True(t, tokens[1].IsExpression)
	assert.Equal(t, Method, tokens[1].Expression.Type)

	assert.False(t, tokens[2].IsExpression)
	assert.Equal(t, " middle ", tokens[2].Literal)

	assert.True(t, tokens[3].IsExpression)
	assert.Equal(t, StatusCode, tokens[3].Expression.Type)

	assert.False(t, tokens[4].IsExpression)
	assert.Equal(t, " end", tokens[4].Literal)
}

func TestParseEmbedded_OnlyLiteralNoBraces(t *testing.T) {
	tokens, err := ParseEmbedded("no expressions here at all")
	assert.NoError(t, err)
	assert.Len(t, tokens, 1)
	assert.False(t, tokens[0].IsExpression)
	assert.Equal(t, "no expressions here at all", tokens[0].Literal)
}

func TestParseEmbedded_AdjacentExpressions(t *testing.T) {
	tokens, err := ParseEmbedded("{$url}{$method}")
	assert.NoError(t, err)
	assert.Len(t, tokens, 2)
	assert.True(t, tokens[0].IsExpression)
	assert.Equal(t, URL, tokens[0].Expression.Type)
	assert.True(t, tokens[1].IsExpression)
	assert.Equal(t, Method, tokens[1].Expression.Type)
}

func TestParseEmbedded_BodyWithPointer(t *testing.T) {
	tokens, err := ParseEmbedded("body={$response.body#/id}")
	assert.NoError(t, err)
	assert.Len(t, tokens, 2)

	assert.False(t, tokens[0].IsExpression)
	assert.Equal(t, "body=", tokens[0].Literal)

	assert.True(t, tokens[1].IsExpression)
	assert.Equal(t, ResponseBody, tokens[1].Expression.Type)
	assert.Equal(t, "/id", tokens[1].Expression.JSONPointer)
}

// ---------------------------------------------------------------------------
// Validate()
// ---------------------------------------------------------------------------

func TestValidate_Valid(t *testing.T) {
	validExprs := []string{
		"$url",
		"$method",
		"$statusCode",
		"$request.header.Accept",
		"$request.query.limit",
		"$request.path.id",
		"$request.body",
		"$request.body#/name",
		"$response.header.Content-Type",
		"$response.body#/data",
		"$inputs.name",
		"$outputs.value",
		"$steps.step1",
		"$steps.step1.outputs.result",
		"$workflows.flow1",
		"$workflows.flow1.outputs.token",
		"$sourceDescriptions.petStore",
		"$sourceDescriptions.petStore.url",
		"$components.parameters.limit",
		"$components.inputs.someInput",
	}
	for _, v := range validExprs {
		assert.NoError(t, Validate(v), "expected %q to be valid", v)
	}
}

func TestValidate_Invalid(t *testing.T) {
	invalidExprs := []string{
		"",
		"url",
		"$",
		"$x",
		"$request.",
		"$inputs.",
		"$steps.",
	}
	for _, v := range invalidExprs {
		assert.Error(t, Validate(v), "expected %q to be invalid", v)
	}
}
