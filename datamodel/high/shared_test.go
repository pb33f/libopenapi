// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package high

import (
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestExtractExtensions(t *testing.T) {
	n := orderedmap.New[low.KeyReference[string], low.ValueReference[*yaml.Node]]()
	n.Set(low.KeyReference[string]{
		Value: "pb33f",
	}, low.ValueReference[*yaml.Node]{
		Value: utils.CreateStringNode("new cowboy in town"),
	})
	ext := ExtractExtensions(n)

	var pb33f string
	err := ext.GetOrZero("pb33f").Decode(&pb33f)
	require.NoError(t, err)

	assert.Equal(t, "new cowboy in town", pb33f)
}

type textExtension struct {
	Cowboy string
	Power  int
}

type parent struct {
	low *child
}

func (p *parent) GoLow() *child {
	return p.low
}

type child struct {
	Extensions *orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]]
}

func (c *child) GetExtensions() *orderedmap.Map[low.KeyReference[string], low.ValueReference[*yaml.Node]] {
	return c.Extensions
}

func TestUnpackExtensions(t *testing.T) {
	var resultA, resultB yaml.Node

	ymlA := `
cowboy: buckaroo
power: 100`

	ymlB := `
cowboy: frogman
power: 2`

	err := yaml.Unmarshal([]byte(ymlA), &resultA)
	assert.NoError(t, err)
	err = yaml.Unmarshal([]byte(ymlB), &resultB)
	assert.NoError(t, err)

	n := orderedmap.New[low.KeyReference[string], low.ValueReference[*yaml.Node]]()
	n.Set(low.KeyReference[string]{
		Value: "x-rancher-a",
	}, low.ValueReference[*yaml.Node]{
		ValueNode: resultA.Content[0],
	})

	n.Set(low.KeyReference[string]{
		Value: "x-rancher-b",
	}, low.ValueReference[*yaml.Node]{
		ValueNode: resultB.Content[0],
	})

	c := new(child)
	c.Extensions = n

	p := new(parent)
	p.low = c

	res, err := UnpackExtensions[textExtension, *child](p)
	assert.NoError(t, err)
	assert.NotEmpty(t, res)
	assert.Equal(t, "buckaroo", res.GetOrZero("x-rancher-a").Cowboy)
	assert.Equal(t, 100, res.GetOrZero("x-rancher-a").Power)
	assert.Equal(t, "frogman", res.GetOrZero("x-rancher-b").Cowboy)
	assert.Equal(t, 2, res.GetOrZero("x-rancher-b").Power)
}

func TestUnpackExtensions_Fail(t *testing.T) {
	var resultA, resultB yaml.Node

	ymlA := `
cowboy: buckaroo
power: 100`

	// this is incorrect types, unpacking will fail.
	ymlB := `
cowboy: 0
power: hello`

	err := yaml.Unmarshal([]byte(ymlA), &resultA)
	assert.NoError(t, err)
	err = yaml.Unmarshal([]byte(ymlB), &resultB)
	assert.NoError(t, err)

	n := orderedmap.New[low.KeyReference[string], low.ValueReference[*yaml.Node]]()
	n.Set(low.KeyReference[string]{
		Value: "x-rancher-a",
	}, low.ValueReference[*yaml.Node]{
		ValueNode: resultA.Content[0],
	})

	n.Set(low.KeyReference[string]{
		Value: "x-rancher-b",
	}, low.ValueReference[*yaml.Node]{
		ValueNode: resultB.Content[0],
	})

	c := new(child)
	c.Extensions = n

	p := new(parent)
	p.low = c

	res, er := UnpackExtensions[textExtension, *child](p)
	assert.Error(t, er)
	assert.Empty(t, res)
}

func TestRenderInline(t *testing.T) {
	// Create a simple struct to test rendering
	type testStruct struct {
		Name    string `yaml:"name,omitempty"`
		Version string `yaml:"version,omitempty"`
	}

	high := &testStruct{Name: "test", Version: "1.0.0"}
	result, err := RenderInline(high, nil)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify the result is a yaml.Node
	node, ok := result.(*yaml.Node)
	require.True(t, ok)
	assert.Equal(t, yaml.MappingNode, node.Kind)
}

