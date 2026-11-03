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
