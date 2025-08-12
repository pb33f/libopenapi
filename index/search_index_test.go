// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"context"
	"os"
	"testing"

	"github.com/pkg-base/yaml"
	"github.com/stretchr/testify/assert"
)

func TestSpecIndex_SearchIndexForReference(t *testing.T) {
	petstore, _ := os.ReadFile("../test_specs/petstorev3.json")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(petstore, &rootNode)

	c := CreateOpenAPIIndexConfig()
	idx := NewSpecIndexWithConfig(&rootNode, c)

	ref, _ := idx.SearchIndexForReference("#/components/schemas/Pet")
	assert.NotNil(t, ref)
}

func TestSpecIndex_SearchIndexForReferenceWithContext(t *testing.T) {
	petstore, _ := os.ReadFile("../test_specs/petstorev3.json")
	var rootNode yaml.Node
	_ = yaml.Unmarshal(petstore, &rootNode)

	c := CreateOpenAPIIndexConfig()
	idx := NewSpecIndexWithConfig(&rootNode, c)

	ref, _, _ := idx.SearchIndexForReferenceWithContext(context.Background(), "#/components/schemas/Pet")
	assert.NotNil(t, ref)

	assert.NotNil(t, idx.GetRootNode())
	idx.SetRootNode(nil)
	assert.Nil(t, idx.GetRootNode())

}