func TestRenderInline_WithLow(t *testing.T) {
	// Test with both high and low models (typical use case)
	type testStruct struct {
		Name string `yaml:"name,omitempty"`
	}

	high := &testStruct{Name: "test"}
	low := &testStruct{Name: "low-test"} // low model, should be passed through

	result, err := RenderInline(high, low)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRenderInlineWithContext(t *testing.T) {
	// Create a simple struct to test rendering with context
	type testStruct struct {
		Name    string `yaml:"name,omitempty"`
		Version string `yaml:"version,omitempty"`
	}

	high := &testStruct{Name: "test", Version: "1.0.0"}
	// Pass a mock context (any type is accepted)
	ctx := struct{}{}
	result, err := RenderInlineWithContext(high, nil, ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify the result is a yaml.Node
	node, ok := result.(*yaml.Node)
	require.True(t, ok)
	assert.Equal(t, yaml.MappingNode, node.Kind)
}

func TestRenderInlineWithContext_WithLow(t *testing.T) {
	// Test with both high and low models and context
	type testStruct struct {
		Name string `yaml:"name,omitempty"`
	}

	high := &testStruct{Name: "test"}
	low := &testStruct{Name: "low-test"}
	ctx := struct{}{}

	result, err := RenderInlineWithContext(high, low, ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

// mockExternalRefResolver is a mock implementation of ExternalRefResolver for testing
type mockExternalRefResolver struct {
	isRef    bool
	ref      string
	indexVal *index.SpecIndex
}

func (m *mockExternalRefResolver) IsReference() bool {
	return m.isRef
}

func (m *mockExternalRefResolver) GetReference() string {
	return m.ref
}

func (m *mockExternalRefResolver) GetIndex() *index.SpecIndex {
	return m.indexVal
}

func TestResolveExternalRef_NilLowObj(t *testing.T) {
	result, err := ResolveExternalRef[string, string](nil, nil, nil)

	assert.NoError(t, err)
	assert.False(t, result.Resolved)
}

func TestResolveExternalRef_NotAReference(t *testing.T) {
	mock := &mockExternalRefResolver{isRef: false}

	result, err := ResolveExternalRef[string, string](mock, nil, nil)

	assert.NoError(t, err)
	assert.False(t, result.Resolved)
}

func TestResolveExternalRef_NilIndex(t *testing.T) {
	mock := &mockExternalRefResolver{isRef: true, ref: "#/components/test", indexVal: nil}

	result, err := ResolveExternalRef[string, string](mock, nil, nil)

	assert.NoError(t, err)
	assert.False(t, result.Resolved)
}

func TestRenderExternalRef_NilLowObj(t *testing.T) {
	result, err := RenderExternalRef[string, string](nil, nil, nil)

	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestRenderExternalRef_NotAReference(t *testing.T) {
	mock := &mockExternalRefResolver{isRef: false}

	result, err := RenderExternalRef[string, string](mock, nil, nil)

	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestRenderExternalRefWithContext_NilLowObj(t *testing.T) {
	ctx := struct{}{}
	result, err := RenderExternalRefWithContext[string, string](nil, nil, nil, ctx)

	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestRenderExternalRefWithContext_NotAReference(t *testing.T) {
	mock := &mockExternalRefResolver{isRef: false}
	ctx := struct{}{}

	result, err := RenderExternalRefWithContext[string, string](mock, nil, nil, ctx)

	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestResolveExternalRef_ComponentNotFound(t *testing.T) {
	// Create a real index with no components
	config := index.CreateOpenAPIIndexConfig()
	idx := index.NewSpecIndexWithConfig(nil, config)

	mock := &mockExternalRefResolver{isRef: true, ref: "#/components/schemas/NotFound", indexVal: idx}

	result, err := ResolveExternalRef[string, string](mock, nil, nil)

	assert.NoError(t, err)
	assert.False(t, result.Resolved)
}

func TestRenderExternalRef_ComponentNotFound(t *testing.T) {
	// Create a real index with no components
	config := index.CreateOpenAPIIndexConfig()
	idx := index.NewSpecIndexWithConfig(nil, config)

	mock := &mockExternalRefResolver{isRef: true, ref: "#/components/schemas/NotFound", indexVal: idx}

	result, err := RenderExternalRef[string, string](mock, nil, nil)

	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestRenderExternalRefWithContext_ComponentNotFound(t *testing.T) {
	// Create a real index with no components
	config := index.CreateOpenAPIIndexConfig()
	idx := index.NewSpecIndexWithConfig(nil, config)

	mock := &mockExternalRefResolver{isRef: true, ref: "#/components/schemas/NotFound", indexVal: idx}
	ctx := struct{}{}

	result, err := RenderExternalRefWithContext[string, string](mock, nil, nil, ctx)

	assert.NoError(t, err)
	assert.Nil(t, result)
}

// testLow is a simple low-level type for testing
type testLow struct {
	Name string `yaml:"name"`
}

// testHigh is a simple high-level type for testing
type testHigh struct {
	Name string `yaml:"name"`
}

func TestResolveExternalRef_Success(t *testing.T) {
	// Create a spec with a component
	spec := `openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    TestSchema:
      type: object
      properties:
        name:
          type: string`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(spec), &rootNode)
	require.NoError(t, err)

	config := index.CreateOpenAPIIndexConfig()
	idx := index.NewSpecIndexWithConfig(&rootNode, config)

	mock := &mockExternalRefResolver{isRef: true, ref: "#/components/schemas/TestSchema", indexVal: idx}

	buildLow := func(node *yaml.Node, idx *index.SpecIndex) (*testLow, error) {
		var l testLow
		err := node.Decode(&l)
		return &l, err
	}

	buildHigh := func(l *testLow) *testHigh {
		return &testHigh{Name: l.Name}
	}

	result, err := ResolveExternalRef(mock, buildLow, buildHigh)

	assert.NoError(t, err)
	assert.True(t, result.Resolved)
	assert.NotNil(t, result.High)
	assert.NotNil(t, result.Low)
}

func TestResolveExternalRef_BuildLowError(t *testing.T) {
	// Create a spec with a component
	spec := `openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    TestSchema:
      type: object`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(spec), &rootNode)
	require.NoError(t, err)

	config := index.CreateOpenAPIIndexConfig()
	idx := index.NewSpecIndexWithConfig(&rootNode, config)

	mock := &mockExternalRefResolver{isRef: true, ref: "#/components/schemas/TestSchema", indexVal: idx}

	buildLow := func(node *yaml.Node, idx *index.SpecIndex) (*testLow, error) {
		return nil, assert.AnError // Return an error
	}

	buildHigh := func(l *testLow) *testHigh {
		return &testHigh{}
	}

	result, err := ResolveExternalRef(mock, buildLow, buildHigh)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to build resolved external reference")
	assert.False(t, result.Resolved)
}

func TestRenderExternalRef_Success(t *testing.T) {
	// Create a spec with a component
	spec := `openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    TestSchema:
      type: object
      properties:
        name:
          type: string`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(spec), &rootNode)
	require.NoError(t, err)

	config := index.CreateOpenAPIIndexConfig()
	idx := index.NewSpecIndexWithConfig(&rootNode, config)

	mock := &mockExternalRefResolver{isRef: true, ref: "#/components/schemas/TestSchema", indexVal: idx}

	buildLow := func(node *yaml.Node, idx *index.SpecIndex) (*testLow, error) {
		var l testLow
		err := node.Decode(&l)
		return &l, err
	}

	buildHigh := func(l *testLow) *testHigh {
		return &testHigh{Name: l.Name}
	}

	result, err := RenderExternalRef(mock, buildLow, buildHigh)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRenderExternalRefWithContext_Success(t *testing.T) {
	// Create a spec with a component
	spec := `openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    TestSchema:
      type: object
      properties:
        name:
          type: string`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(spec), &rootNode)
	require.NoError(t, err)

	config := index.CreateOpenAPIIndexConfig()
	idx := index.NewSpecIndexWithConfig(&rootNode, config)

	mock := &mockExternalRefResolver{isRef: true, ref: "#/components/schemas/TestSchema", indexVal: idx}
	ctx := struct{}{}

	buildLow := func(node *yaml.Node, idx *index.SpecIndex) (*testLow, error) {
		var l testLow
		err := node.Decode(&l)
		return &l, err
	}

	buildHigh := func(l *testLow) *testHigh {
		return &testHigh{Name: l.Name}
	}

	result, err := RenderExternalRefWithContext(mock, buildLow, buildHigh, ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRenderExternalRef_BuildLowError(t *testing.T) {
	// Create a spec with a component
	spec := `openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    TestSchema:
      type: object`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(spec), &rootNode)
	require.NoError(t, err)

	config := index.CreateOpenAPIIndexConfig()
	idx := index.NewSpecIndexWithConfig(&rootNode, config)

	mock := &mockExternalRefResolver{isRef: true, ref: "#/components/schemas/TestSchema", indexVal: idx}

	buildLow := func(node *yaml.Node, idx *index.SpecIndex) (*testLow, error) {
		return nil, assert.AnError // Return an error
	}

	buildHigh := func(l *testLow) *testHigh {
		return &testHigh{}
	}

	result, err := RenderExternalRef(mock, buildLow, buildHigh)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestRenderExternalRefWithContext_BuildLowError(t *testing.T) {
	// Create a spec with a component
	spec := `openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    TestSchema:
      type: object`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(spec), &rootNode)
	require.NoError(t, err)

	config := index.CreateOpenAPIIndexConfig()
	idx := index.NewSpecIndexWithConfig(&rootNode, config)

	mock := &mockExternalRefResolver{isRef: true, ref: "#/components/schemas/TestSchema", indexVal: idx}
	ctx := struct{}{}

	buildLow := func(node *yaml.Node, idx *index.SpecIndex) (*testLow, error) {
		return nil, assert.AnError // Return an error
	}

	buildHigh := func(l *testLow) *testHigh {
		return &testHigh{}
	}

	result, err := RenderExternalRefWithContext(mock, buildLow, buildHigh, ctx)

	assert.Error(t, err)
	assert.Nil(t, result)
}
