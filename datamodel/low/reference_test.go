// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package low

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/utils"

	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/resolver"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestNodeReference_IsEmpty(t *testing.T) {
	t.Parallel()
	nr := new(NodeReference[string])
	assert.True(t, nr.IsEmpty())
}

func TestNodeReference_GenerateMapKey(t *testing.T) {
	t.Parallel()
	nr := new(NodeReference[string])
	nr.ValueNode = &yaml.Node{
		Line:   22,
		Column: 23,
	}
	assert.Equal(t, "22:23", nr.GenerateMapKey())
}

func TestNodeReference_Mutate(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	nr := new(NodeReference[string])
	nr.KeyNode = &yaml.Node{
		Content: []*yaml.Node{{
			Value: "$ref",
		}},
	}
	assert.True(t, nr.IsReference())
}

func TestValueReference_Mutate(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	nr := new(ValueReference[string])
	assert.True(t, nr.IsEmpty())
}

func TestValueReference_GenerateMapKey(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	nr := new(KeyReference[string])
	assert.True(t, nr.IsEmpty())
}

func TestKeyReference_GenerateMapKey(t *testing.T) {
	t.Parallel()
	nr := new(KeyReference[string])
	nr.KeyNode = &yaml.Node{
		Line:   22,
		Column: 23,
	}
	assert.Equal(t, "22:23", nr.GenerateMapKey())
}

func TestIsCircular_LookupFromJourney(t *testing.T) {
	t.Parallel()

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

	resolve := resolver.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	ref, err := LocateRefNode(idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.True(t, IsCircular(ref, idx))
}

func TestIsCircular_LookupFromJourney_Optional(t *testing.T) {
	t.Parallel()
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

	resolve := resolver.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 0)

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	ref, err := LocateRefNode(idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.True(t, IsCircular(ref, idx))
}

func TestIsCircular_LookupFromLoopPoint(t *testing.T) {
	t.Parallel()
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

	resolve := resolver.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	ref, err := LocateRefNode(idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.True(t, IsCircular(ref, idx))
}

func TestIsCircular_LookupFromLoopPoint_Optional(t *testing.T) {
	t.Parallel()
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

	resolve := resolver.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 0)

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	ref, err := LocateRefNode(idxNode.Content[0], idx)
	assert.NoError(t, err)
	assert.True(t, IsCircular(ref, idx))
}

func TestIsCircular_FromRefLookup(t *testing.T) {
	t.Parallel()

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

func TestIsCircular_FromRefLookup_Optional(t *testing.T) {
	t.Parallel()
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

	resolve := resolver.NewResolver(idx)
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
	t.Parallel()
	assert.False(t, IsCircular(nil, nil))
}

func TestGetCircularReferenceResult_NoNode(t *testing.T) {
	t.Parallel()
	assert.Nil(t, GetCircularReferenceResult(nil, nil))
}

func TestGetCircularReferenceResult_FromJourney(t *testing.T) {
	t.Parallel()
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

func TestGetCircularReferenceResult_FromJourney_Optional(t *testing.T) {
	t.Parallel()
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

	resolve := resolver.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 0)

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	ref, err := LocateRefNode(idxNode.Content[0], idx)
	assert.NoError(t, err)
	circ := GetCircularReferenceResult(ref, idx)
	assert.NotNil(t, circ)
	assert.Equal(t, "Nothing -> Something -> Nothing", circ.GenerateJourneyPath())
}

func TestGetCircularReferenceResult_FromLoopPoint(t *testing.T) {
	t.Parallel()
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

func TestGetCircularReferenceResult_FromLoopPoint_Optional(t *testing.T) {
	t.Parallel()
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

	resolve := resolver.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 0)

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	ref, err := LocateRefNode(idxNode.Content[0], idx)
	assert.NoError(t, err)
	circ := GetCircularReferenceResult(ref, idx)
	assert.NotNil(t, circ)
	assert.Equal(t, "Nothing -> Something -> Nothing", circ.GenerateJourneyPath())
}

func TestGetCircularReferenceResult_FromMappedRef(t *testing.T) {
	t.Parallel()
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

	resolve := resolver.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	circ := GetCircularReferenceResult(idxNode.Content[0], idx)
	assert.NotNil(t, circ)
	assert.Equal(t, "Nothing -> Something -> Nothing", circ.GenerateJourneyPath())
}

func TestGetCircularReferenceResult_FromMappedRef_Optional(t *testing.T) {
	t.Parallel()
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

	resolve := resolver.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 0)

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	circ := GetCircularReferenceResult(idxNode.Content[0], idx)
	assert.NotNil(t, circ)
	assert.Equal(t, "Nothing -> Something -> Nothing", circ.GenerateJourneyPath())
}

func TestGetCircularReferenceResult_NothingFound(t *testing.T) {
	t.Parallel()
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

func TestHashToString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "5994471abb01112afcc18159f6cc74b4f511b99806da59b3caf5a9c173cacfc5",
		HashToString(sha256.Sum256([]byte("12345"))))

}

