// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	lowmodel "github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"strings"
	"testing"
)

func TestNewSecurityRequirement(t *testing.T) {

	var cNode yaml.Node

	yml := `pizza:
    - cheese
    - tomato`

	_ = yaml.Unmarshal([]byte(yml), &cNode)

	var lowExt lowbase.SecurityRequirement
	_ = lowmodel.BuildModel(cNode.Content[0], &lowExt)

	_ = lowExt.Build(cNode.Content[0], nil)

	highExt := NewSecurityRequirement(&lowExt)

	assert.Len(t, highExt.Requirements["pizza"], 2)

	wentLow := highExt.GoLow()
	assert.Len(t, wentLow.Requirements.Value, 1)
	assert.NotNil(t, highExt.GoLowUntyped())

	// render the high-level object as YAML
	highBytes, _ := highExt.Render()
	assert.Equal(t, strings.TrimSpace(string(highBytes)), yml)
}
