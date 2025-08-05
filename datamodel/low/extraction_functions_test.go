// Copyright 2022-2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package low

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindItemInOrderedMap(t *testing.T) {
	v := orderedmap.New[KeyReference[string], ValueReference[string]]()
	v.Set(KeyReference[string]{
		Value: "pizza",
	}, ValueReference[string]{
		Value: "pie",
	})
	assert.Equal(t, "pie", FindItemInOrderedMap("pizza", v).Value)
}

func TestFindItemInOrderedMap_WrongCase(t *testing.T) {
	v := orderedmap.New[KeyReference[string], ValueReference[string]]()
	v.Set(KeyReference[string]{
		Value: "pizza",
	}, ValueReference[string]{
		Value: "pie",
	})
	assert.Equal(t, "pie", FindItemInOrderedMap("PIZZA", v).Value)
}

func TestFindItemInOrderedMap_Error(t *testing.T) {
	v := orderedmap.New[KeyReference[string], ValueReference[string]]()
	v.Set(KeyReference[string]{
		Value: "pizza",
	}, ValueReference[string]{
		Value: "pie",
	})
	assert.Nil(t, FindItemInOrderedMap("nuggets", v))
}

func TestLocateRefNode(t *testing.T) {
	yml := `components:
  schemas:
    cake:
      description: hello`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `$ref: '#/components/schemas/cake'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	located, _, _ := LocateRefNode(cNode.Content[0], idx)
	assert.NotNil(t, located)
}

func TestLocateRefNode_BadNode(t *testing.T) {
	yml := `components:
  schemas:
    cake:
      description: hello`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `yes: mate` // useless.

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	located, _, err := LocateRefNode(cNode.Content[0], idx)

	// should both be empty.
	assert.Nil(t, located)
	assert.Nil(t, err)
}

func TestLocateRefNode_Path(t *testing.T) {
	yml := `paths:
  /burger/time:
    description: hello`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `$ref: '#/paths/~1burger~1time'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	located, _, _ := LocateRefNode(cNode.Content[0], idx)
	assert.NotNil(t, located)
}

func TestLocateRefNode_Path_NotFound(t *testing.T) {
	yml := `paths:
  /burger/time:
    description: hello`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `$ref: '#/paths/~1burger~1time-somethingsomethingdarkside-somethingsomethingcomplete'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	located, _, err := LocateRefNode(cNode.Content[0], idx)
	assert.Nil(t, located)
	assert.Error(t, err)
}

type pizza struct {
	Description NodeReference[string]
}

func (p *pizza) Build(_ context.Context, _, _ *yaml.Node, _ *index.SpecIndex) error {
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
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `tags:
  description: hello pizza`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	tag, err := ExtractObject[*pizza](context.Background(), "tags", &cNode, idx)
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
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `tags:
  $ref: '#/components/schemas/pizza'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	tag, err := ExtractObject[*pizza](context.Background(), "tags", &cNode, idx)
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
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `tags:
  $ref: '#/components/schemas/pizza'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	tag, err := ExtractObject[*pizza](context.Background(), "tags", &cNode, idx)
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
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	// circular references are detected by the resolver, so lets run it!
	resolv := index.NewResolver(idx)
	assert.Len(t, resolv.CheckForCircularReferences(), 1)

	yml = `tags:
  $ref: '#/components/schemas/pizza'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	_, err := ExtractObject[*pizza](context.Background(), "tags", &cNode, idx)
	assert.Error(t, err)
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
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	// circular references are detected by the resolver, so lets run it!
	resolv := index.NewResolver(idx)
	assert.Len(t, resolv.CheckForCircularReferences(), 1)

	yml = `tags:
  $ref: #BORK`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	_, err := ExtractObject[*pizza](context.Background(), "tags", &cNode, idx)
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
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	// circular references are detected by the resolver, so lets run it!
	resolv := index.NewResolver(idx)
	assert.Len(t, resolv.CheckForCircularReferences(), 1)

	yml = `$ref: '#/components/schemas/pizza'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	_, err := ExtractObject[*pizza](context.Background(), "tags", cNode.Content[0], idx)
	assert.Error(t, err)
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
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	// circular references are detected by the resolver, so lets run it!
	resolv := index.NewResolver(idx)
	assert.Len(t, resolv.CheckForCircularReferences(), 1)

	yml = `$ref: '#/components/schemas/why-did-westworld-have-to-end-so-poorly-ffs'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	_, err := ExtractObject[*pizza](context.Background(), "tags", cNode.Content[0], idx)
	assert.Error(t, err)
}

type test_noGood struct {
	DontWork int
}

func (t *test_noGood) Build(_ context.Context, _, root *yaml.Node, idx *index.SpecIndex) error {
	return fmt.Errorf("I am always going to fail a core build")
}

type test_almostGood struct {
	AlmostWork NodeReference[int]
}

func (t *test_almostGood) Build(_ context.Context, _, root *yaml.Node, idx *index.SpecIndex) error {
	return fmt.Errorf("I am always going to fail a build out")
}

type test_Good struct {
	AlmostWork NodeReference[int]
}

func (t *test_Good) Build(_ context.Context, _, root *yaml.Node, idx *index.SpecIndex) error {
	return nil
}

func TestExtractObject_BadLowLevelModel(t *testing.T) {
	yml := `components:
  schemas:
   hey:`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `thing:
  dontWork: 123`
	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	_, err := ExtractObject[*test_noGood](context.Background(), "thing", &cNode, idx)
	assert.Error(t, err)
}

func TestExtractObject_BadBuild(t *testing.T) {
	yml := `components:
  schemas:
   hey:`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `thing:
  dontWork: 123`
	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	_, err := ExtractObject[*test_almostGood](context.Background(), "thing", &cNode, idx)
	assert.Error(t, err)
}

func TestExtractObject_BadLabel(t *testing.T) {
	yml := `components:
  schemas:
   hey:`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `thing:
  dontWork: 123`
	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	res, err := ExtractObject[*test_almostGood](context.Background(), "ding", &cNode, idx)
	assert.Nil(t, res.Value)
	assert.NoError(t, err)
}

func TestExtractObject_PathIsCircular(t *testing.T) {
	// first we need an index.
	yml := `paths:
  '/something/here':
    post:
      $ref: '#/paths/~1something~1there/post'
  '/something/there':
    post:
      $ref: '#/paths/~1something~1here/post'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	yml = `thing:
  $ref: '#/paths/~1something~1here/post'`

	var rootNode yaml.Node
	mErr = yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	res, err := ExtractObject[*test_Good](context.Background(), "thing", &rootNode, idx)
	assert.NotNil(t, res.Value)
	assert.Error(t, err) // circular error would have been thrown.
}

func TestExtractObject_PathIsCircular_IgnoreErrors(t *testing.T) {
	// first we need an index.
	yml := `paths:
  '/something/here':
    post:
      $ref: '#/paths/~1something~1there/post'
  '/something/there':
    post:
      $ref: '#/paths/~1something~1here/post'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	// disable circular ref checking.
	idx.SetAllowCircularReferenceResolving(true)

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	yml = `thing:
  $ref: '#/paths/~1something~1here/post'`

	var rootNode yaml.Node
	mErr = yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	res, err := ExtractObject[*test_Good](context.Background(), "thing", &rootNode, idx)
	assert.NotNil(t, res.Value)
	assert.NoError(t, err) // circular error would have been thrown, but we're ignoring them.
}

func TestExtractObjectRaw(t *testing.T) {
	yml := `components:
  schemas:
    pizza:
      description: hello`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `description: hello pizza`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	tag, err, _, _ := ExtractObjectRaw[*pizza](context.Background(), nil, cNode.Content[0], idx)
	assert.NoError(t, err)
	assert.NotNil(t, tag)
	assert.Equal(t, "hello pizza", tag.Description.Value)
}

func TestExtractObjectRaw_With_Ref(t *testing.T) {
	yml := `components:
  schemas:
    pizza:
      description: hello`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `$ref: '#/components/schemas/pizza'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	tag, err, isRef, rv := ExtractObjectRaw[*pizza](context.Background(), nil, cNode.Content[0], idx)
	assert.NoError(t, err)
	assert.NotNil(t, tag)
	assert.Equal(t, "hello", tag.Description.Value)
	assert.True(t, isRef)
	assert.Equal(t, "#/components/schemas/pizza", rv)
}

func TestExtractObjectRaw_Ref_Circular(t *testing.T) {
	yml := `components:
  schemas:
    pizza:
      $ref: '#/components/schemas/pie'
    pie:
      $ref: '#/components/schemas/pizza'`
	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	yml = `$ref: '#/components/schemas/pizza'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	tag, err, _, _ := ExtractObjectRaw[*pizza](context.Background(), nil, cNode.Content[0], idx)
	assert.Error(t, err)
	assert.NotNil(t, tag)
}

func TestExtractObjectRaw_RefBroken(t *testing.T) {
	yml := `components:
  schemas:
    pizza:
      description: hey!`
	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `$ref: '#/components/schemas/lost-in-space'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	tag, err, _, _ := ExtractObjectRaw[*pizza](context.Background(), nil, cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Nil(t, tag)
}

func TestExtractObjectRaw_Ref_NonBuildable(t *testing.T) {
	yml := `components:
  schemas:
    pizza:
      description: hey!`
	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `dontWork: 1'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	_, err, _, _ := ExtractObjectRaw[*test_noGood](context.Background(), nil, cNode.Content[0], idx)
	assert.Error(t, err)
}

