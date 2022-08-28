// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package low

import (
	"fmt"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/resolver"
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

	located, _ := LocateRefNode(cNode.Content[0], idx)
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

	located, _ := LocateRefNode(cNode.Content[0], idx)
	assert.NotNil(t, located)

}

func TestLocateRefNode_Path_NotFound(t *testing.T) {

	yml := `paths:
  /burger/time:
    description: hello`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	yml = `$ref: '#/paths/~1burger~1time-somethingsomethingdarkside-somethingsomethingcomplete'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	located, err := LocateRefNode(cNode.Content[0], idx)
	assert.Nil(t, located)
	assert.Error(t, err)

}

type pizza struct {
	Description NodeReference[string]
}

func (p *pizza) Build(n *yaml.Node, idx *index.SpecIndex) error {
	return nil
}

func TestExtractObject(t *testing.T) {

	yml := `components:
  schemas:
    pizza:
      description: hello`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	yml = `tags:
  description: hello pizza`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	tag, err := ExtractObject[*pizza]("tags", &cNode, idx)
	assert.NoError(t, err)
	assert.NotNil(t, tag)
	assert.Equal(t, "hello pizza", tag.Value.Description.Value)
}

func TestExtractObject_Ref(t *testing.T) {

	yml := `components:
  schemas:
    pizza:
      description: hello pizza`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	yml = `tags:
  $ref: '#/components/schemas/pizza'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	tag, err := ExtractObject[*pizza]("tags", &cNode, idx)
	assert.NoError(t, err)
	assert.NotNil(t, tag)
	assert.Equal(t, "hello pizza", tag.Value.Description.Value)
}

func TestExtractObject_DoubleRef(t *testing.T) {

	yml := `components:
  schemas:
    cake:
      description: cake time!
    pizza:
      $ref: '#/components/schemas/cake'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	yml = `tags:
  $ref: '#/components/schemas/pizza'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	tag, err := ExtractObject[*pizza]("tags", &cNode, idx)
	assert.NoError(t, err)
	assert.NotNil(t, tag)
	assert.Equal(t, "cake time!", tag.Value.Description.Value)
}

func TestExtractObject_DoubleRef_Circular(t *testing.T) {

	yml := `components:
  schemas:
    loopy:
      $ref: '#/components/schemas/cake'
    cake:
      $ref: '#/components/schemas/loopy'
    pizza:
      $ref: '#/components/schemas/cake'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	// circular references are detected by the resolver, so lets run it!
	resolv := resolver.NewResolver(idx)
	assert.Len(t, resolv.CheckForCircularReferences(), 1)

	yml = `tags:
  $ref: '#/components/schemas/pizza'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	tag, err := ExtractObject[*pizza]("tags", &cNode, idx)
	assert.NoError(t, err)
	assert.NotNil(t, tag)
	assert.Equal(t, "", tag.Value.Description.Value)
	assert.Equal(t, "cake -> loopy -> cake", idx.GetCircularReferences()[0].GenerateJourneyPath())
}

func TestExtractObject_DoubleRef_Circular_Fail(t *testing.T) {

	yml := `components:
  schemas:
    loopy:
      $ref: '#/components/schemas/cake'
    cake:
      $ref: '#/components/schemas/loopy'
    pizza:
      $ref: '#/components/schemas/cake'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	// circular references are detected by the resolver, so lets run it!
	resolv := resolver.NewResolver(idx)
	assert.Len(t, resolv.CheckForCircularReferences(), 1)

	yml = `tags:
  $ref: #BORK`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	_, err := ExtractObject[*pizza]("tags", &cNode, idx)
	assert.Error(t, err)
}

func TestExtractObject_DoubleRef_Circular_Direct(t *testing.T) {

	yml := `components:
  schemas:
    loopy:
      $ref: '#/components/schemas/cake'
    cake:
      $ref: '#/components/schemas/loopy'
    pizza:
      $ref: '#/components/schemas/cake'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	// circular references are detected by the resolver, so lets run it!
	resolv := resolver.NewResolver(idx)
	assert.Len(t, resolv.CheckForCircularReferences(), 1)

	yml = `$ref: '#/components/schemas/pizza'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	tag, err := ExtractObject[*pizza]("tags", cNode.Content[0], idx)
	assert.NoError(t, err)
	assert.NotNil(t, tag)
	assert.Equal(t, "", tag.Value.Description.Value)
	assert.Equal(t, "cake -> loopy -> cake", idx.GetCircularReferences()[0].GenerateJourneyPath())
}

func TestExtractObject_DoubleRef_Circular_Direct_Fail(t *testing.T) {

	yml := `components:
  schemas:
    loopy:
      $ref: '#/components/schemas/cake'
    cake:
      $ref: '#/components/schemas/loopy'
    pizza:
      $ref: '#/components/schemas/cake'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	// circular references are detected by the resolver, so lets run it!
	resolv := resolver.NewResolver(idx)
	assert.Len(t, resolv.CheckForCircularReferences(), 1)

	yml = `$ref: '#/components/schemas/why-did-westworld-have-to-end-so-poorly-ffs'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	_, err := ExtractObject[*pizza]("tags", cNode.Content[0], idx)
	assert.Error(t, err)

}

type test_noGood struct {
	DontWork int
}

func (t *test_noGood) Build(root *yaml.Node, idx *index.SpecIndex) error {
	return fmt.Errorf("I am always going to fail")
}

type test_almostGood struct {
	AlmostWork NodeReference[int]
}

func (t *test_almostGood) Build(root *yaml.Node, idx *index.SpecIndex) error {
	return fmt.Errorf("I am always going to fail")
}

func TestExtractObject_BadLowLevelModel(t *testing.T) {

	yml := `components:
  schemas:
   hey:`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	yml = `thing:
  dontWork: 123`
	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	_, err := ExtractObject[*test_noGood]("thing", &cNode, idx)
	assert.Error(t, err)

}

func TestExtractObject_BadBuild(t *testing.T) {

	yml := `components:
  schemas:
   hey:`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	yml = `thing:
  dontWork: 123`
	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	_, err := ExtractObject[*test_almostGood]("thing", &cNode, idx)
	assert.Error(t, err)

}

func TestExtractObject_BadLabel(t *testing.T) {

	yml := `components:
  schemas:
   hey:`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&idxNode)

	yml = `thing:
  dontWork: 123`
	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	res, err := ExtractObject[*test_almostGood]("ding", &cNode, idx)
	assert.Nil(t, res.Value)
	assert.NoError(t, err)

}
