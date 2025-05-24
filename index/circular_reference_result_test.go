// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCircularReferenceResult_GenerateJourneyPath(t *testing.T) {
	refs := []*Reference{
		{Name: "chicken"},
		{Name: "nuggets"},
		{Name: "chicken"},
		{Name: "soup"},
		{Name: "chicken"},
		{Name: "nuggets"},
		{Name: "for"},
		{Name: "me"},
		{Name: "and"},
		{Name: "you"},
	}

	cr := &CircularReferenceResult{Journey: refs}
	assert.Equal(t, "chicken -> nuggets -> chicken -> soup -> "+
		"chicken -> nuggets -> for -> me -> and -> you", cr.GenerateJourneyPath())
}