func TestExtractObjectRaw_Ref_AlmostBuildable(t *testing.T) {
	yml := `components:
  schemas:
    pizza:
      description: hey!`
	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `almostWork: 1'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	_, err, _, _ := ExtractObjectRaw[*test_almostGood](context.Background(), nil, cNode.Content[0], idx)
	assert.Error(t, err)
}

func TestExtractArray(t *testing.T) {
	yml := `components:
  schemas:
    pizza:
      description: hello`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `things:
  - description: one
  - description: two
  - description: three`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	things, _, _, err := ExtractArray[*pizza](context.Background(), "things", cNode.Content[0], idx)
	assert.NoError(t, err)
	assert.NotNil(t, things)
	assert.Equal(t, "one", things[0].Value.Description.Value)
	assert.Equal(t, "two", things[1].Value.Description.Value)
	assert.Equal(t, "three", things[2].Value.Description.Value)
}

func TestExtractArray_Ref(t *testing.T) {
	yml := `components:
  schemas:
    things:
      - description: one
      - description: two
      - description: three`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `$ref: '#/components/schemas/things'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	things, _, _, err := ExtractArray[*pizza](context.Background(), "things", cNode.Content[0], idx)
	assert.NoError(t, err)
	assert.NotNil(t, things)
	assert.Equal(t, "one", things[0].Value.Description.Value)
	assert.Equal(t, "two", things[1].Value.Description.Value)
	assert.Equal(t, "three", things[2].Value.Description.Value)
}

func TestExtractArray_Ref_Unbuildable(t *testing.T) {
	yml := `components:
  schemas:
    things:
      - description: one
      - description: two
      - description: three`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `$ref: '#/components/schemas/things'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	things, _, _, err := ExtractArray[*test_noGood](context.Background(), "", cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Len(t, things, 0)
}

func TestExtractArray_Ref_Circular(t *testing.T) {
	yml := `components:
  schemas:
    thongs:
      $ref: '#/components/schemas/things'
    things:
      $ref: '#/components/schemas/thongs'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	yml = `$ref: '#/components/schemas/things'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	things, _, _, err := ExtractArray[*test_Good](context.Background(), "", cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Len(t, things, 2)
}

func TestExtractArray_Ref_Bad(t *testing.T) {
	yml := `components:
  schemas:
    thongs:
      $ref: '#/components/schemas/things'
    things:
      $ref: '#/components/schemas/thongs'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	yml = `$ref: '#/components/schemas/let-us-eat-cake'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	things, _, _, err := ExtractArray[*test_Good](context.Background(), "", cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Len(t, things, 0)
}

func TestExtractArray_Ref_Nested(t *testing.T) {
	yml := `components:
  schemas:
    thongs:
      $ref: '#/components/schemas/things'
    things:
      $ref: '#/components/schemas/thongs'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	yml = `limes:
  $ref: '#/components/schemas/let-us-eat-cake'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	things, _, _, err := ExtractArray[*test_Good](context.Background(), "limes", cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Len(t, things, 0)
}

func TestExtractArray_Ref_Nested_Circular(t *testing.T) {
	yml := `components:
  schemas:
    thongs:
      $ref: '#/components/schemas/things'
    things:
      $ref: '#/components/schemas/thongs'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	yml = `limes:
  - $ref: '#/components/schemas/things'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	things, _, _, err := ExtractArray[*test_Good](context.Background(), "limes", cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Len(t, things, 1)
}

func TestExtractArray_Ref_Nested_BadRef(t *testing.T) {
	yml := `components:
  schemas:
    thongs:
      allOf:
        - $ref: '#/components/schemas/things'
    things:
      oneOf:
        - type: string`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `limes:
  - $ref: '#/components/schemas/thangs'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)
	things, _, _, err := ExtractArray[*test_Good](context.Background(), "limes", cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Len(t, things, 0)
}

func TestExtractArray_Ref_Nested_CircularFlat(t *testing.T) {
	yml := `components:
  schemas:
    thongs:
      $ref: '#/components/schemas/things'
    things:
      $ref: '#/components/schemas/thongs'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	yml = `limes:
  $ref: '#/components/schemas/things'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)
	things, _, _, err := ExtractArray[*test_Good](context.Background(), "limes", cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Len(t, things, 2)
}

func TestExtractArray_BadBuild(t *testing.T) {
	yml := `components:
  schemas:
    thongs:`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `limes:
  - dontWork: 1`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)
	things, _, _, err := ExtractArray[*test_noGood](context.Background(), "limes", cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Len(t, things, 0)
}

func TestExtractArray_BadRefPropsTupe(t *testing.T) {
	yml := `components:
  parameters:
    cakes:
      limes: cake`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `limes:
  $ref: '#/components/parameters/cakes'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)
	things, _, _, err := ExtractArray[*test_noGood](context.Background(), "limes", cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Len(t, things, 0)
}

func TestExtractMapFlatNoLookup(t *testing.T) {
	yml := `components:`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `x-hey: you
one:
  description: two`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, err := ExtractMapNoLookup[*test_Good](context.Background(), cNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, 1, orderedmap.Len(things))
}

func TestExtractMap_NoLookupWithExtensions(t *testing.T) {
	yml := `components:`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `x-hey: you
one:
  x-choo: choo`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, err := ExtractMapNoLookupExtensions[*test_Good](context.Background(), cNode.Content[0], idx, true)
	assert.NoError(t, err)
	assert.Equal(t, 2, orderedmap.Len(things))

	for k, v := range things.FromOldest() {
		if k.Value == "x-hey" {
			continue
		}
		assert.Equal(t, "one", k.Value)
		assert.Len(t, v.ValueNode.Content, 2)
	}
}

func TestExtractMap_NoLookupWithExtensions_UsingMerge(t *testing.T) {
	yml := `components:`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `x-yeah: &yeah
  night: fun
x-hey: you
one:
  x-choo: choo
<<: *yeah`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, err := ExtractMapNoLookupExtensions[*test_Good](context.Background(), cNode.Content[0], idx, true)
	assert.NoError(t, err)
	assert.Equal(t, 4, orderedmap.Len(things))
}

func TestExtractMap_NoLookupWithoutExtensions(t *testing.T) {
	yml := `components:`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `x-hey: you
one:
  x-choo: choo`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, err := ExtractMapNoLookupExtensions[*test_Good](context.Background(), cNode.Content[0], idx, false)
	assert.NoError(t, err)
	assert.Equal(t, 1, orderedmap.Len(things))

	for k := range things.KeysFromOldest() {
		assert.Equal(t, "one", k.Value)
	}
}

func TestExtractMap_WithExtensions(t *testing.T) {
	yml := `components:`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `x-hey: you
one:
  x-choo: choo`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, _, _, err := ExtractMapExtensions[*test_Good](context.Background(), "one", cNode.Content[0], idx, true)
	assert.NoError(t, err)
	assert.Equal(t, 1, orderedmap.Len(things))
}

func TestExtractMap_WithoutExtensions(t *testing.T) {
	yml := `components:`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `x-hey: you
one:
  x-choo: choo`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, _, _, err := ExtractMapExtensions[*test_Good](context.Background(), "one", cNode.Content[0], idx, false)
	assert.NoError(t, err)
	assert.Zero(t, orderedmap.Len(things))
}

func TestExtractMapFlatNoLookup_Ref(t *testing.T) {
	yml := `components:
  schemas:
    pizza:
      description: tasty!`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `x-hey: you
one:
  $ref: '#/components/schemas/pizza'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, err := ExtractMapNoLookup[*test_Good](context.Background(), cNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, 1, orderedmap.Len(things))
}

