// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"testing"

	"github.com/pb33f/libopenapi/arazzo/expression"
	high "github.com/pb33f/libopenapi/datamodel/high/arazzo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
