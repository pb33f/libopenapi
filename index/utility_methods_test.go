// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"net/url"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func Test_seekRefEnd(t *testing.T) {

	d := `cob: wob
rob:
  $ref: "#/cob"
bill:
  $ref: "#/rob"`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)
	idx := NewSpecIndex(&rootNode)
	r := seekRefEnd(idx, "#/rob")
	assert.NotNil(t, r)

}

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

func Test_extractRequiredReferenceProperties_abs3(t *testing.T) {
	d := `$ref: oh/pillow.yaml`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)
	props := make(map[string][]string)

	data := extractRequiredReferenceProperties("/big/fat/camel.yaml#/milk",
		rootNode.Content[0], "cakes", props)
	assert.Len(t, props, 1)
	if runtime.GOOS != "windows" {
		assert.Equal(t, "cakes", props["/big/fat/oh/pillow.yaml"][0])
	}
	assert.NotNil(t, data)
}

func Test_extractRequiredReferenceProperties_rel_full(t *testing.T) {
	d := `$ref: "#/a/nice/picture/of/cake"`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)
	props := make(map[string][]string)

	data := extractRequiredReferenceProperties("/chalky/milky/camel.yaml#/milk",
		rootNode.Content[0], "cakes", props)
	assert.Len(t, props, 1)
	if runtime.GOOS != "windows" {
		assert.Equal(t, "cakes", props["/chalky/milky/camel.yaml#/a/nice/picture/of/cake"][0])
	}
	assert.NotNil(t, data)
}

func Test_extractRequiredReferenceProperties_rel(t *testing.T) {
	d := `$ref: oh/camel.yaml#/rum/cake`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)
	props := make(map[string][]string)

	data := extractRequiredReferenceProperties("/camel.yaml#/milk",
		rootNode.Content[0], "cakes", props)
	assert.Len(t, props, 1)
	if runtime.GOOS != "windows" {
		assert.Equal(t, "cakes", props["/oh/camel.yaml#/rum/cake"][0])
	}
	assert.NotNil(t, data)
}

func Test_extractRequiredReferenceProperties_abs2(t *testing.T) {
	d := `$ref: /oh/my/camel.yaml#/rum/cake`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)
	props := make(map[string][]string)

	data := extractRequiredReferenceProperties("../flannel.yaml#/milk",
		rootNode.Content[0], "cakes", props)
	assert.Len(t, props, 1)
	if runtime.GOOS != "windows" {
		assert.Equal(t, "cakes", props["/oh/my/camel.yaml#/rum/cake"][0])
	}
	assert.NotNil(t, data)
}

func Test_extractRequiredReferenceProperties_http_rel(t *testing.T) {
	d := `$ref: my/wet/camel.yaml#/rum/cake`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)
	props := make(map[string][]string)

	data := extractRequiredReferenceProperties("http://beer-world.com/lost/in/space.yaml#/vase",
		rootNode.Content[0], "cakes", props)
	assert.Len(t, props, 1)
	assert.Equal(t, "cakes", props["http://beer-world.com/lost/in/my/wet/camel.yaml#/rum/cake"][0])
	assert.NotNil(t, data)
}

func Test_extractRequiredReferenceProperties_http_rel_nocomponent(t *testing.T) {
	d := `$ref: my/wet/camel.yaml`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)
	props := make(map[string][]string)

	data := extractRequiredReferenceProperties("http://beer-world.com/lost/in/space.yaml#/vase",
		rootNode.Content[0], "cakes", props)
	assert.Len(t, props, 1)
	assert.Equal(t, "cakes", props["http://beer-world.com/lost/in/my/wet/camel.yaml"][0])
	assert.NotNil(t, data)
}

func Test_extractRequiredReferenceProperties_nocomponent(t *testing.T) {
	d := `$ref: my/wet/camel.yaml`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)
	props := make(map[string][]string)

	data := extractRequiredReferenceProperties("#/rotund/cakes",
		rootNode.Content[0], "cakes", props)
	assert.Len(t, props, 1)
	assert.Equal(t, "cakes", props["my/wet/camel.yaml"][0])
	assert.NotNil(t, data)
}

func Test_extractRequiredReferenceProperties_component_http(t *testing.T) {
	d := `$ref: go-to-bed.com/no/more/cake.yaml#/lovely/jam`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)
	props := make(map[string][]string)

	data := extractRequiredReferenceProperties("http://bunny-bun-bun.com/no.yaml",
		rootNode.Content[0], "cakes", props)
	assert.Len(t, props, 1)
	assert.Equal(t, "cakes", props["http://bunny-bun-bun.com/go-to-bed.com/no/more/cake.yaml#/lovely/jam"][0])
	assert.NotNil(t, data)
}

func Test_extractRequiredReferenceProperties_nocomponent_http(t *testing.T) {
	d := `$ref: go-to-bed.com/no/more/cake.yaml`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)
	props := make(map[string][]string)

	data := extractRequiredReferenceProperties("http://bunny-bun-bun.com/no.yaml",
		rootNode.Content[0], "cakes", props)
	assert.Len(t, props, 1)
	assert.Equal(t, "cakes", props["http://bunny-bun-bun.com/go-to-bed.com/no/more/cake.yaml"][0])
	assert.NotNil(t, data)
}

func Test_extractRequiredReferenceProperties_nocomponent_http2(t *testing.T) {
	d := `$ref: go-to-bed.com/no/more/cake.yaml`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)
	props := make(map[string][]string)

	data := extractRequiredReferenceProperties("/why.yaml",
		rootNode.Content[0], "cakes", props)
	assert.Len(t, props, 1)
	if runtime.GOOS != "windows" {
		assert.Equal(t, "cakes", props["/go-to-bed.com/no/more/cake.yaml"][0])
	}
	assert.NotNil(t, data)
}

func Test_extractDefinitionRequiredRefProperties_nil(t *testing.T) {
	assert.Nil(t, extractDefinitionRequiredRefProperties(nil, nil, "", nil))
}

func TestSyncMapToMap_Nil(t *testing.T) {
	assert.Nil(t, syncMapToMap[string, string](nil))
}

func Test_HashNode(t *testing.T) {
	d := `plum: soup
chicken: wing
beef: burger
pork: chop`
	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(d), &rootNode)

	hash := HashNode(&rootNode)
	assert.NotEmpty(t, hash)
	assert.Equal(t, "e9aba1ce94ac8bd0143524ce594c0c7d38c06c09eca7ae96725187f488661fcd", hash)
}

func Test_HashNode_TooDeep(t *testing.T) {

	nodeA := &yaml.Node{}
	nodeB := &yaml.Node{}

	// create an infinite loop.
	nodeA.Content = append(nodeA.Content, nodeB)
	nodeB.Content = append(nodeB.Content, nodeA)

	hash := HashNode(nodeA)
	assert.NotEmpty(t, hash)
	assert.Equal(t, "e6d506f4b5a87b3f37ac8bed41c8411a5883b5f20d141d45ee92245c023a73e4", hash)
}
