// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSpecIndex_GetConfig(t *testing.T) {
	idx1 := new(SpecIndex)
	c := SpecIndexConfig{}
	idx1.config = &c
	assert.Equal(t, &c, idx1.GetConfig())
}
