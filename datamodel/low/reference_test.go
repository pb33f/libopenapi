// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package low

import (
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/resolver"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestNodeReference_IsEmpty(t *testing.T) {
	nr := new(NodeReference[string])
	assert.True(t, nr.IsEmpty())
}

func TestNodeReference_GenerateMapKey(t *testing.T) {
	nr := new(NodeReference[string])
	nr.ValueNode = &yaml.Node{
		Line:   22,
		Column: 23,
	}
	assert.Equal(t, "22:23", nr.GenerateMapKey())
}

func TestNodeReference_Mutate(t *testing.T) {
	nr := new(NodeReference[string])
	nr.ValueNode = &yaml.Node{
		Line:   22,
		Column: 23,
	}
	n := nr.Mutate("nice one!")
	assert.Equal(t, "nice one!", n.Value)
	assert.Equal(t, "nice one!", nr.ValueNode.Value)
}

func TestValueReference_Mutate(t *testing.T) {
	nr := new(ValueReference[string])
	nr.ValueNode = &yaml.Node{
		Line:   22,
		Column: 23,
	}
	n := nr.Mutate("nice one!")
	assert.Equal(t, "nice one!", n.Value)
	assert.Equal(t, "nice one!", nr.ValueNode.Value)
}

func TestValueReference_IsEmpty(t *testing.T) {
	nr := new(ValueReference[string])
	assert.True(t, nr.IsEmpty())
}

func TestValueReference_GenerateMapKey(t *testing.T) {
	nr := new(ValueReference[string])
	nr.ValueNode = &yaml.Node{
		Line:   22,
		Column: 23,
	}
	assert.Equal(t, "22:23", nr.GenerateMapKey())
}

func TestKeyReference_IsEmpty(t *testing.T) {
	nr := new(KeyReference[string])
	assert.True(t, nr.IsEmpty())
}

func TestKeyReference_GenerateMapKey(t *testing.T) {
	nr := new(KeyReference[string])
	nr.KeyNode = &yaml.Node{
		Line:   22,
		Column: 23,
	}
	assert.Equal(t, "22:23", nr.GenerateMapKey())
}

func TestIsCircular_LookupFromJourney(t *testing.T) {

	yml := `components:
  schemas:
    Something:
      properties:
        nothing:
          $ref: '#/components/schemas/Nothing'
    Nothing:
      properties:
        something: 
          $ref: '#/components/schemas/Something'`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `$ref: '#/components/schemas/Something'`

	resolve := resolver.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	ref, err := LocateRefNode(idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.True(t, IsCircular(ref, idx))
}

func TestIsCircular_LookupFromLoopPoint(t *testing.T) {

	yml := `components:
  schemas:
    Something:
      properties:
        nothing:
          $ref: '#/components/schemas/Nothing'
    Nothing:
      properties:
        something: 
          $ref: '#/components/schemas/Something'`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `$ref: '#/components/schemas/Nothing'`

	resolve := resolver.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	ref, err := LocateRefNode(idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.True(t, IsCircular(ref, idx))
}

func TestIsCircular_FromRefLookup(t *testing.T) {

	yml := `components:
  schemas:
    NotCircle:
      description: not a circle
    Something:
      properties:
        nothing:
          $ref: '#/components/schemas/Nothing'
    Nothing:
      properties:
        something: 
          $ref: '#/components/schemas/Something'`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	resolve := resolver.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	yml = `$ref: '#/components/schemas/Nothing'`
	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	assert.True(t, IsCircular(idxNode.Content[0], idx))

	yml = `$ref: '#/components/schemas/NotCircle'`
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	assert.False(t, IsCircular(idxNode.Content[0], idx))
}

func TestIsCircular_NoNode(t *testing.T) {
	assert.False(t, IsCircular(nil, nil))
}

func TestGetCircularReferenceResult_NoNode(t *testing.T) {
	assert.Nil(t, GetCircularReferenceResult(nil, nil))
}

func TestGetCircularReferenceResult_FromJourney(t *testing.T) {

	yml := `components:
  schemas:
    Something:
      properties:
        nothing:
          $ref: '#/components/schemas/Nothing'
    Nothing:
      properties:
        something: 
          $ref: '#/components/schemas/Something'`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `$ref: '#/components/schemas/Something'`

	resolve := resolver.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	ref, err := LocateRefNode(idxNode.Content[0], idx)
	assert.NoError(t, err)
	circ := GetCircularReferenceResult(ref, idx)
	assert.NotNil(t, circ)
	assert.Equal(t, "Nothing -> Something -> Nothing", circ.GenerateJourneyPath())

}

func TestGetCircularReferenceResult_FromLoopPoint(t *testing.T) {

	yml := `components:
  schemas:
    Something:
      properties:
        nothing:
          $ref: '#/components/schemas/Nothing'
    Nothing:
      properties:
        something: 
          $ref: '#/components/schemas/Something'`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `$ref: '#/components/schemas/Nothing'`

	resolve := resolver.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	ref, err := LocateRefNode(idxNode.Content[0], idx)
	assert.NoError(t, err)
	circ := GetCircularReferenceResult(ref, idx)
	assert.NotNil(t, circ)
	assert.Equal(t, "Nothing -> Something -> Nothing", circ.GenerateJourneyPath())

}

func TestGetCircularReferenceResult_FromMappedRef(t *testing.T) {

	yml := `components:
  schemas:
    Something:
      properties:
        nothing:
          $ref: '#/components/schemas/Nothing'
    Nothing:
      properties:
        something: 
          $ref: '#/components/schemas/Something'`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `$ref: '#/components/schemas/Nothing'`

	resolve := resolver.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	circ := GetCircularReferenceResult(idxNode.Content[0], idx)
	assert.NotNil(t, circ)
	assert.Equal(t, "Nothing -> Something -> Nothing", circ.GenerateJourneyPath())

}

func TestGetCircularReferenceResult_NothingFound(t *testing.T) {

	yml := `components:
  schemas:
    NotCircle:
      description: not a circle`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	resolve := resolver.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 0)

	var idxNode yaml.Node
	yml = `$ref: '#/components/schemas/NotCircle'`
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	assert.Nil(t, GetCircularReferenceResult(idxNode.Content[0], idx))

}
