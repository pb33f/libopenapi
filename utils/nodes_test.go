// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCreateBoolNode(t *testing.T) {
	b := CreateBoolNode("true")
	assert.Equal(t, "!!bool", b.Tag)
	assert.Equal(t, "true", b.Value)
}

func TestCreateEmptyMapNode(t *testing.T) {
	m := CreateEmptyMapNode()
	assert.Equal(t, "!!map", m.Tag)
	assert.Len(t, m.Content, 0)
}

func TestCreateEmptySequenceNode(t *testing.T) {
	s := CreateEmptySequenceNode()
	assert.Equal(t, "!!seq", s.Tag)
	assert.Len(t, s.Content, 0)
}

func TestCreateFloatNode(t *testing.T) {
	f := CreateFloatNode("3.14")
	assert.Equal(t, "!!float", f.Tag)
	assert.Equal(t, "3.14", f.Value)
}

func TestCreateIntNode(t *testing.T) {
	i := CreateIntNode("42")
	assert.Equal(t, "!!int", i.Tag)
	assert.Equal(t, "42", i.Value)
}

func TestCreateRefNode(t *testing.T) {
	r := CreateRefNode("#/components/schemas/MySchema")
	assert.Equal(t, "!!map", r.Tag)
	assert.Len(t, r.Content, 2)
	assert.Equal(t, "!!str", r.Content[0].Tag)
	assert.Equal(t, "$ref", r.Content[0].Value)
	assert.Equal(t, "!!str", r.Content[1].Tag)
	assert.Equal(t, "#/components/schemas/MySchema", r.Content[1].Value)
}