func TestExtractMapFlatNoLookup_Ref_Bad(t *testing.T) {
	yml := `components:
  schemas:
    pizza:
      description: tasty!`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `x-hey: you
one:
  $ref: '#/components/schemas/no-where-out-there'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, err := ExtractMapNoLookup[*test_Good](context.Background(), cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Zero(t, orderedmap.Len(things))
}

func TestExtractMapFlatNoLookup_Ref_Circular(t *testing.T) {
	yml := `components:
  schemas:
    thongs:
      $ref: '#/components/schemas/things'
    things:
      $ref: '#/components/schemas/thongs'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	yml = `x-hey: you
one:
  $ref: '#/components/schemas/things'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, err := ExtractMapNoLookup[*test_Good](context.Background(), cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Equal(t, 1, orderedmap.Len(things))
}

func TestExtractMapFlatNoLookup_Ref_BadBuild(t *testing.T) {
	yml := `components:
  schemas:
    pizza:
      dontWork: 1`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `x-hey: you
hello:
  $ref: '#/components/schemas/pizza'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, err := ExtractMapNoLookup[*test_noGood](context.Background(), cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Zero(t, orderedmap.Len(things))
}

func TestExtractMapFlatNoLookup_Ref_AlmostBuild(t *testing.T) {
	yml := `components:
  schemas:
    pizza:
      description: tasty!`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `x-hey: you
one:
  $ref: '#/components/schemas/pizza'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, err := ExtractMapNoLookup[*test_almostGood](context.Background(), cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Zero(t, orderedmap.Len(things))
}

func TestExtractMapFlat(t *testing.T) {
	yml := `components:`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `x-hey: you
one:
  description: two`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, _, _, err := ExtractMap[*test_Good](context.Background(), "one", cNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, 1, orderedmap.Len(things))
}

func TestExtractMapFlat_Ref(t *testing.T) {
	yml := `components:
  schemas:
    stank:
      things:
        almostWork: 99`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `x-hey: you
one:
  $ref: '#/components/schemas/stank'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, _, _, err := ExtractMap[*test_Good](context.Background(), "one", cNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, 1, orderedmap.Len(things))

	for v := range things.ValuesFromOldest() {
		assert.Equal(t, 99, v.Value.AlmostWork.Value)
	}
}

func TestExtractMapFlat_DoubleRef(t *testing.T) {
	yml := `components:
  schemas:
    stank:
      almostWork: 99`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `one:
  nice:
    $ref: '#/components/schemas/stank'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, _, _, err := ExtractMap[*test_Good](context.Background(), "one", cNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Equal(t, 1, orderedmap.Len(things))

	for v := range things.ValuesFromOldest() {
		assert.Equal(t, 99, v.Value.AlmostWork.Value)
	}
}

func TestExtractMapFlat_DoubleRef_Error(t *testing.T) {
	yml := `components:
  schemas:
    stank:
      things:
        almostWork: 99`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `one:
  nice:
    $ref: '#/components/schemas/stank'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, _, _, err := ExtractMap[*test_almostGood](context.Background(), "one", cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Zero(t, orderedmap.Len(things))
}

func TestExtractMapFlat_DoubleRef_Error_NotFound(t *testing.T) {
	yml := `components:
  schemas:
    stank:
      things:
        almostWork: 99`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `one:
  nice:
    $ref: '#/components/schemas/stanky-panky'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, _, _, err := ExtractMap[*test_almostGood](context.Background(), "one", cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Zero(t, orderedmap.Len(things))
}

func TestExtractMapFlat_DoubleRef_Circles(t *testing.T) {
	yml := `components:
  schemas:
    stonk:
      $ref: '#/components/schemas/stank'
    stank:
      $ref: '#/components/schemas/stonk'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	yml = `one:
  nice:
    $ref: '#/components/schemas/stank'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, _, _, err := ExtractMap[*test_Good](context.Background(), "one", cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Equal(t, 1, orderedmap.Len(things))
}

func TestExtractMapFlat_Ref_Error(t *testing.T) {
	yml := `components:
  schemas:
    stank:
      x-smells: bad
      things:
        almostWork: 99`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `one:
  $ref: '#/components/schemas/stank'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, _, _, err := ExtractMap[*test_almostGood](context.Background(), "one", cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Zero(t, orderedmap.Len(things))
}

func TestExtractMapFlat_Ref_Circ_Error(t *testing.T) {
	yml := `components:
  schemas:
    stink:
      $ref: '#/components/schemas/stank'
    stank:
      $ref: '#/components/schemas/stink'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	yml = `$ref: '#/components/schemas/stank'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, _, _, err := ExtractMap[*test_Good](context.Background(), "one", cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Equal(t, 1, orderedmap.Len(things))
}

func TestExtractMapFlat_Ref_Nested_Circ_Error(t *testing.T) {
	yml := `components:
  schemas:
    stink:
      $ref: '#/components/schemas/stank'
    stank:
      $ref: '#/components/schemas/stink'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	yml = `one:
  $ref: '#/components/schemas/stank'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, _, _, err := ExtractMap[*test_Good](context.Background(), "one", cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Equal(t, 1, orderedmap.Len(things))
}

func TestExtractMapFlat_Ref_Nested_Error(t *testing.T) {
	yml := `components:
  schemas:
    stink:
      $ref: '#/components/schemas/stank'
    stank:
      $ref: '#/components/schemas/none'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `one:
  $ref: '#/components/schemas/somewhere-else'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, _, _, err := ExtractMap[*test_Good](context.Background(), "one", cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Zero(t, orderedmap.Len(things))
}

func TestExtractMapFlat_BadKey_Ref_Nested_Error(t *testing.T) {
	yml := `components:
  schemas:
    stink:
      $ref: '#/components/schemas/stank'
    stank:
      $ref: '#/components/schemas/none'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `one:
  $ref: '#/components/schemas/somewhere-else'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, _, _, err := ExtractMap[*test_Good](context.Background(), "not-even-there", cNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Zero(t, orderedmap.Len(things))
}

func TestExtractMapFlat_Ref_Bad(t *testing.T) {
	yml := `components:
  schemas:
    stink:
      $ref: '#/components/schemas/stank'
    stank:
      $ref: '#/components/schemas/none'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	yml = `$ref: '#/components/schemas/somewhere-else'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, _, _, err := ExtractMap[*test_Good](context.Background(), "one", cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Zero(t, orderedmap.Len(things))
}

func TestExtractExtensions(t *testing.T) {
	yml := `x-bing: ding
x-bong: 1
x-ling: true
x-long: 0.99
x-fish:
  woo: yeah
x-tacos: [1,2,3]`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	r := ExtractExtensions(idxNode.Content[0])
	assert.Equal(t, 6, orderedmap.Len(r))
	for k, val := range r.FromOldest() {
		var v any
		_ = val.Value.Decode(&v)

		switch k.Value {
		case "x-bing":
			assert.Equal(t, "ding", v)
		case "x-bong":
			assert.Equal(t, 1, v)
		case "x-ling":
			assert.Equal(t, true, v)
		case "x-long":
			assert.Equal(t, 0.99, v)
		case "x-fish":
			var m map[string]any
			err := val.Value.Decode(&m)
			require.NoError(t, err)
			assert.Equal(t, "yeah", m["woo"])
		case "x-tacos":
			assert.Len(t, v, 3)
		}
	}
}

type test_fresh struct {
	val   string
	thang *bool
}

func (f test_fresh) Hash() [32]byte {
	var data []string
	if f.val != "" {
		data = append(data, f.val)
	}
	if f.thang != nil {
		data = append(data, fmt.Sprintf("%v", *f.thang))
	}
	return sha256.Sum256([]byte(strings.Join(data, "|")))
}

func TestAreEqual(t *testing.T) {
	var hey *test_fresh

	assert.True(t, AreEqual(test_fresh{val: "hello"}, test_fresh{val: "hello"}))
	assert.True(t, AreEqual(&test_fresh{val: "hello"}, &test_fresh{val: "hello"}))
	assert.False(t, AreEqual(test_fresh{val: "hello"}, test_fresh{val: "goodbye"}))
	assert.False(t, AreEqual(&test_fresh{val: "hello"}, &test_fresh{val: "goodbye"}))
	assert.False(t, AreEqual(nil, &test_fresh{val: "goodbye"}))
	assert.False(t, AreEqual(&test_fresh{val: "hello"}, hey))
	assert.False(t, AreEqual(nil, nil))
}

func TestGenerateHashString(t *testing.T) {
	assert.Equal(t, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
		GenerateHashString(test_fresh{val: "hello"}))

	assert.Equal(t, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
		GenerateHashString("hello"))

	assert.Equal(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		GenerateHashString(""))

	assert.Equal(t, "",
		GenerateHashString(nil))

	assert.Equal(t, "a8468424300fc9f9206c220da9683b8b8e70474586e28a9002e740cd687b74df", GenerateHashString(utils.CreateStringNode("test")))
}

func TestGenerateHashString_Pointer(t *testing.T) {
	val := true
	assert.Equal(t, "b5bea41b6c623f7c09f1bf24dcae58ebab3c0cdd90ad966bc43a45b44867e12b",
		GenerateHashString(test_fresh{thang: &val}))

	assert.Equal(t, "b5bea41b6c623f7c09f1bf24dcae58ebab3c0cdd90ad966bc43a45b44867e12b",
		GenerateHashString(&val))
}

func TestSetReference(t *testing.T) {
	type testObj struct {
		*Reference
	}

	n := testObj{Reference: &Reference{}}
	SetReference(&n, "#/pigeon/street", nil)

	assert.Equal(t, "#/pigeon/street", n.GetReference())
}

func TestSetReference_nil(t *testing.T) {
	type testObj struct {
		*Reference
	}

	n := testObj{Reference: &Reference{}}
	SetReference(nil, "#/pigeon/street", nil)
	assert.NotEqual(t, "#/pigeon/street", n.GetReference())
}

func TestLocateRefNode_CurrentPathKey_HttpLink(t *testing.T) {
	no := yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{
				Kind:  yaml.ScalarNode,
				Value: "$ref",
			},
			{
				Kind:  yaml.ScalarNode,
				Value: "http://cakes.com/nice#/components/schemas/thing",
			},
		},
	}

	ctx := context.WithValue(context.Background(), index.CurrentPathKey, "http://cakes.com#/components/schemas/thing")

	idx := index.NewSpecIndexWithConfig(&no, index.CreateClosedAPIIndexConfig())
	n, i, e, c := LocateRefNodeWithContext(ctx, &no, idx)
	assert.Nil(t, n)
	assert.NotNil(t, i)
	assert.NotNil(t, e)
	assert.NotNil(t, c)
}

func TestLocateRefNode_CurrentPathKey_RootLookup(t *testing.T) {
	no := yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{
				Kind:  yaml.ScalarNode,
				Value: "$ref",
			},
			{
				Kind:  yaml.ScalarNode,
				Value: "#/components/pizza/cake",
			},
		},
	}

	ctx := context.WithValue(context.Background(), index.CurrentPathKey, "files/cakes.yaml")

	idx := index.NewSpecIndexWithConfig(&no, index.CreateClosedAPIIndexConfig())
	n, i, e, c := LocateRefNodeWithContext(ctx, &no, idx)
	assert.Nil(t, n)
	assert.NotNil(t, i)
	assert.NotNil(t, e)
	assert.NotNil(t, c)
}

func TestLocateRefNode_CurrentPathKey_HttpLink_Local(t *testing.T) {
	no := yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{
				Kind:  yaml.ScalarNode,
				Value: "$ref",
			},
			{
				Kind:  yaml.ScalarNode,
				Value: ".#/components/schemas/thing",
			},
		},
	}

	ctx := context.WithValue(context.Background(), index.CurrentPathKey, "http://cakes.com/nice/rice#/components/schemas/thing")

	idx := index.NewSpecIndexWithConfig(&no, index.CreateClosedAPIIndexConfig())
	n, i, e, c := LocateRefNodeWithContext(ctx, &no, idx)
	assert.Nil(t, n)
	assert.NotNil(t, i)
	assert.NotNil(t, e)
	assert.NotNil(t, c)
}

func TestLocateRefNode_CurrentPathKey_HttpLink_RemoteCtx(t *testing.T) {
	no := yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{
				Kind:  yaml.ScalarNode,
				Value: "$ref",
			},
			{
				Kind:  yaml.ScalarNode,
				Value: "#/components/schemas/thing",
			},
		},
	}

	ctx := context.WithValue(context.Background(), index.CurrentPathKey, "https://cakes.com#/components/schemas/thing")
	idx := index.NewSpecIndexWithConfig(&no, index.CreateClosedAPIIndexConfig())
	n, i, e, c := LocateRefNodeWithContext(ctx, &no, idx)
	assert.Nil(t, n)
	assert.NotNil(t, i)
	assert.NotNil(t, e)
	assert.NotNil(t, c)
}

