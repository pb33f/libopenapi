// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package low

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	"github.com/pkg-base/libopenapi/utils"

	"github.com/pkg-base/libopenapi/index"
	"github.com/pkg-base/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	nr.KeyNode = &yaml.Node{
		Line:   22,
		Column: 23,
	}
	n := nr.Mutate("nice one!")
	assert.NotNil(t, nr.GetValueNode())
	assert.Empty(t, nr.GetValue())
	assert.False(t, nr.IsReference())
	assert.Equal(t, "nice one!", n.Value)
	assert.Equal(t, "nice one!", nr.ValueNode.Value)
}

func TestNodeReference_RefNode(t *testing.T) {
	nr := new(NodeReference[string])
	nr.KeyNode = utils.CreateRefNode("#/components/schemas/SomeSchema")
	nr.SetReference("#/components/schemas/SomeSchema", nr.KeyNode)
	assert.True(t, nr.IsReference())
	assert.Equal(t, nr.KeyNode, nr.GetReferenceNode())
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
	assert.NotNil(t, nr.GetValueNode())
	assert.Empty(t, nr.GetValue())
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
      required:
        - nothing
    Nothing:
      properties:
        something: 
          $ref: '#/components/schemas/Something'
      required:
        - something
`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `$ref: '#/components/schemas/Something'`

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	ref, _, err := LocateRefNode(idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.True(t, IsCircular(ref, idx))
}

func TestIsCircular_LookupFromJourney_Optional(t *testing.T) {
	yml := `components:
  schemas:
    Something:
      properties:
        nothing:
          $ref: '#/components/schemas/Nothing'
    Nothing:
      properties:
        something: 
          $ref: '#/components/schemas/Something'
`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `$ref: '#/components/schemas/Something'`

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 0)

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	ref, _, err := LocateRefNode(idxNode.Content[0], idx)
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
      required:
        - nothing
    Nothing:
      properties:
        something: 
          $ref: '#/components/schemas/Something'
      required:
        - something
`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `$ref: '#/components/schemas/Nothing'`

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	ref, _, err := LocateRefNode(idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.True(t, IsCircular(ref, idx))
}

func TestIsCircular_LookupFromLoopPoint_Optional(t *testing.T) {
	yml := `components:
  schemas:
    Something:
      properties:
        nothing:
          $ref: '#/components/schemas/Nothing'
    Nothing:
      properties:
        something: 
          $ref: '#/components/schemas/Something'
`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `$ref: '#/components/schemas/Nothing'`

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 0)

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	ref, _, err := LocateRefNode(idxNode.Content[0], idx)
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
      required:
        - nothing
    Nothing:
      properties:
        something: 
          $ref: '#/components/schemas/Something'
      required:
        - something
`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	resolve := index.NewResolver(idx)
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

func TestIsCircular_FromRefLookup_Optional(t *testing.T) {
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
          $ref: '#/components/schemas/Something'
`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 0)

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
      required:
        - nothing
    Nothing:
      properties:
        something: 
          $ref: '#/components/schemas/Something'
      required:
        - something
`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `$ref: '#/components/schemas/Something'`

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	ref, _, err := LocateRefNode(idxNode.Content[0], idx)
	assert.NoError(t, err)
	circ := GetCircularReferenceResult(ref, idx)
	assert.NotNil(t, circ)
	assert.Equal(t, "Nothing -> Something -> Nothing", circ.GenerateJourneyPath())
}

func TestGetCircularReferenceResult_FromJourney_Optional(t *testing.T) {
	yml := `components:
  schemas:
    Something:
      properties:
        nothing:
          $ref: '#/components/schemas/Nothing'
    Nothing:
      properties:
        something: 
          $ref: '#/components/schemas/Something'
`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `$ref: '#/components/schemas/Something'`

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 0)

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	ref, _, err := LocateRefNode(idxNode.Content[0], idx)
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
      required:
        - nothing
    Nothing:
      properties:
        something: 
          $ref: '#/components/schemas/Something'
      required:
        - something
`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `$ref: '#/components/schemas/Nothing'`

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	ref, _, err := LocateRefNode(idxNode.Content[0], idx)
	assert.NoError(t, err)
	circ := GetCircularReferenceResult(ref, idx)
	assert.NotNil(t, circ)
	assert.Equal(t, "Nothing -> Something -> Nothing", circ.GenerateJourneyPath())
}

func TestGetCircularReferenceResult_FromLoopPoint_Optional(t *testing.T) {
	yml := `components:
  schemas:
    Something:
      properties:
        nothing:
          $ref: '#/components/schemas/Nothing'
    Nothing:
      properties:
        something: 
          $ref: '#/components/schemas/Something'
`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `$ref: '#/components/schemas/Nothing'`

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 0)

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	ref, _, err := LocateRefNode(idxNode.Content[0], idx)
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
      required:
        - nothing
    Nothing:
      properties:
        something: 
          $ref: '#/components/schemas/Something'
      required:
        - something
`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `$ref: '#/components/schemas/Nothing'`

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	circ := GetCircularReferenceResult(idxNode.Content[0], idx)
	assert.NotNil(t, circ)
	assert.Equal(t, "Nothing -> Something -> Nothing", circ.GenerateJourneyPath())
}

func TestGetCircularReferenceResult_FromMappedRef_Optional(t *testing.T) {
	yml := `components:
  schemas:
    Something:
      properties:
        nothing:
          $ref: '#/components/schemas/Nothing'
    Nothing:
      properties:
        something: 
          $ref: '#/components/schemas/Something'
