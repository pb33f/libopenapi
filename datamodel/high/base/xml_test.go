// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"fmt"
	"strings"
	"testing"

	lowmodel "github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func ExampleNewXML() {
	// create an example schema object
	// this can be either JSON or YAML.
	yml := `
namespace: https://pb33f.io/schema
name: something
attribute: true
prefix: sample
wrapped: true`

	// unmarshal raw bytes
	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	// build out the low-level model
	var lowXML lowbase.XML
	_ = lowmodel.BuildModel(node.Content[0], &lowXML)
	_ = lowXML.Build(node.Content[0], nil)

	// build the high level tag
	highXML := NewXML(&lowXML)

	// print out the XML namespace
	fmt.Print(highXML.Namespace)
	// Output: https://pb33f.io/schema
}

func TestContact_Render(t *testing.T) {
	// create an example schema object
	// this can be either JSON or YAML.
	yml := `namespace: https://pb33f.io/schema
name: something
attribute: true
prefix: sample
wrapped: true`

	// unmarshal raw bytes
	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	// build out the low-level model
	var lowXML lowbase.XML
	_ = lowmodel.BuildModel(node.Content[0], &lowXML)
	_ = lowXML.Build(node.Content[0], nil)

	// build the high level tag
	highXML := NewXML(&lowXML)

	// print out the XML doc
	highXMLBytes, _ := highXML.Render()
	assert.Equal(t, yml, strings.TrimSpace(string(highXMLBytes)))

	highXML.Attribute = false
	highXMLBytes, _ = highXML.Render()
	assert.NotEqual(t, yml, strings.TrimSpace(string(highXMLBytes)))
}