func TestReference_IsReference(t *testing.T) {
	t.Parallel()
	ref := Reference{
		Reference: "#/components/schemas/SomeSchema",
	}
	assert.True(t, ref.IsReference())

}

func TestNodeReference_NodeLineNumber(t *testing.T) {
	t.Parallel()

	n := utils.CreateStringNode("pizza")
	nr := NodeReference[string]{
		Value:     "pizza",
		ValueNode: n,
	}

	n.Line = 3
	assert.Equal(t, 3, nr.NodeLineNumber())
}

func TestNodeReference_NodeLineNumberEmpty(t *testing.T) {
	t.Parallel()

	nr := NodeReference[string]{
		Value: "pizza",
	}
	assert.Equal(t, 0, nr.NodeLineNumber())
}

func TestNodeReference_GetReference(t *testing.T) {
	t.Parallel()

	nr := NodeReference[string]{
		Reference: "#/happy/sunday",
	}
	assert.Equal(t, "#/happy/sunday", nr.GetReference())
}

func TestNodeReference_SetReference(t *testing.T) {
	t.Parallel()

	nr := NodeReference[string]{}
	nr.SetReference("#/happy/sunday")
}

func TestNodeReference_IsReference(t *testing.T) {
	t.Parallel()

	nr := NodeReference[string]{
		ReferenceNode: true,
	}
	assert.True(t, nr.IsReference())
}

func TestNodeReference_GetKeyNode(t *testing.T) {
	t.Parallel()

	nr := NodeReference[string]{
		KeyNode: utils.CreateStringNode("pizza"),
	}
	assert.Equal(t, "pizza", nr.GetKeyNode().Value)

}

func TestNodeReference_GetValueUntyped(t *testing.T) {
	t.Parallel()

	type anything struct {
		thing string
	}

	nr := NodeReference[any]{
		Value: anything{thing: "ding"},
	}

	assert.Equal(t, "{ding}", fmt.Sprint(nr.GetValueUntyped()))
}

func TestValueReference_NodeLineNumber(t *testing.T) {
	t.Parallel()

	n := utils.CreateStringNode("pizza")
	nr := ValueReference[string]{
		Value:     "pizza",
		ValueNode: n,
	}

	n.Line = 3
	assert.Equal(t, 3, nr.NodeLineNumber())
}

func TestValueReference_NodeLineNumber_Nil(t *testing.T) {
	t.Parallel()

	nr := ValueReference[string]{
		Value: "pizza",
	}

	assert.Equal(t, 0, nr.NodeLineNumber())
}

func TestValueReference_GetReference(t *testing.T) {
	t.Parallel()

	nr := ValueReference[string]{
		Reference: "#/happy/sunday",
	}
	assert.Equal(t, "#/happy/sunday", nr.GetReference())
}

func TestValueReference_SetReference(t *testing.T) {
	t.Parallel()

	nr := ValueReference[string]{}
	nr.SetReference("#/happy/sunday")
}

func TestValueReference_GetValueUntyped(t *testing.T) {
	t.Parallel()

	type anything struct {
		thing string
	}

	nr := ValueReference[any]{
		Value: anything{thing: "ding"},
	}

	assert.Equal(t, "{ding}", fmt.Sprint(nr.GetValueUntyped()))
}

func TestValueReference_IsReference(t *testing.T) {
	t.Parallel()

	nr := NodeReference[string]{
		ReferenceNode: true,
	}
	assert.True(t, nr.IsReference())
}

func TestValueReference_MarshalYAML_Ref(t *testing.T) {
	t.Parallel()

	nr := ValueReference[string]{
		ReferenceNode: true,
		Reference:     "#/burgers/beer",
	}

	data, _ := yaml.Marshal(nr)
	assert.Equal(t, `$ref: '#/burgers/beer'`, strings.TrimSpace(string(data)))

}

func TestValueReference_MarshalYAML(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	type anything struct {
		thing string
	}

	nr := KeyReference[any]{
		Value: anything{thing: "ding"},
	}

	assert.Equal(t, "{ding}", fmt.Sprint(nr.GetValueUntyped()))
}

func TestKeyReference_GetKeyNode(t *testing.T) {
	t.Parallel()
	kn := utils.CreateStringNode("pizza")
	kn.Line = 3

	nr := KeyReference[any]{
		KeyNode: kn,
	}

	assert.Equal(t, 3, nr.GetKeyNode().Line)
	assert.Equal(t, "pizza", nr.GetKeyNode().Value)
}