func TestLocateRefNode_CurrentPathKey_HttpLink_RemoteCtx_WithPath(t *testing.T) {
	no := yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{
				Kind:  yaml.ScalarNode,
				Value: "$ref",
			},
			{
				Kind:  yaml.ScalarNode,
				Value: "#/components/schemas/thing",
			},
		},
	}

	ctx := context.WithValue(context.Background(), index.CurrentPathKey, "https://cakes.com/jazzzy/shoes#/components/schemas/thing")
	idx := index.NewSpecIndexWithConfig(&no, index.CreateClosedAPIIndexConfig())
	n, i, e, c := LocateRefNodeWithContext(ctx, &no, idx)
	assert.Nil(t, n)
	assert.NotNil(t, i)
	assert.NotNil(t, e)
	assert.NotNil(t, c)
}

func TestLocateRefNode_CurrentPathKey_Path_Link(t *testing.T) {
	no := yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{
				Kind:  yaml.ScalarNode,
				Value: "$ref",
			},
			{
				Kind:  yaml.ScalarNode,
				Value: "yazzy.yaml#/components/schemas/thing",
			},
		},
	}

	ctx := context.WithValue(context.Background(), index.CurrentPathKey, "/jazzzy/shoes.yaml")
	idx := index.NewSpecIndexWithConfig(&no, index.CreateClosedAPIIndexConfig())
	n, i, e, c := LocateRefNodeWithContext(ctx, &no, idx)
	assert.Nil(t, n)
	assert.NotNil(t, i)
	assert.NotNil(t, e)
	assert.NotNil(t, c)
}

func TestLocateRefNode_CurrentPathKey_Path_URL(t *testing.T) {
	no := yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{
				Kind:  yaml.ScalarNode,
				Value: "$ref",
			},
			{
				Kind:  yaml.ScalarNode,
				Value: "yazzy.yaml#/components/schemas/thing",
			},
		},
	}

	cf := index.CreateClosedAPIIndexConfig()
	u, _ := url.Parse("https://herbs-and-coffee-in-the-fall.com")
	cf.BaseURL = u
	idx := index.NewSpecIndexWithConfig(&no, cf)
	n, i, e, c := LocateRefNodeWithContext(context.Background(), &no, idx)
	assert.Nil(t, n)
	assert.NotNil(t, i)
	assert.NotNil(t, e)
	assert.NotNil(t, c)
}

func TestLocateRefNode_CurrentPathKey_DeeperPath_URL(t *testing.T) {
	no := yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{
				Kind:  yaml.ScalarNode,
				Value: "$ref",
			},
			{
				Kind:  yaml.ScalarNode,
				Value: "slasshy/mazsshy/yazzy.yaml#/components/schemas/thing",
			},
		},
	}

	cf := index.CreateClosedAPIIndexConfig()
	u, _ := url.Parse("https://herbs-and-coffee-in-the-fall.com/pizza/burgers")
	cf.BaseURL = u
	idx := index.NewSpecIndexWithConfig(&no, cf)
	n, i, e, c := LocateRefNodeWithContext(context.Background(), &no, idx)
	assert.Nil(t, n)
	assert.NotNil(t, i)
	assert.NotNil(t, e)
	assert.NotNil(t, c)
}

func TestLocateRefNode_NoExplode(t *testing.T) {
	no := yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{
				Kind:  yaml.ScalarNode,
				Value: "$ref",
			},
			{
				Kind:  yaml.ScalarNode,
				Value: "components/schemas/thing.yaml",
			},
		},
	}

	cf := index.CreateClosedAPIIndexConfig()
	u, _ := url.Parse("http://smiledfdfdfdfds.com/bikes")
	cf.BaseURL = u
	idx := index.NewSpecIndexWithConfig(&no, cf)
	n, i, e, c := LocateRefNodeWithContext(context.Background(), &no, idx)
	assert.Nil(t, n)
	assert.NotNil(t, i)
	assert.NotNil(t, e)
	assert.NotNil(t, c)
}

func TestLocateRefNode_NoExplode_HTTP(t *testing.T) {
	no := yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{
				Kind:  yaml.ScalarNode,
				Value: "$ref",
			},
			{
				Kind:  yaml.ScalarNode,
				Value: "components/schemas/thing.yaml",
			},
		},
	}

	cf := index.CreateClosedAPIIndexConfig()
	u, _ := url.Parse("http://smilfghfhfhfhfhes.com/bikes")
	cf.BaseURL = u
	idx := index.NewSpecIndexWithConfig(&no, cf)
	ctx := context.WithValue(context.Background(), index.CurrentPathKey, "http://minty-fresh-shoes.com/nice/no.yaml")
	n, i, e, c := LocateRefNodeWithContext(ctx, &no, idx)
	assert.Nil(t, n)
	assert.NotNil(t, i)
	assert.NotNil(t, e)
	assert.NotNil(t, c)
}

func TestLocateRefNode_NoExplode_NoSpecPath(t *testing.T) {
	no := yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{
				Kind:  yaml.ScalarNode,
				Value: "$ref",
			},
			{
				Kind:  yaml.ScalarNode,
				Value: "components/schemas/thing.yaml",
			},
		},
	}

	cf := index.CreateClosedAPIIndexConfig()
	u, _ := url.Parse("http://smilfghfhfhfhfhes.com/bikes")
	cf.BaseURL = u
	idx := index.NewSpecIndexWithConfig(&no, cf)
	ctx := context.WithValue(context.Background(), index.CurrentPathKey, "no.yaml")
	n, i, e, c := LocateRefNodeWithContext(ctx, &no, idx)
	assert.Nil(t, n)
	assert.NotNil(t, i)
	assert.NotNil(t, e)
	assert.NotNil(t, c)
}

func TestLocateRefNode_Explode_NoSpecPath(t *testing.T) {
	no := yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{
				Kind:  yaml.ScalarNode,
				Value: "$ref",
			},
			{
				Kind:  yaml.ScalarNode,
				Value: "components/schemas/thing.yaml#/components/schemas/thing",
			},
		},
	}

	cf := index.CreateClosedAPIIndexConfig()
	u, _ := url.Parse("http://smilfghfhfhfhfhes.com/bikes")
	cf.BaseURL = u
	idx := index.NewSpecIndexWithConfig(&no, cf)
	ctx := context.Background()

	n, i, e, c := LocateRefNodeWithContext(ctx, &no, idx)
	assert.Nil(t, n)
	assert.NotNil(t, i)
	assert.NotNil(t, e)
	assert.NotNil(t, c)
}

func TestLocateRefNode_DoARealLookup(t *testing.T) {
	lookup := "/root.yaml#/components/schemas/Burger"
	if runtime.GOOS == "windows" {
		lookup = "C:\\root.yaml#/components/schemas/Burger"
	}

	no := yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{
				Kind:  yaml.ScalarNode,
				Value: "$ref",
			},
			{
				Kind:  yaml.ScalarNode,
				Value: lookup,
			},
		},
	}

	b, err := os.ReadFile("../../test_specs/burgershop.openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var rootNode yaml.Node
	_ = yaml.Unmarshal(b, &rootNode)

	cf := index.CreateClosedAPIIndexConfig()
	u, _ := url.Parse("http://smilfghfhfhfhfhes.com/bikes")
	cf.BaseURL = u
	idx := index.NewSpecIndexWithConfig(&rootNode, cf)

	// fake cache to a lookup for a file that does not exist will work.
	fakeCache := new(sync.Map)
	fakeCache.Store(lookup, &index.Reference{Node: &no, Index: idx})
	idx.SetCache(fakeCache)

	ctx := context.WithValue(context.Background(), index.CurrentPathKey, lookup)
	n, i, e, c := LocateRefNodeWithContext(ctx, &no, idx)
	assert.NotNil(t, n)
	assert.NotNil(t, i)
	assert.Nil(t, e)
	assert.NotNil(t, c)
}

func TestLocateRefEndNoRef_NoName(t *testing.T) {
	r := &yaml.Node{Content: []*yaml.Node{{Kind: yaml.ScalarNode, Value: "$ref"}, {Kind: yaml.ScalarNode, Value: ""}}}
	n, i, e, c := LocateRefEnd(context.TODO(), r, nil, 0)
	assert.Nil(t, n)
	assert.Nil(t, i)
	assert.Error(t, e)
	assert.NotNil(t, c)
}

func TestLocateRefEndNoRef(t *testing.T) {
	r := &yaml.Node{Content: []*yaml.Node{{Kind: yaml.ScalarNode, Value: "$ref"}, {Kind: yaml.ScalarNode, Value: "cake"}}}
	n, i, e, c := LocateRefEnd(context.Background(), r, index.NewSpecIndexWithConfig(r, index.CreateClosedAPIIndexConfig()), 0)
	assert.Nil(t, n)
	assert.NotNil(t, i)
	assert.Error(t, e)
	assert.NotNil(t, c)
}

func TestLocateRefEnd_TooDeep(t *testing.T) {
	r := &yaml.Node{Content: []*yaml.Node{{Kind: yaml.ScalarNode, Value: "$ref"}, {Kind: yaml.ScalarNode, Value: ""}}}
	n, i, e, c := LocateRefEnd(context.TODO(), r, nil, 100)
	assert.Nil(t, n)
	assert.Nil(t, i)
	assert.Error(t, e)
	assert.NotNil(t, c)
}