`

	var iNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &iNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&iNode)

	yml = `$ref: '#/components/schemas/Nothing'`

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 0)

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

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 0)

	var idxNode yaml.Node
	yml = `$ref: '#/components/schemas/NotCircle'`
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	assert.Nil(t, GetCircularReferenceResult(idxNode.Content[0], idx))
}

func TestHashToString(t *testing.T) {
	assert.Equal(t, "5994471abb01112afcc18159f6cc74b4f511b99806da59b3caf5a9c173cacfc5",
		HashToString(sha256.Sum256([]byte("12345"))))
}

func TestReference_IsReference(t *testing.T) {
	ref := Reference{}
	ref.SetReference("#/components/schemas/SomeSchema", nil)
	assert.True(t, ref.IsReference())
}

func TestNodeReference_NodeLineNumber(t *testing.T) {
	n := utils.CreateStringNode("pizza")
	nr := &NodeReference[string]{
		Value:     "pizza",
		ValueNode: n,
	}

	n.Line = 3
	assert.Equal(t, 3, nr.NodeLineNumber())
}

func TestNodeReference_NodeLineNumberEmpty(t *testing.T) {
	nr := &NodeReference[string]{
		Value: "pizza",
	}
	assert.Equal(t, 0, nr.NodeLineNumber())
}

func TestNodeReference_GetReference(t *testing.T) {
	nr := &NodeReference[string]{}
	nr.SetReference("#/happy/sunday", nil)
	assert.Equal(t, "#/happy/sunday", nr.GetReference())
}

func TestNodeReference_SetReference(t *testing.T) {
	nr := &NodeReference[string]{}
	nr.SetReference("#/happy/sunday", nil)
}

func TestNodeReference_GetKeyNode(t *testing.T) {
	nr := &NodeReference[string]{
		KeyNode: utils.CreateStringNode("pizza"),
	}
	assert.Equal(t, "pizza", nr.GetKeyNode().Value)
}

func TestNodeReference_GetValueUntyped(t *testing.T) {
	type anything struct {
		thing string
	}

	nr := &NodeReference[any]{
		Value: anything{thing: "ding"},
	}

	assert.Equal(t, "{ding}", fmt.Sprint(nr.GetValueUntyped()))
}

func TestValueReference_NodeLineNumber(t *testing.T) {
	n := utils.CreateStringNode("pizza")
	nr := ValueReference[string]{
		Value:     "pizza",
		ValueNode: n,
	}

	n.Line = 3
	assert.Equal(t, 3, nr.NodeLineNumber())
}

func TestValueReference_NodeLineNumber_Nil(t *testing.T) {
	nr := ValueReference[string]{
		Value: "pizza",
	}

	assert.Equal(t, 0, nr.NodeLineNumber())
}

func TestValueReference_GetReference(t *testing.T) {
	nr := ValueReference[string]{}
	nr.SetReference("#/happy/sunday", nil)
	assert.Equal(t, "#/happy/sunday", nr.GetReference())
}

func TestValueReference_GetValueUntyped(t *testing.T) {
	type anything struct {
		thing string
	}

	nr := ValueReference[any]{
		Value: anything{thing: "ding"},
	}

	assert.Equal(t, "{ding}", fmt.Sprint(nr.GetValueUntyped()))
}

func TestValueReference_MarshalYAML_Ref(t *testing.T) {
	nr := ValueReference[string]{}
	nr.SetReference("#/burgers/beer", nil)

	data, _ := yaml.Marshal(nr)
	assert.Equal(t, `$ref: '#/burgers/beer'`, strings.TrimSpace(string(data)))
}

func TestValueReference_MarshalYAML(t *testing.T) {
	v := map[string]interface{}{
		"beer": "burger",
		"wine": "cheese",
	}

	var enc yaml.Node
	enc.Encode(&v)

	nr := ValueReference[any]{
		Value:     v,
		ValueNode: &enc,
	}

	data, _ := yaml.Marshal(nr)

	expected := `beer: burger
wine: cheese`

	assert.Equal(t, expected, strings.TrimSpace(string(data)))
}

func TestKeyReference_GetValueUntyped(t *testing.T) {
	type anything struct {
		thing string
	}

	nr := KeyReference[any]{
		Value: anything{thing: "ding"},
	}

	assert.Equal(t, "{ding}", fmt.Sprint(nr.GetValueUntyped()))
}

func TestKeyReference_GetKeyNode(t *testing.T) {
	kn := utils.CreateStringNode("pizza")
	kn.Line = 3

	nr := KeyReference[any]{
		KeyNode: kn,
	}

	assert.Equal(t, 3, nr.GetKeyNode().Line)
	assert.Equal(t, "pizza", nr.GetKeyNode().Value)
}

func TestKeyReference_MarshalYAML(t *testing.T) {
	kn := utils.CreateStringNode("pizza")
	kr := KeyReference[string]{
		KeyNode: kn,
	}

	on, err := kr.MarshalYAML()
	require.NoError(t, err)
	assert.Equal(t, kn, on)
}

func TestGetCircularReferenceResult(t *testing.T) {
	kn := utils.CreateStringNode("pizza")
	assert.Empty(t, GetCircularReferenceResult(kn, &index.SpecIndex{})) // tests no resolver path
}
