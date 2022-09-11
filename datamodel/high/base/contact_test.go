// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	lowmodel "github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestNewContact(t *testing.T) {

	var cNode yaml.Node

	yml := `name: pizza
url: https://pb33f.io
email: buckaroo@pb33f.io`

	_ = yaml.Unmarshal([]byte(yml), &cNode)

	// build low
	var lowContact lowbase.Contact
	_ = lowmodel.BuildModel(&cNode, &lowContact)

	// build high
	highContact := NewContact(&lowContact)

	assert.Equal(t, "pizza", highContact.Name)
	assert.Equal(t, "https://pb33f.io", highContact.URL)
	assert.Equal(t, "buckaroo@pb33f.io", highContact.Email)
	assert.Equal(t, 1, highContact.GoLow().Name.KeyNode.Line)

}