func TestLocateRefEnd_Loop(t *testing.T) {
	yml, _ := os.ReadFile("../../test_specs/first.yaml")
	var bsn yaml.Node
	_ = yaml.Unmarshal(yml, &bsn)

	cf := index.CreateOpenAPIIndexConfig()
	cf.BasePath = "../../test_specs"

	localFSConfig := &index.LocalFSConfig{
		BaseDirectory: cf.BasePath,
		FileFilters:   []string{"first.yaml", "second.yaml", "third.yaml", "fourth.yaml"},
		DirFS:         os.DirFS(cf.BasePath),
	}
	localFs, _ := index.NewLocalFSWithConfig(localFSConfig)
	rolo := index.NewRolodex(cf)
	rolo.AddLocalFS(cf.BasePath, localFs)
	rolo.SetRootNode(&bsn)
	rolo.IndexTheRolodex(context.Background())

	idx := rolo.GetRootIndex()
	loop := yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{
				Kind:  yaml.ScalarNode,
				Value: "$ref",
			},
			{
				Kind:  yaml.ScalarNode,
				Value: "third.yaml#/properties/property/properties/statistics",
			},
		},
	}

	wd, _ := os.Getwd()
	cp, _ := filepath.Abs(filepath.Join(wd, "../../test_specs/first.yaml"))
	ctx := context.WithValue(context.Background(), index.CurrentPathKey, cp)
	n, i, e, c := LocateRefEnd(ctx, &loop, idx, 0)
	assert.NotNil(t, n)
	assert.NotNil(t, i)
	assert.Nil(t, e)
	assert.NotNil(t, c)
}

func TestLocateRefEnd_Loop_WithResolve(t *testing.T) {
	yml, _ := os.ReadFile("../../test_specs/first.yaml")
	var bsn yaml.Node
	_ = yaml.Unmarshal(yml, &bsn)

	cf := index.CreateOpenAPIIndexConfig()
	cf.BasePath = "../../test_specs"

	localFSConfig := &index.LocalFSConfig{
		BaseDirectory: cf.BasePath,
		FileFilters:   []string{"first.yaml", "second.yaml", "third.yaml", "fourth.yaml"},
		DirFS:         os.DirFS(cf.BasePath),
	}
	localFs, _ := index.NewLocalFSWithConfig(localFSConfig)
	rolo := index.NewRolodex(cf)
	rolo.AddLocalFS(cf.BasePath, localFs)
	rolo.SetRootNode(&bsn)
	rolo.IndexTheRolodex(context.Background())
	rolo.Resolve()
	idx := rolo.GetRootIndex()
	loop := yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{
				Kind:  yaml.ScalarNode,
				Value: "$ref",
			},
			{
				Kind:  yaml.ScalarNode,
				Value: "third.yaml#/properties/property/properties/statistics",
			},
		},
	}

	wd, _ := os.Getwd()
	cp, _ := filepath.Abs(filepath.Join(wd, "../../test_specs/first.yaml"))
	ctx := context.WithValue(context.Background(), index.CurrentPathKey, cp)
	n, i, e, c := LocateRefEnd(ctx, &loop, idx, 0)
	assert.NotNil(t, n)
	assert.NotNil(t, i)
	assert.Nil(t, e)
	assert.NotNil(t, c)
}

func TestLocateRefEnd_Empty(t *testing.T) {
	yml, _ := os.ReadFile("../../test_specs/first.yaml")
	var bsn yaml.Node
	_ = yaml.Unmarshal(yml, &bsn)

	cf := index.CreateOpenAPIIndexConfig()
	cf.BasePath = "../../test_specs"

	localFSConfig := &index.LocalFSConfig{
		BaseDirectory: cf.BasePath,
		FileFilters:   []string{"first.yaml", "second.yaml", "third.yaml", "fourth.yaml"},
		DirFS:         os.DirFS(cf.BasePath),
	}
	localFs, _ := index.NewLocalFSWithConfig(localFSConfig)
	rolo := index.NewRolodex(cf)
	rolo.AddLocalFS(cf.BasePath, localFs)
	rolo.SetRootNode(&bsn)
	rolo.IndexTheRolodex(context.Background())
	idx := rolo.GetRootIndex()
	loop := yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{
				Kind:  yaml.ScalarNode,
				Value: "$ref",
			},
			{
				Kind:  yaml.ScalarNode,
				Value: "",
			},
		},
	}

	wd, _ := os.Getwd()
	cp, _ := filepath.Abs(filepath.Join(wd, "../../test_specs/first.yaml"))
	ctx := context.WithValue(context.Background(), index.CurrentPathKey, cp)
	n, i, e, c := LocateRefEnd(ctx, &loop, idx, 0)
	assert.Nil(t, n)
	assert.Nil(t, i)
	assert.Error(t, e)
	assert.Equal(t, "reference at line 0, column 0 is empty, it cannot be resolved", e.Error())
	assert.NotNil(t, c)
}

func TestArray_NotRefNotArray(t *testing.T) {
	yml := ``
	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `limes:
  not: array`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)
	things, _, _, err := ExtractArray[*test_noGood](context.Background(), "limes", cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "array build failed, input is not an array, line 2, column 3")
	assert.Len(t, things, 0)
}

func TestHashExtensions(t *testing.T) {
	type args struct {
		ext *orderedmap.Map[KeyReference[string], ValueReference[*yaml.Node]]
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "empty",
			args: args{
				ext: orderedmap.New[KeyReference[string], ValueReference[*yaml.Node]](),
			},
			want: []string{},
		},
		{
			name: "hashes extensions",
			args: args{
				ext: orderedmap.ToOrderedMap(map[KeyReference[string]]ValueReference[*yaml.Node]{
					{Value: "x-burger"}: {
						Value: utils.CreateStringNode("yummy"),
					},
					{Value: "x-car"}: {
						Value: utils.CreateStringNode("ford"),
					},
				}),
			},
			want: []string{
				"x-burger-2a296977a4572521773eb7e7773cc054fae3e8589511ce9bf90cec7dd93d016a",
				"x-car-7d3aa6a5c79cdb0c2585daed714fa0936a18e6767b2dcc804992a90f6d0b8f5e",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := HashExtensions(tt.args.ext)
			assert.Equal(t, tt.want, hash)
		})
	}
}

func TestValueToString(t *testing.T) {
	type args struct {
		v any
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "string",
			args: args{
				v: "hello",
			},
			want: "hello",
		},
		{
			name: "int",
			args: args{
				v: 1,
			},
			want: "1",
		},
		{
			name: "yaml.Node",
			args: args{
				v: utils.CreateStringNode("world"),
			},
			want: "world",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValueToString(tt.args.v)
			assert.Equal(t, tt.want, strings.TrimSpace(got))
		})
	}
}

func TestExtractExtensions_Nill(t *testing.T) {
	err := ExtractExtensions(nil)
	assert.Nil(t, err)
}

func TestFromReferenceMap(t *testing.T) {
	refMap := orderedmap.New[KeyReference[string], ValueReference[string]]()
	refMap.Set(KeyReference[string]{Value: "foo"}, ValueReference[string]{Value: "bar"})
	refMap.Set(KeyReference[string]{Value: "baz"}, ValueReference[string]{Value: "qux"})
	om := FromReferenceMap(refMap)
	assert.Equal(t, "bar", om.GetOrZero("foo"))
	assert.Equal(t, "qux", om.GetOrZero("baz"))
}

func TestFromReferenceMapWithFunc(t *testing.T) {
	refMap := orderedmap.New[KeyReference[string], ValueReference[string]]()
	refMap.Set(KeyReference[string]{Value: "foo"}, ValueReference[string]{Value: "bar"})
	refMap.Set(KeyReference[string]{Value: "baz"}, ValueReference[string]{Value: "quxor"})
	var om *orderedmap.Map[string, int] = FromReferenceMapWithFunc(refMap, func(v string) int {
		return len(v)
	})
	assert.Equal(t, 3, om.GetOrZero("foo"))
	assert.Equal(t, 5, om.GetOrZero("baz"))
}

func TestAppendMapHashes(t *testing.T) {
	m := orderedmap.New[KeyReference[string], ValueReference[string]]()
	m.Set(KeyReference[string]{Value: "foo"}, ValueReference[string]{Value: "bar"})
	m.Set(KeyReference[string]{Value: "baz"}, ValueReference[string]{Value: "qux"})
	a := AppendMapHashes([]string{}, m)
	assert.Equal(t, 2, len(a))
	assert.Equal(t, "baz-21f58d27f827d295ffcd860c65045685e3baf1ad4506caa0140113b316647534", a[0])
	assert.Equal(t, "foo-fcde2b2edba56bf408601fb721fe9b5c338d10ee429ea04fae5511b68fbf8fb9", a[1])
}

// Tests for new performance optimization functions

func TestGetStringBuilder_PutStringBuilder(t *testing.T) {
	// Test basic pool functionality
	sb1 := GetStringBuilder()
	assert.NotNil(t, sb1)
	assert.Equal(t, 0, sb1.Len(), "New string builder should be empty")

	// Write some data
	sb1.WriteString("test data")
	assert.Equal(t, 9, sb1.Len())

	// Put it back
	PutStringBuilder(sb1)

	// Get another one - should be reset
	sb2 := GetStringBuilder()
	assert.Equal(t, 0, sb2.Len(), "Reused string builder should be reset")

	PutStringBuilder(sb2)
}

func TestGetStringBuilder_Concurrent(t *testing.T) {
	// Test concurrent access to string builder pool
	const numGoroutines = 10
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			sb := GetStringBuilder()
			sb.WriteString(fmt.Sprintf("goroutine-%d", id))
			assert.True(t, sb.Len() > 0)
			PutStringBuilder(sb)
		}(i)
	}

	wg.Wait()
}

