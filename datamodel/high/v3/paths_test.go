// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	v3low "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestPaths_MarshalYAML(t *testing.T) {

	yml := `/foo/bar/bizzle:
    get:
        description: get a bizzle
/jim/jam/jizzle:
    post:
        description: post a jizzle
/beer:
    get:
        description: get a beer now.`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())

	var n v3low.Paths
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.NoError(t, err)

	high := NewPaths(&n)
	assert.NotNil(t, high)

	rend, _ := high.Render()
	assert.Equal(t, yml, strings.TrimSpace(string(rend)))

	// mutate
	deprecated := true
	high.PathItems["/beer"].Get.Deprecated = &deprecated

	yml = `/foo/bar/bizzle:
    get:
        description: get a bizzle
/jim/jam/jizzle:
    post:
        description: post a jizzle
/beer:
    get:
        description: get a beer now.
        deprecated: true`

	rend, _ = high.Render()
	assert.Equal(t, yml, strings.TrimSpace(string(rend)))

}

func TestPaths_MarshalYAMLInline(t *testing.T) {

	yml := `/foo/bar/bizzle:
    get:
        description: get a bizzle
/jim/jam/jizzle:
    post:
        description: post a jizzle
/beer:
    get:
        description: get a beer now.`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &rootNode)

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())

	var n v3low.Paths
	err := low.BuildModel(&idxNode, &n)
	assert.NoError(t, err)

	err = n.Build(idxNode.Content[0], idx)
	assert.NoError(t, err)

	high := NewPaths(&n)
	assert.NotNil(t, high)

	rend, _ := high.Render()
	assert.Equal(t, yml, strings.TrimSpace(string(rend)))

	// mutate
	deprecated := true
	high.PathItems["/beer"].Get.Deprecated = &deprecated

	yml = `/foo/bar/bizzle:
    get:
        description: get a bizzle
/jim/jam/jizzle:
    post:
        description: post a jizzle
/beer:
    get:
        description: get a beer now.
        deprecated: true`

	rend, _ = high.RenderInline()
	assert.Equal(t, yml, strings.TrimSpace(string(rend)))

}
