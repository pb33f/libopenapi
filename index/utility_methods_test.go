// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"net/url"
	"testing"
)

func TestGenerateCleanSpecConfigBaseURL(t *testing.T) {
	u, _ := url.Parse("https://pb33f.io/things/stuff")
	path := "."
	assert.Equal(t, "https://pb33f.io/things/stuff",
		GenerateCleanSpecConfigBaseURL(u, path, false))
}

func TestGenerateCleanSpecConfigBaseURL_RelativeDeep(t *testing.T) {
	u, _ := url.Parse("https://pb33f.io/things/stuff/jazz/cakes/winter/oil")
	path := "../../../../foo/bar/baz/crap.yaml#thang"
	assert.Equal(t, "https://pb33f.io/things/stuff/foo/bar/baz",
		GenerateCleanSpecConfigBaseURL(u, path, false))

	assert.Equal(t, "https://pb33f.io/things/stuff/foo/bar/baz/crap.yaml#thang",
		GenerateCleanSpecConfigBaseURL(u, path, true))
}

func TestGenerateCleanSpecConfigBaseURL_NoBaseURL(t *testing.T) {

	u, _ := url.Parse("/things/stuff/jazz/cakes/winter/oil")
	path := "../../../../foo/bar/baz/crap.yaml#thang"
	assert.Equal(t, "/things/stuff/foo/bar/baz",
		GenerateCleanSpecConfigBaseURL(u, path, false))

	assert.Equal(t, "/things/stuff/foo/bar/baz/crap.yaml#thang",
		GenerateCleanSpecConfigBaseURL(u, path, true))
}

func TestGenerateCleanSpecConfigBaseURL_HttpStrip(t *testing.T) {

	u, _ := url.Parse(".")
	path := "http://thing.com/crap.yaml#thang"
	assert.Equal(t, "http://thing.com",
		GenerateCleanSpecConfigBaseURL(u, path, false))

	assert.Equal(t, "",
		GenerateCleanSpecConfigBaseURL(u, "crap.yaml#thing", true))
}

func Test_extractRequiredReferenceProperties(t *testing.T) {

	d := `$ref: http://internets/shoes`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)
	props := make(map[string][]string)

	data := extractRequiredReferenceProperties("the-big.yaml#/cheese/thing",
		rootNode.Content[0], "cakes", props)
	assert.Len(t, props, 1)
	assert.NotNil(t, data)
}

func Test_extractRequiredReferenceProperties_singleFile(t *testing.T) {

	d := `$ref: http://cake.yaml/camel.yaml`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)
	props := make(map[string][]string)

	data := extractRequiredReferenceProperties("dingo-bingo-bango.yaml",
		rootNode.Content[0], "cakes", props)
	assert.Len(t, props, 1)
	assert.NotNil(t, data)
}

func Test_extractRequiredReferenceProperties_http(t *testing.T) {

	d := `$ref: http://cake.yaml/camel.yaml`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)
	props := make(map[string][]string)

	data := extractRequiredReferenceProperties("http://dingo-bingo-bango.yaml/camel.yaml",
		rootNode.Content[0], "cakes", props)
	assert.Len(t, props, 1)
	assert.NotNil(t, data)
}

func Test_extractRequiredReferenceProperties_abs(t *testing.T) {

	d := `$ref: http://cake.yaml/camel.yaml`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)
	props := make(map[string][]string)

	data := extractRequiredReferenceProperties("/camel.yaml",
		rootNode.Content[0], "cakes", props)
	assert.Len(t, props, 1)
	assert.NotNil(t, data)
}

func Test_extractDefinitionRequiredRefProperties_nil(t *testing.T) {
	assert.Nil(t, extractDefinitionRequiredRefProperties(nil, nil, ""))
}