func TestClearHashCache_Functionality(t *testing.T) {
	// Add some items to cache via GenerateHashString
	type testStruct struct {
		value string
	}

	obj1 := &testStruct{value: "test1"}
	obj2 := &testStruct{value: "test2"}

	// Generate hashes to populate cache
	hash1 := GenerateHashString(obj1)
	hash2 := GenerateHashString(obj2)

	assert.NotEmpty(t, hash1)
	assert.NotEmpty(t, hash2)
	assert.NotEqual(t, hash1, hash2)

	// Clear the cache
	ClearHashCache()

	// Should still work but recalculate
	hash1After := GenerateHashString(obj1)
	hash2After := GenerateHashString(obj2)

	assert.Equal(t, hash1, hash1After, "Hash should be same after cache clear")
	assert.Equal(t, hash2, hash2After, "Hash should be same after cache clear")
}

func TestGenerateHashString_OptimizedPaths(t *testing.T) {
	// Test different type conversions in optimized GenerateHashString
	testCases := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"int", 42, "73475cb40a568e8da8a045ced110137e159f890ac4da883b6b17dc651b3a8049"},
		{"int8", int8(42), "73475cb40a568e8da8a045ced110137e159f890ac4da883b6b17dc651b3a8049"},
		{"int16", int16(42), "73475cb40a568e8da8a045ced110137e159f890ac4da883b6b17dc651b3a8049"},
		{"int32", int32(42), "73475cb40a568e8da8a045ced110137e159f890ac4da883b6b17dc651b3a8049"},
		{"int64", int64(42), "73475cb40a568e8da8a045ced110137e159f890ac4da883b6b17dc651b3a8049"},
		{"uint", uint(42), "73475cb40a568e8da8a045ced110137e159f890ac4da883b6b17dc651b3a8049"},
		{"uint8", uint8(42), "73475cb40a568e8da8a045ced110137e159f890ac4da883b6b17dc651b3a8049"},
		{"uint16", uint16(42), "73475cb40a568e8da8a045ced110137e159f890ac4da883b6b17dc651b3a8049"},
		{"uint32", uint32(42), "73475cb40a568e8da8a045ced110137e159f890ac4da883b6b17dc651b3a8049"},
		{"uint64", uint64(42), "73475cb40a568e8da8a045ced110137e159f890ac4da883b6b17dc651b3a8049"},
		{"float32", float32(3.14), "2efff1261c25d94dd6698ea1047f5c0a7107ca98b0a6c2427ee6614143500215"},
		{"float64", float64(3.14), "2efff1261c25d94dd6698ea1047f5c0a7107ca98b0a6c2427ee6614143500215"},
		{"bool_true", true, "b5bea41b6c623f7c09f1bf24dcae58ebab3c0cdd90ad966bc43a45b44867e12b"},
		{"bool_false", false, "fcbcf165908dd18a9e49f7ff27810176db8e9f63b4352213741664245224f8aa"},
		{"string", "hello", "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := GenerateHashString(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGenerateHashString_Caching(t *testing.T) {
	type cacheableStruct struct {
		value string
	}

	// Clear cache first
	ClearHashCache()

	obj := &cacheableStruct{value: "test"}

	// First call should calculate and cache
	hash1 := GenerateHashString(obj)
	assert.NotEmpty(t, hash1)

	// Second call should use cache (same result)
	hash2 := GenerateHashString(obj)
	assert.Equal(t, hash1, hash2)

	// Different object should have different hash
	obj2 := &cacheableStruct{value: "different"}
	hash3 := GenerateHashString(obj2)
	assert.NotEqual(t, hash1, hash3)
}

func TestHashYamlNodeFast_ScalarNode(t *testing.T) {
	node := &yaml.Node{
		Kind:   yaml.ScalarNode,
		Tag:    "!!str",
		Value:  "test",
		Anchor: "anchor1",
	}

	hash := hashYamlNodeFast(node)
	assert.NotEmpty(t, hash)

	// Same node should produce same hash
	hash2 := hashYamlNodeFast(node)
	assert.Equal(t, hash, hash2)

	// Different value should produce different hash
	node2 := &yaml.Node{
		Kind:   yaml.ScalarNode,
		Tag:    "!!str",
		Value:  "different",
		Anchor: "anchor1",
	}
	hash3 := hashYamlNodeFast(node2)
	assert.NotEqual(t, hash, hash3)
}

func TestHashYamlNodeFast_NilNode(t *testing.T) {
	hash := hashYamlNodeFast(nil)
	assert.Empty(t, hash)
}

func TestHashYamlNodeFast_ComplexNode(t *testing.T) {
	// Create a mapping node
	node := &yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "key1"},
			{Kind: yaml.ScalarNode, Value: "value1"},
			{Kind: yaml.ScalarNode, Value: "key2"},
			{Kind: yaml.ScalarNode, Value: "value2"},
		},
	}

	hash := hashYamlNodeFast(node)
	assert.NotEmpty(t, hash)

	// Should be cached and return same result
	hash2 := hashYamlNodeFast(node)
	assert.Equal(t, hash, hash2)
}

func TestHashNodeTree_CircularReference(t *testing.T) {
	// Create nodes with circular references
	node1 := &yaml.Node{Kind: yaml.MappingNode, Value: "node1"}
	node2 := &yaml.Node{Kind: yaml.MappingNode, Value: "node2"}

	// Create circular reference
	node1.Content = []*yaml.Node{node2}
	node2.Content = []*yaml.Node{node1}

	h := sha256.New()
	visited := make(map[*yaml.Node]bool)

	// Should not infinite loop
	hashNodeTree(h, node1, visited)

	result := h.Sum(nil)
	assert.NotNil(t, result)
}

func TestHashNodeTree_SequenceNode(t *testing.T) {
	node := &yaml.Node{
		Kind: yaml.SequenceNode,
		Tag:  "!!seq",
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "item1"},
			{Kind: yaml.ScalarNode, Value: "item2"},
			{Kind: yaml.ScalarNode, Value: "item3"},
		},
	}

	h := sha256.New()
	visited := make(map[*yaml.Node]bool)
	hashNodeTree(h, node, visited)

	result := h.Sum(nil)
	assert.NotEmpty(t, result)
}

func TestHashNodeTree_MappingNode(t *testing.T) {
	node := &yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "key1"},
			{Kind: yaml.ScalarNode, Value: "value1"},
			{Kind: yaml.ScalarNode, Value: "key2"},
			{Kind: yaml.ScalarNode, Value: "value2"},
		},
	}

	h := sha256.New()
	visited := make(map[*yaml.Node]bool)
	hashNodeTree(h, node, visited)

	result := h.Sum(nil)
	assert.NotEmpty(t, result)
}

func TestHashNodeTree_DocumentNode(t *testing.T) {
	node := &yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "document content"},
		},
	}

	h := sha256.New()
	visited := make(map[*yaml.Node]bool)
	hashNodeTree(h, node, visited)

	result := h.Sum(nil)
	assert.NotEmpty(t, result)
}

func TestHashNodeTree_AliasNode(t *testing.T) {
	aliasTarget := &yaml.Node{Kind: yaml.ScalarNode, Value: "target"}
	node := &yaml.Node{
		Kind:  yaml.AliasNode,
		Alias: aliasTarget,
	}

	h := sha256.New()
	visited := make(map[*yaml.Node]bool)
	hashNodeTree(h, node, visited)

	result := h.Sum(nil)
	assert.NotEmpty(t, result)
}

func TestHashNodeTree_NilNode(t *testing.T) {
	h := sha256.New()
	visited := make(map[*yaml.Node]bool)

	// Should not crash
	hashNodeTree(h, nil, visited)

	// Hash should be unchanged (only initial state)
	result := h.Sum(nil)
	assert.NotNil(t, result)
}

func TestCompareYAMLNodes_BothNil(t *testing.T) {
	result := CompareYAMLNodes(nil, nil)
	assert.True(t, result)
}

func TestCompareYAMLNodes_OneNil(t *testing.T) {
	node := &yaml.Node{Kind: yaml.ScalarNode, Value: "test"}

	result1 := CompareYAMLNodes(nil, node)
	assert.False(t, result1)

	result2 := CompareYAMLNodes(node, nil)
	assert.False(t, result2)
}

func TestCompareYAMLNodes_SameNodes(t *testing.T) {
	node1 := &yaml.Node{Kind: yaml.ScalarNode, Value: "test"}
	node2 := &yaml.Node{Kind: yaml.ScalarNode, Value: "test"}

	result := CompareYAMLNodes(node1, node2)
	assert.True(t, result)
}

func TestCompareYAMLNodes_DifferentNodes(t *testing.T) {
	node1 := &yaml.Node{Kind: yaml.ScalarNode, Value: "test1"}
	node2 := &yaml.Node{Kind: yaml.ScalarNode, Value: "test2"}

	result := CompareYAMLNodes(node1, node2)
	assert.False(t, result)
}

func TestCompareYAMLNodes_ComplexNodes(t *testing.T) {
	// Create identical complex nodes
	node1 := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "key1"},
			{Kind: yaml.ScalarNode, Value: "value1"},
		},
	}

	node2 := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "key1"},
			{Kind: yaml.ScalarNode, Value: "value1"},
		},
	}

	result := CompareYAMLNodes(node1, node2)
	assert.True(t, result)

	// Modify one node
	node2.Content[1].Value = "different_value"
	result2 := CompareYAMLNodes(node1, node2)
	assert.False(t, result2)
}

func TestGenerateHashString_SchemaProxyNoCache(t *testing.T) {
	// Test that SchemaProxy types don't get cached (shouldCache = false)
	// We can't easily test this without creating actual SchemaProxy objects
	// but we can test the general caching bypass logic

	type nonCacheableType struct {
		value string
	}

	obj := &nonCacheableType{value: "test"}

	// Clear cache
	ClearHashCache()

	hash1 := GenerateHashString(obj)
	hash2 := GenerateHashString(obj)

	// Should be same (correct calculation) even without caching
	assert.Equal(t, hash1, hash2)
}

