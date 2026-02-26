// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/stretchr/testify/assert"
)

func TestBuildSlice_EmptyReturnsNil(t *testing.T) {
	out := buildSlice[int, string](nil, func(v int) string { return "" })
	assert.Nil(t, out)
}

func TestBuildSlice_ConvertsValues(t *testing.T) {
	refs := []low.ValueReference[int]{
		{Value: 2},
		{Value: 3},
	}
	out := buildSlice(refs, func(v int) string { return string(rune('0' + v)) })
	assert.Equal(t, []string{"2", "3"}, out)
}

func TestBuildValueSlice_EmptyReturnsNil(t *testing.T) {
	out := buildValueSlice[string](nil)
	assert.Nil(t, out)
}

func TestBuildValueSlice_ExtractsValues(t *testing.T) {
	refs := []low.ValueReference[string]{
		{Value: "a"},
		{Value: "b"},
	}
	out := buildValueSlice(refs)
	assert.Equal(t, []string{"a", "b"}, out)
}
