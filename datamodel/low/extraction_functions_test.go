// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package low

import (
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestFindItemInMap(t *testing.T) {
	v := make(map[KeyReference[string]]ValueReference[string])
	v[KeyReference[string]{
		Value: "pizza",
	}] = ValueReference[string]{
		Value: "pie",
	}
	assert.Equal(t, "pie", FindItemInMap("pizza", v).Value)
}

func TestFindItemInMap_Error(t *testing.T) {
	v := make(map[KeyReference[string]]ValueReference[string])
	v[KeyReference[string]{
		Value: "pizza",
	}] = ValueReference[string]{
		Value: "pie",
	}
	assert.Nil(t, FindItemInMap("nuggets", v))
}

func TestLocateRefNode(t *testing.T) {

	yml := `components:
  schemas:
    cake:
      description: hello`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	yml = `$ref: '#/components/schemas/cake'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	located := LocateRefNode(cNode.Content[0], idx)
	assert.NotNil(t, located)

}

func TestLocateRefNode_Path(t *testing.T) {

	yml := `paths:
  /burger/time:
    description: hello`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	yml = `$ref: '#/paths/~1burger~1time'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	located := LocateRefNode(cNode.Content[0], idx)
	assert.NotNil(t, located)

}