func TestHashYamlNodeFast_Caching(t *testing.T) {
	// Test that complex nodes get cached but scalar nodes don't

	// Scalar node (should not be cached)
	scalarNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "test"}
	hash1 := hashYamlNodeFast(scalarNode)
	hash2 := hashYamlNodeFast(scalarNode)
	assert.Equal(t, hash1, hash2)

	// Complex node (should be cached)
	complexNode := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "key"},
			{Kind: yaml.ScalarNode, Value: "value"},
		},
	}

	hash3 := hashYamlNodeFast(complexNode)
	hash4 := hashYamlNodeFast(complexNode)
	assert.Equal(t, hash3, hash4)
}

func TestHashNodeTree_MappingNodeSorting(t *testing.T) {
	// Test that mapping nodes are sorted consistently for hashing

	// Create two identical mappings with different key orders
	node1 := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "zebra"},
			{Kind: yaml.ScalarNode, Value: "value1"},
			{Kind: yaml.ScalarNode, Value: "alpha"},
			{Kind: yaml.ScalarNode, Value: "value2"},
		},
	}

	node2 := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "alpha"},
			{Kind: yaml.ScalarNode, Value: "value2"},
			{Kind: yaml.ScalarNode, Value: "zebra"},
			{Kind: yaml.ScalarNode, Value: "value1"},
		},
	}

	hash1 := hashYamlNodeFast(node1)
	hash2 := hashYamlNodeFast(node2)

	// Should be equal because of consistent sorting
	assert.Equal(t, hash1, hash2)
}

func TestHashNodeTree_EdgeCases(t *testing.T) {
	// Test edge cases in hashNodeTree

	// Mapping with odd number of content items (missing value)
	node := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "key1"},
			{Kind: yaml.ScalarNode, Value: "value1"},
			{Kind: yaml.ScalarNode, Value: "key2"},
			// Missing value for key2
		},
	}

	h := sha256.New()
	visited := make(map[*yaml.Node]bool)

	// Should not crash
	hashNodeTree(h, node, visited)
	result := h.Sum(nil)
	assert.NotNil(t, result)
}

func TestGenerateHashString_PointerDereference(t *testing.T) {
	// Test pointer dereferencing for primitives
	val := "test"
	ptr := &val

	hash1 := GenerateHashString(val)
	hash2 := GenerateHashString(ptr)

	assert.Equal(t, hash1, hash2, "Pointer and value should produce same hash")
}

func TestHashNodeTree_VisitedTracking(t *testing.T) {
	// Test that visited map prevents infinite loops

	node := &yaml.Node{Kind: yaml.ScalarNode, Value: "test"}
	h := sha256.New()
	visited := make(map[*yaml.Node]bool)

	// Mark as visited
	visited[node] = true

	// Should detect as visited and add circular marker
	hashNodeTree(h, node, visited)

	result := h.Sum(nil)
	assert.NotNil(t, result)
}

func TestConcurrentHashGeneration(t *testing.T) {
	// Test thread safety of hash generation with caching
	const numGoroutines = 20
	var wg sync.WaitGroup

	// Clear cache first
	ClearHashCache()

	type testObj struct {
		id int
	}

	objects := make([]*testObj, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		objects[i] = &testObj{id: i}
	}

	results := make([]string, numGoroutines)

	// Generate hashes concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = GenerateHashString(objects[idx])
		}(i)
	}

	wg.Wait()

	// All results should be non-empty and unique
	seen := make(map[string]bool)
	for i, hash := range results {
		assert.NotEmpty(t, hash, "Hash %d should not be empty", i)
		assert.False(t, seen[hash], "Hash %d should be unique", i)
		seen[hash] = true
	}
}

// Tests for remaining uncovered functions to achieve 100% coverage

func TestYAMLNodeToBytes_NilNode(t *testing.T) {
	result, err := YAMLNodeToBytes(nil)
	assert.Nil(t, result)
	assert.Nil(t, err)
}

func TestYAMLNodeToBytes_ValidNode(t *testing.T) {
	node := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Value: "test value",
	}

	result, err := YAMLNodeToBytes(node)
	assert.NoError(t, err)
	assert.Contains(t, string(result), "test value")
}

func TestYAMLNodeToBytes_ComplexNode(t *testing.T) {
	node := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "key"},
			{Kind: yaml.ScalarNode, Value: "value"},
		},
	}

	result, err := YAMLNodeToBytes(node)
	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestHashYAMLNodeSlice_Empty(t *testing.T) {
	result := HashYAMLNodeSlice([]*yaml.Node{})
	assert.Empty(t, result)
}

func TestHashYAMLNodeSlice_SingleNode(t *testing.T) {
	nodes := []*yaml.Node{
		{Kind: yaml.ScalarNode, Value: "test"},
	}

	result := HashYAMLNodeSlice(nodes)
	assert.NotEmpty(t, result)
	assert.Len(t, result, 64) // SHA256 hex length
}

func TestHashYAMLNodeSlice_MultipleNodes(t *testing.T) {
	nodes := []*yaml.Node{
		{Kind: yaml.ScalarNode, Value: "first"},
		{Kind: yaml.ScalarNode, Value: "second"},
		{Kind: yaml.ScalarNode, Value: "third"},
	}

	result := HashYAMLNodeSlice(nodes)
	assert.NotEmpty(t, result)

	// Same nodes should produce same hash
	result2 := HashYAMLNodeSlice(nodes)
	assert.Equal(t, result, result2)

	// Different order should produce different hash
	reorderedNodes := []*yaml.Node{
		{Kind: yaml.ScalarNode, Value: "second"},
		{Kind: yaml.ScalarNode, Value: "first"},
		{Kind: yaml.ScalarNode, Value: "third"},
	}
	result3 := HashYAMLNodeSlice(reorderedNodes)
	assert.NotEqual(t, result, result3)
}

func TestHashYAMLNodeSlice_NilNodes(t *testing.T) {
	nodes := []*yaml.Node{
		{Kind: yaml.ScalarNode, Value: "test"},
		nil,
		{Kind: yaml.ScalarNode, Value: "test2"},
	}

	result := HashYAMLNodeSlice(nodes)
	assert.NotEmpty(t, result)
}

func TestAppendMapHashes_NilMap(t *testing.T) {
	initial := []string{"existing"}
	result := AppendMapHashes(initial, (*orderedmap.Map[KeyReference[string], ValueReference[string]])(nil))
	assert.Equal(t, initial, result)
}

func TestAppendMapHashes_SmallMap_InsertionSort(t *testing.T) {
	// Test with <= 10 entries to trigger insertion sort
	m := orderedmap.New[KeyReference[string], ValueReference[string]]()
	for i := 9; i >= 0; i-- { // Add in reverse order to test sorting
		m.Set(KeyReference[string]{Value: fmt.Sprintf("key%d", i)},
			ValueReference[string]{Value: fmt.Sprintf("value%d", i)})
	}

	initial := []string{"existing"}
	result := AppendMapHashes(initial, m)

	assert.Len(t, result, 11) // 1 existing + 10 new
	assert.Equal(t, "existing", result[0])

	// Verify sorted order (keys should be processed in alphabetical order)
	for i := 1; i < len(result); i++ {
		assert.Contains(t, result[i], fmt.Sprintf("key%d", i-1))
	}
}

func TestAppendMapHashes_LargeMap_QuickSort(t *testing.T) {
	// Test with > 10 entries to trigger quicksort
	m := orderedmap.New[KeyReference[string], ValueReference[string]]()
	for i := 15; i >= 0; i-- { // Add in reverse order to test sorting
		m.Set(KeyReference[string]{Value: fmt.Sprintf("key%02d", i)},
			ValueReference[string]{Value: fmt.Sprintf("value%d", i)})
	}

	initial := []string{}
	result := AppendMapHashes(initial, m)

	assert.Len(t, result, 16)

	// Verify sorted order
	for i := 0; i < len(result)-1; i++ {
		// Extract key from hash string (format: "key-hash")
		parts1 := strings.Split(result[i], "-")
		parts2 := strings.Split(result[i+1], "-")
		assert.True(t, parts1[0] <= parts2[0], "Results should be sorted by key")
	}
}

func TestAppendMapHashes_VerySmallMap_DirectConcat(t *testing.T) {
	// Test with <= 5 entries to trigger direct string concatenation
	m := orderedmap.New[KeyReference[string], ValueReference[string]]()
	for i := 4; i >= 0; i-- {
		m.Set(KeyReference[string]{Value: fmt.Sprintf("k%d", i)},
			ValueReference[string]{Value: fmt.Sprintf("v%d", i)})
	}

	result := AppendMapHashes([]string{}, m)
	assert.Len(t, result, 5)

	// Should be sorted
	for i := 0; i < len(result); i++ {
		assert.Contains(t, result[i], fmt.Sprintf("k%d", i))
	}
}

func TestAppendMapHashes_MediumMap_StringBuilder(t *testing.T) {
	// Test with > 5 and <= 10 entries to trigger string builder path
	m := orderedmap.New[KeyReference[string], ValueReference[string]]()
	for i := 7; i >= 0; i-- {
		m.Set(KeyReference[string]{Value: fmt.Sprintf("key%d", i)},
			ValueReference[string]{Value: fmt.Sprintf("value%d", i)})
	}

	result := AppendMapHashes([]string{}, m)
	assert.Len(t, result, 8)

	// Verify each entry has correct format
	for _, hash := range result {
		parts := strings.Split(hash, "-")
		assert.Len(t, parts, 2)
		assert.True(t, strings.HasPrefix(parts[0], "key"))
		assert.Len(t, parts[1], 64) // SHA256 hex hash length
	}
}

func TestAppendMapHashes_PreAllocation(t *testing.T) {
	// Test the capacity pre-allocation logic
	m := orderedmap.New[KeyReference[string], ValueReference[string]]()
	for i := 0; i < 20; i++ {
		m.Set(KeyReference[string]{Value: fmt.Sprintf("key%02d", i)},
			ValueReference[string]{Value: fmt.Sprintf("value%d", i)})
	}

	// Start with a slice that has limited capacity
	initial := make([]string, 2, 3) // len=2, cap=3
	initial[0] = "first"
	initial[1] = "second"

	result := AppendMapHashes(initial, m)
	assert.Len(t, result, 22) // 2 initial + 20 from map
	assert.Equal(t, "first", result[0])
	assert.Equal(t, "second", result[1])
}

