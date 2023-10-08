// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSpecIndex_Children(t *testing.T) {
	idx1 := new(SpecIndex)
	idx2 := new(SpecIndex)
	idx3 := new(SpecIndex)
	idx4 := new(SpecIndex)
	idx5 := new(SpecIndex)
	idx1.AddChild(idx2)
	idx1.AddChild(idx3)
	idx3.AddChild(idx4)
	idx4.AddChild(idx5)
	assert.Equal(t, 2, len(idx1.GetChildren()))
	assert.Equal(t, 1, len(idx3.GetChildren()))
	assert.Equal(t, 1, len(idx4.GetChildren()))
	assert.Equal(t, 0, len(idx5.GetChildren()))
}

func TestSpecIndex_GetConfig(t *testing.T) {
	idx1 := new(SpecIndex)
	c := SpecIndexConfig{}
	idx1.config = &c
	assert.Equal(t, &c, idx1.GetConfig())
}
