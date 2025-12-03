// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
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

// TestSearchIndexForReference_LastDitchRolodexFallback tests the last-ditch effort
// code path where a reference is found by iterating through rolodex indexes
// after all other lookup methods fail.
func TestSearchIndexForReference_LastDitchRolodexFallback(t *testing.T) {
	// Primary index with NO components - searches will fail here
	primarySpec := `openapi: 3.0.1
info:
  title: Primary
  version: "1.0"`

	var primaryRoot yaml.Node
	_ = yaml.Unmarshal([]byte(primarySpec), &primaryRoot)

	c := CreateOpenAPIIndexConfig()
	primaryIdx := NewSpecIndexWithConfig(&primaryRoot, c)

	// Secondary index WITH the component we want to find
	secondarySpec := `openapi: 3.0.1
info:
  title: Secondary
  version: "1.0"
components:
  schemas:
    Pet:
      type: object
      properties:
        name:
          type: string`

	var secondaryRoot yaml.Node
	_ = yaml.Unmarshal([]byte(secondarySpec), &secondaryRoot)

	secondaryIdx := NewSpecIndexWithConfig(&secondaryRoot, c)

	// Create rolodex and add secondary index
	rolo := NewRolodex(c)
	rolo.AddIndex(secondaryIdx)

	// Set rolodex on primary index
	primaryIdx.SetRolodex(rolo)

	// Search for reference that:
	// 1. Doesn't exist in primary index's allMappedRefs
	// 2. Has roloLookup = "" (simple ref format)
	// 3. Should be found via last-ditch rolodex iteration
	ref, idx := primaryIdx.SearchIndexForReference("#/components/schemas/Pet")

	assert.NotNil(t, ref, "Reference should be found via rolodex fallback")
	assert.NotNil(t, idx, "Index should be returned")
	assert.Equal(t, "Pet", ref.Name)
}