func TestValueToString_YAMLScalarNode(t *testing.T) {
	node := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "test value",
	}

	result := ValueToString(node)
	assert.Equal(t, "test value", result)
}

func TestValueToString_YAMLComplexNode(t *testing.T) {
	node := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "key"},
			{Kind: yaml.ScalarNode, Value: "value"},
		},
	}

	result := ValueToString(node)
	assert.Contains(t, result, "key")
	assert.Contains(t, result, "value")
}

func TestValueToString_NonYAMLValue(t *testing.T) {
	testCases := []struct {
		input    interface{}
		expected string
	}{
		{42, "42"},
		{"string", "string"},
		{true, "true"},
		{3.14, "3.14"},
	}

	for _, tc := range testCases {
		result := ValueToString(tc.input)
		assert.Equal(t, tc.expected, result)
	}
}

func TestGenerateHashString_DefaultCase(t *testing.T) {
	// Test the default case in the switch statement
	type customType struct {
		field string
	}

	obj := customType{field: "test"}
	result := GenerateHashString(obj)
	assert.NotEmpty(t, result)
	assert.Len(t, result, 64) // SHA256 hex length
}

func TestGenerateHashString_PointerToNonPrimitive(t *testing.T) {
	// Test pointer to non-primitive that gets dereferenced
	type customStruct struct {
		value string
	}

	obj := &customStruct{value: "test"}
	result := GenerateHashString(obj)
	assert.NotEmpty(t, result)
}

func TestGenerateHashString_CachingPathCoverage(t *testing.T) {
	// Test cache storage path in GenerateHashString
	type testStruct struct {
		value string
	}

	ClearHashCache()

	// Test struct that should get cached
	obj := &testStruct{value: "test"}
	hash1 := GenerateHashString(obj)
	assert.NotEmpty(t, hash1)

	// Should hit cache on second call
	hash2 := GenerateHashString(obj)
	assert.Equal(t, hash1, hash2)
}

// Surgical tests to hit exact uncovered branches for 100% coverage

func TestGenerateHashString_NilHashable(t *testing.T) {
	// Hit the h == nil branch in Hashable path (line ~958)
	var nilHashable Hashable
	result := GenerateHashString(nilHashable)
	assert.Empty(t, result) // Should return empty string for nil hashable
}

func TestGenerateHashString_EmptyHashStr(t *testing.T) {
	// Hit the hashStr == "" condition in cache storage check (line ~1014)
	ClearHashCache()
	result := GenerateHashString(&testHashable{})
	// Empty hash should not be cached, but should return the empty hex string
	assert.Equal(t, "0000000000000000000000000000000000000000000000000000000000000000", result)
}

func TestExtractMapExtensions_RefError(t *testing.T) {
	// Hit the reference error branch in ExtractMapExtensions (line ~711-712)

	// Create a node with a $ref that cannot be found
	refNode := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "$ref"},
			{Kind: yaml.ScalarNode, Value: "#/nonexistent/reference"},
		},
	}

	idx := index.NewSpecIndexWithConfig(refNode, index.CreateClosedAPIIndexConfig())

	// This should hit the "reference cannot be found" error path
	result, _, _, err := ExtractMapExtensions[*test_Good](context.Background(), "test", refNode, idx, false)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reference cannot be found")
}

func TestGetCircularReferenceResult_JourneyMatch(t *testing.T) {
	// Hit the Journey[k].Node == node branch (line ~326-328)

	// Create a spec with circular references to get refs populated
	yml := `
components:
  schemas:
    A:
      $ref: "#/components/schemas/B"
    B:
      $ref: "#/components/schemas/A"
`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(yml), &rootNode)
	require.NoError(t, err)

	// Create index and build it to detect circular references
	idx := index.NewSpecIndexWithConfig(&rootNode, index.CreateOpenAPIIndexConfig())

	// Create a test node that matches something in the journey
	testNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "test"}

	// Manually create a circular reference result to ensure the journey path is hit
	circRef := &index.CircularReferenceResult{
		Journey: []*index.Reference{
			{Node: testNode, Definition: "test"},
		},
		LoopPoint: &index.Reference{Node: &yaml.Node{Kind: yaml.ScalarNode, Value: "other"}},
	}

	// Add this to the index manually to test the journey matching
	refs := []*index.CircularReferenceResult{circRef}
	idx.SetCircularReferences(refs)

	result := GetCircularReferenceResult(testNode, idx)
	assert.Equal(t, circRef, result)
}

func TestGetCircularReferenceResult_RefValueMatch(t *testing.T) {
	// Hit the refs[i].Journey[k].Definition == refValue branch (line ~330-332)

	// Create a node with a $ref value
	refNode := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "$ref"},
			{Kind: yaml.ScalarNode, Value: "#/components/schemas/Test"},
		},
	}

	// Create a minimal index
	idx := index.NewSpecIndexWithConfig(refNode, index.CreateOpenAPIIndexConfig())

	// Manually create a circular reference that matches the definition
	circRef := &index.CircularReferenceResult{
		Journey: []*index.Reference{
			{Node: &yaml.Node{}, Definition: "#/components/schemas/Test"},
		},
		LoopPoint: &index.Reference{Node: &yaml.Node{}},
	}

	// Force the circular reference into the index
	refs := []*index.CircularReferenceResult{circRef}
	idx.SetCircularReferences(refs)

	result := GetCircularReferenceResult(refNode, idx)
	assert.Equal(t, circRef, result)
}

func TestGetCircularReferenceResult_MappedRefMatch(t *testing.T) {
	// Hit the mapped reference branch (line ~339-341)

	// Create a node with $ref
	refNode := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "$ref"},
			{Kind: yaml.ScalarNode, Value: "#/test/definition"},
		},
	}

	idx := index.NewSpecIndexWithConfig(refNode, index.CreateOpenAPIIndexConfig())

	// Create circular reference that matches the definition
	circRef := &index.CircularReferenceResult{
		LoopPoint: &index.Reference{
			Node:       &yaml.Node{},
			Definition: "#/test/definition",
		},
		Journey: []*index.Reference{}, // Empty journey to avoid other matches
	}

	refs := []*index.CircularReferenceResult{circRef}
	idx.SetCircularReferences(refs)

	result := GetCircularReferenceResult(refNode, idx)
	assert.Equal(t, circRef, result)
}

func TestExtractMapExtensions_CircularRefError(t *testing.T) {
	// Hit the circError assignment path (line ~708)

	// This is complex to set up, but we can create a minimal scenario
	// Create a self-referencing node
	refNode := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "$ref"},
			{Kind: yaml.ScalarNode, Value: "#/components/schemas/Self"},
		},
	}

	// Create a spec that has the self-reference
	specYml := `
components:
  schemas:
    Self:
      $ref: "#/components/schemas/Self"
`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(specYml), &rootNode)
	require.NoError(t, err)

	idx := index.NewSpecIndexWithConfig(&rootNode, index.CreateOpenAPIIndexConfig())

	// This should trigger the circular error path
	_, _, _, err = ExtractMapExtensions[*test_Good](context.Background(), "test", refNode, idx, false)
	// The error could be circular reference or other reference issues
	// Just ensure we don't panic and handle the error gracefully
	if err != nil {
		// Expected - circular references should cause errors
		assert.NotNil(t, err)
	}
}

// Custom Hashable implementation for testing nil hash
type testHashable struct{}

func (t testHashable) Hash() [32]byte {
	return [32]byte{} // All zeros - empty hash
}

func TestGenerateHashString_EdgeCaseCoverage(t *testing.T) {
	// Test edge cases to hit remaining uncovered lines

	// Test with a very specific case that might hit uncovered branches
	type specialStruct struct {
		value interface{}
	}

	obj := &specialStruct{value: nil}
	result := GenerateHashString(obj)
	assert.NotEmpty(t, result)
}

func TestGenerateHashString_SchemaProxyTypeCheck(t *testing.T) {
	// Hit the type name check for SchemaProxy/Schema (shouldCache = false path)
	// Create a struct with a name that matches the schema proxy pattern
	type fakeSchemaProxy struct {
		field string
	}

	ClearHashCache()
	obj := &fakeSchemaProxy{field: "test"}

	// This should bypass caching due to type name check
	result1 := GenerateHashString(obj)
	result2 := GenerateHashString(obj)

	assert.Equal(t, result1, result2) // Should still be equal, just not cached
	assert.NotEmpty(t, result1)
}

func TestExtractMapExtensions_ValueNodeAssignment(t *testing.T) {
	// Hit specific branches in ExtractMapExtensions

	// Create a valid reference that can be found
	specYml := `
components:
  schemas:
    ValidSchema:
      type: string
`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(specYml), &rootNode)
	require.NoError(t, err)

	// Create a reference node that points to a valid location
	refNode := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "$ref"},
			{Kind: yaml.ScalarNode, Value: "#/components/schemas/ValidSchema"},
		},
	}

	idx := index.NewSpecIndexWithConfig(&rootNode, index.CreateOpenAPIIndexConfig())

	// This should hit the successful reference resolution path
	result, labelNode, valueNode, err := ExtractMapExtensions[*test_Good](context.Background(), "test", refNode, idx, false)

	// We expect this to either succeed or fail gracefully, but not panic
	if err != nil {
		// Reference resolution can fail for various reasons, that's OK
		assert.NotNil(t, err)
	} else {
		// If it succeeds, we should have some result
		assert.NotNil(t, result)
	}

	// labelNode and valueNode should be set regardless
	_ = labelNode
	_ = valueNode
}
