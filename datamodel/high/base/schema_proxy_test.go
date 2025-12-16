// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestSchemaProxy_MarshalYAML(t *testing.T) {
	const ymlComponents = `components:
    schemas:
     rice:
       type: string
     nice:
       properties:
         rice:
           $ref: '#/components/schemas/rice'
     ice:
       properties:
         rice:
           $ref: '#/components/schemas/rice'`

	idx := func() *index.SpecIndex {
		var idxNode yaml.Node
		err := yaml.Unmarshal([]byte(ymlComponents), &idxNode)
		assert.NoError(t, err)
		return index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())
	}()

	const ref = "#/components/schemas/nice"
	const ymlSchema = `$ref: '` + ref + `'`
	var node yaml.Node
	_ = yaml.Unmarshal([]byte(ymlSchema), &node)

	lowProxy := new(lowbase.SchemaProxy)
	err := lowProxy.Build(context.Background(), nil, node.Content[0], idx)
	assert.NoError(t, err)

	lowRef := low.NodeReference[*lowbase.SchemaProxy]{
		Value: lowProxy,
	}

	sp := NewSchemaProxy(&lowRef)

	origin := sp.GetReferenceOrigin()
	assert.Nil(t, origin)

	rend, _ := sp.Render()
	assert.Equal(t, "$ref: '#/components/schemas/nice'", strings.TrimSpace(string(rend)))
}

func TestCreateSchemaProxy_Fail(t *testing.T) {
	proxy := &SchemaProxy{}
	assert.Nil(t, proxy.Schema())
}

func TestCreateSchemaProxy(t *testing.T) {
	sp := CreateSchemaProxy(&Schema{Description: "iAmASchema"})
	assert.Equal(t, "iAmASchema", sp.rendered.Description)
	assert.False(t, sp.IsReference())
	assert.Nil(t, sp.GetValueNode())
}

func TestCreateSchemaProxy_NoNilValue(t *testing.T) {
	sp := CreateSchemaProxy(&Schema{Description: "iAmASchema"})
	sp.Schema()

	// jerry rig the test.
	nodeRef := low.NodeReference[*lowbase.SchemaProxy]{}
	nodeRef.ValueNode = &yaml.Node{}
	sp.schema = &nodeRef

	assert.Equal(t, "iAmASchema", sp.rendered.Description)
	assert.NotNil(t, sp.GetValueNode())
}

func TestCreateSchemaProxyRef(t *testing.T) {
	sp := CreateSchemaProxyRef("#/components/schemas/MySchema")
	assert.Equal(t, "#/components/schemas/MySchema", sp.GetReference())
	assert.True(t, sp.IsReference())
}

func TestSchemaProxy_GetReference(t *testing.T) {
	refNode := utils.CreateStringNode("#/components/schemas/MySchema")

	ref := low.Reference{}
	ref.SetReference("#/components/schemas/MySchema", refNode)

	sp := &SchemaProxy{
		schema: &low.NodeReference[*lowbase.SchemaProxy]{
			Value: &lowbase.SchemaProxy{
				Reference: ref,
			},
		},
	}
	assert.Equal(t, "#/components/schemas/MySchema", sp.GetReference())
	assert.Equal(t, refNode, sp.GetReferenceNode())
}

func TestSchemaProxy_IsReference_Nil(t *testing.T) {
	var sp *SchemaProxy
	assert.False(t, sp.IsReference())
}

func TestSchemaProxy_NoSchema_GetOrigin(t *testing.T) {
	sp := &SchemaProxy{}
	assert.Nil(t, sp.GetReferenceOrigin())
}

func TestCreateSchemaProxyRef_GetReferenceNode(t *testing.T) {
	refNode := utils.CreateRefNode("#/components/schemas/MySchema")

	sp := CreateSchemaProxyRef("#/components/schemas/MySchema")
	assert.Equal(t, refNode, sp.GetReferenceNode())
}

func TestCreateRefNode_MarshalYAML(t *testing.T) {
	ref := low.Reference{}
	ref.SetReference("#/components/schemas/MySchema", nil)

	sp := &SchemaProxy{
		schema: &low.NodeReference[*lowbase.SchemaProxy]{
			Value: &lowbase.SchemaProxy{
				Reference: ref,
			},
		},
	}
	node, err := sp.MarshalYAML()
	require.NoError(t, err)
	assert.Equal(t, node, utils.CreateRefNode("#/components/schemas/MySchema"))
}

func TestSchemaProxy_MarshalYAML_InlineCircular(t *testing.T) {
	const ymlComponents = `openapi: 3.1
components:
  schemas:
    spice:
      properties:
        ice:
          $ref: '#/components/schemas/nice'
    nice:
      properties:
        rice:
          $ref: '#/components/schemas/nice'`

	idx := func() *index.SpecIndex {
		var idxNode yaml.Node
		err := yaml.Unmarshal([]byte(ymlComponents), &idxNode)
		assert.NoError(t, err)
		return index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())
	}()

	resolver := index.NewResolver(idx)
	resolver.CheckForCircularReferences()

	const ymlSchema = `properties:
  rice:
    $ref: '#/components/schemas/nice'`
	var node yaml.Node
	_ = yaml.Unmarshal([]byte(ymlSchema), &node)

	lowProxy := new(lowbase.SchemaProxy)
	err := lowProxy.Build(context.Background(), &node, node.Content[0], idx)
	assert.NoError(t, err)

	lowRef := low.NodeReference[*lowbase.SchemaProxy]{
		Value:   lowProxy,
		KeyNode: &node,
	}

	spEmpty := NewSchemaProxy(nil)
	assert.Nil(t, spEmpty.GetSchemaKeyNode())

	sp := NewSchemaProxy(&lowRef)
	assert.NotNil(t, sp.GetSchemaKeyNode())

	rend, _ := sp.MarshalYAMLInline()
	assert.NotNil(t, rend)
}

func TestSchemaProxy_MarshalYAML_IgnoredCircular(t *testing.T) {
	const ymlComponents = `openapi: 3.1
components:
  schemas:
    dice:
      properties:
        mice:
          anyOf:
            - $ref: '#/components/schemas/nice'
  
    spice:
      properties:
        ice:
          allOf:
            - $ref: '#/components/schemas/dice'
    nice:
      allOf:
        - $ref: '#/components/schemas/spice'
      properties:
        rice:
          oneOf: 
            - $ref: '#/components/schemas/nice'`

	idx := func() *index.SpecIndex {
		var idxNode yaml.Node
		err := yaml.Unmarshal([]byte(ymlComponents), &idxNode)
		assert.NoError(t, err)

		cfg := index.CreateOpenAPIIndexConfig()
		cfg.IgnoreArrayCircularReferences = true
		cfg.IgnorePolymorphicCircularReferences = true
		return index.NewSpecIndexWithConfig(&idxNode, cfg)
	}()

	resolver := index.NewResolver(idx)
	resolver.CheckForCircularReferences()

	const ymlSchema = `items:
  $ref: '#/components/schemas/nice'`
	var node yaml.Node
	_ = yaml.Unmarshal([]byte(ymlSchema), &node)

	lowProxy := new(lowbase.SchemaProxy)
	err := lowProxy.Build(context.Background(), &node, node.Content[0], idx)
	ref := low.Reference{}
	ref.SetReference("#/components/schemas/spice", node.Content[0])
	lowProxy.Reference = ref

	assert.NoError(t, err)

	lowRef := low.NodeReference[*lowbase.SchemaProxy]{
		Value:     lowProxy,
		KeyNode:   node.Content[0],
		Reference: ref,
	}

	spEmpty := NewSchemaProxy(nil)
	assert.Nil(t, spEmpty.GetSchemaKeyNode())

	sp := NewSchemaProxy(&lowRef)
	assert.NotNil(t, sp.GetSchemaKeyNode())

	rend, _ := sp.MarshalYAMLInline()
	assert.NotNil(t, rend)
}

func TestSchemaProxy_MarshalYAML_MatchBasePath(t *testing.T) {
	const ymlComponents = `properties:
  spice:
    allOf:
      - $ref: '#/properties/rice'
  rice:
    oneOf: 
      - $ref: './schema.yaml'`

	_ = os.WriteFile("schema.yaml", []byte(ymlComponents), 0o777)
	defer os.RemoveAll("schema.yaml")

	actualYaml := []byte("$ref: 'schema.yaml'")

	cwd, _ := os.Getwd()
	basePath := cwd

	// create an index config
	config := index.CreateOpenAPIIndexConfig()
	rolodex := index.NewRolodex(config)

	fsCfg := &index.LocalFSConfig{
		BaseDirectory: basePath,
		IndexConfig:   config,
	}

	fileFS, err := index.NewLocalFSWithConfig(fsCfg)
	if err != nil {
		panic(err)
	}

	var rootNode yaml.Node
	_ = yaml.Unmarshal(actualYaml, &rootNode)

	rolodex.SetRootNode(&rootNode)

	rolodex.AddLocalFS(basePath, fileFS)

	indexingError := rolodex.IndexTheRolodex(context.Background())
	if indexingError != nil {
		panic(indexingError)
	}

	rolodex.Resolve()

	// there should be no errors at this point
	resolvingErrors := rolodex.GetCaughtErrors()
	if resolvingErrors != nil {
		panic(resolvingErrors)
	}

	lowProxy := new(lowbase.SchemaProxy)
	err = lowProxy.Build(context.Background(), &rootNode, rootNode.Content[0], rolodex.GetRootIndex())
	assert.NoError(t, err)
	ref := low.Reference{}

	ref.SetReference("#/properties/spice", rootNode.Content[0])
	lowProxy.Reference = ref
	lowProxy.GetIndex().SetAbsolutePath(filepath.Join(basePath, "schema.yaml"))

	assert.NoError(t, err)

	lowRef := low.NodeReference[*lowbase.SchemaProxy]{
		Value:     lowProxy,
		KeyNode:   rootNode.Content[0],
		Reference: ref,
	}

	spEmpty := NewSchemaProxy(nil)
	assert.Nil(t, spEmpty.GetSchemaKeyNode())
	sp := NewSchemaProxy(&lowRef)

	assert.NotNil(t, sp.GetSchemaKeyNode())

	rend, _ := sp.MarshalYAMLInline()
	assert.NotNil(t, rend)
}

func TestSchemaProxy_MarshalYAML_StripBasePath(t *testing.T) {
	const ymlComponents = `properties:
  spice:
    allOf:
      - $ref: '#/properties/rice'
  rice:
    oneOf: 
      - $ref: './schema_n.yaml'`

	_ = os.WriteFile("schema_n.yaml", []byte(ymlComponents), 0o777)
	defer os.RemoveAll("schema_n.yaml")

	actualYaml := []byte("$ref: './schema_n.yaml'")

	cwd, _ := os.Getwd()
	basePath := cwd

	// create an index config
	config := index.CreateOpenAPIIndexConfig()
	rolodex := index.NewRolodex(config)

	fsCfg := &index.LocalFSConfig{
		BaseDirectory: basePath,
		IndexConfig:   config,
	}

	fileFS, err := index.NewLocalFSWithConfig(fsCfg)
	if err != nil {
		panic(err)
	}

	var rootNode yaml.Node
	_ = yaml.Unmarshal(actualYaml, &rootNode)

	rolodex.SetRootNode(&rootNode)

	rolodex.AddLocalFS(basePath, fileFS)

	indexingError := rolodex.IndexTheRolodex(context.Background())
	if indexingError != nil {
		panic(indexingError)
	}

	// there should be no errors at this point
	resolvingErrors := rolodex.GetCaughtErrors()
	if resolvingErrors != nil {
		panic(resolvingErrors)
	}

	lowProxy := new(lowbase.SchemaProxy)
	err = lowProxy.Build(context.Background(), &rootNode, rootNode.Content[0], rolodex.GetRootIndex())
	assert.NoError(t, err)
	ref := low.Reference{}

	assert.NoError(t, err)

	lowRef := low.NodeReference[*lowbase.SchemaProxy]{
		Value:     lowProxy,
		KeyNode:   rootNode.Content[0],
		Reference: ref,
	}

	sp := NewSchemaProxy(&lowRef)

	assert.NotNil(t, sp.GetSchemaKeyNode())

	ref.SetReference("./schema_n.yaml", rootNode.Content[0])
	lowProxy.Reference = ref

	rend, _ := sp.MarshalYAMLInline()
	assert.NotNil(t, rend)

	// should not have rendered and should be the same as the input
	// check by hashing.
	assert.Equal(t, index.HashNode(rootNode.Content[0]), index.HashNode(rend.(*yaml.Node)))
}

func TestSchemaProxy_MarshalYAML_BadSchema(t *testing.T) {
	actualYaml := []byte("$ref: './schema_k.yaml'")

	var rootNode yaml.Node
	_ = yaml.Unmarshal(actualYaml, &rootNode)

	lowRef := low.NodeReference[*lowbase.SchemaProxy]{
		KeyNode: rootNode.Content[0],
	}

	sp := NewSchemaProxy(&lowRef)
	rend, err := sp.MarshalYAMLInline()
	assert.Nil(t, rend)
	assert.Error(t, err)
}

func TestSchemaProxy_MarshalYAML_Inline_HTTP(t *testing.T) {
	// this triggers http code by fudging references, found when importing from URLs directly.

	first := `type: object
properties:
  cakes:
    type: array
    items: 
      $ref: 'http#/properties/cakes'`

	var rootNode yaml.Node
	_ = yaml.Unmarshal([]byte(first), &rootNode)

	cf := index.CreateOpenAPIIndexConfig()
	cf.SkipDocumentCheck = true

	rolodex := index.NewRolodex(cf)
	rolodex.SetRootNode(&rootNode)
	rErr := rolodex.IndexTheRolodex(context.Background())

	assert.NoError(t, rErr)

	circularRefs := []*index.CircularReferenceResult{
		{
			LoopPoint: &index.Reference{
				Definition:     "#/components/schemas/Ten",
				FullDefinition: "http#/properties/cakes",
			},
		},
	}

	rolodex.GetRootIndex().SetCircularReferences(circularRefs)

	lowProxy := new(lowbase.SchemaProxy)
	err := lowProxy.Build(context.Background(), nil, rootNode.Content[0], rolodex.GetRootIndex())
	assert.NoError(t, err)

	lowRef := low.NodeReference[*lowbase.SchemaProxy]{
		Value: lowProxy,
	}

	sp := NewSchemaProxy(&lowRef)

	rend, _ := sp.Schema().Properties.GetOrZero("cakes").Schema().Items.A.MarshalYAMLInline()
	assert.NotNil(t, rend)
}

func TestSchemaProxy_ConcurrentCacheAccess(t *testing.T) {
	// Create schema that will be cached
	const ymlComponents = `components:
  schemas:
    TestSchema:
      type: object`

	var idxNode yaml.Node
	err := yaml.Unmarshal([]byte(ymlComponents), &idxNode)
	assert.NoError(t, err)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())

	// Create multiple proxies that will share the same cache entry
	proxies := make([]*SchemaProxy, 10)
	for i := range proxies {
		const ymlSchema = `$ref: '#/components/schemas/TestSchema'`
		var node yaml.Node
		yaml.Unmarshal([]byte(ymlSchema), &node)

		lowProxy := new(lowbase.SchemaProxy)
		lowProxy.Build(context.Background(), nil, node.Content[0], idx)

		proxies[i] = NewSchemaProxy(&low.NodeReference[*lowbase.SchemaProxy]{
			Value: lowProxy, ValueNode: node.Content[0],
		})
	}

	// Trigger race by having all proxies access Schema() simultaneously
	var wg sync.WaitGroup
	for _, proxy := range proxies {
		wg.Add(1)
		go func(p *SchemaProxy) {
			defer wg.Done()
			schema := p.Schema() // This should trigger the race with old code
			assert.NotNil(t, schema)
			// Check if ParentProxy is set - with our fix, cached schemas may not have it
			if schema.ParentProxy == nil {
				t.Logf("Warning: Schema ParentProxy is nil for cached schema")
			}
		}(proxy)
	}
	wg.Wait()
}

func TestSchemaProxy_ParentProxyPreservedForCachedSchemas(t *testing.T) {
	const ymlComponents = `components:
  schemas:
    TestSchema:
      type: object
      properties:
        name:
          type: string`

	var idxNode yaml.Node
	err := yaml.Unmarshal([]byte(ymlComponents), &idxNode)
	assert.NoError(t, err)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())

	const ymlSchema = `$ref: '#/components/schemas/TestSchema'`
	var node1 yaml.Node
	yaml.Unmarshal([]byte(ymlSchema), &node1)

	lowProxy1 := new(lowbase.SchemaProxy)
	lowProxy1.Build(context.Background(), nil, node1.Content[0], idx)

	proxy1 := NewSchemaProxy(&low.NodeReference[*lowbase.SchemaProxy]{
		Value: lowProxy1, ValueNode: node1.Content[0],
	})

	schema1 := proxy1.Schema()
	assert.NotNil(t, schema1)
	assert.Equal(t, proxy1, schema1.ParentProxy, "First schema should have correct ParentProxy")

	var node2 yaml.Node
	yaml.Unmarshal([]byte(ymlSchema), &node2)

	lowProxy2 := new(lowbase.SchemaProxy)
	lowProxy2.Build(context.Background(), nil, node2.Content[0], idx)

	proxy2 := NewSchemaProxy(&low.NodeReference[*lowbase.SchemaProxy]{
		Value: lowProxy2, ValueNode: node2.Content[0],
	})

	schema2 := proxy2.Schema()
	assert.NotNil(t, schema2)
	assert.Equal(t, proxy2, schema2.ParentProxy, "Second schema should have its own ParentProxy, not the first proxy's")
	assert.NotEqual(t, schema1.ParentProxy, schema2.ParentProxy, "Different proxies should have different parent relationships")
}

func TestSetBundlingMode(t *testing.T) {
	// First, reset to known state by decrementing until we hit 0
	// This handles any leftover state from parallel tests
	for IsBundlingMode() {
		SetBundlingMode(false)
	}
	assert.False(t, IsBundlingMode(), "Bundling mode should be false initially")

	// Toggle on
	SetBundlingMode(true)
	assert.True(t, IsBundlingMode(), "Bundling mode should be true after setting")

	// Toggle off
	SetBundlingMode(false)
	assert.False(t, IsBundlingMode(), "Bundling mode should be false after unsetting")

	// Test multiple increments (nested bundling)
	SetBundlingMode(true)
	SetBundlingMode(true)
	assert.True(t, IsBundlingMode(), "Bundling mode should be true with count=2")

	SetBundlingMode(false)
	assert.True(t, IsBundlingMode(), "Bundling mode should still be true with count=1")

	SetBundlingMode(false)
	assert.False(t, IsBundlingMode(), "Bundling mode should be false with count=0")
}

func TestSetPreserveReference(t *testing.T) {
	proxy := CreateSchemaProxyRef("#/components/schemas/Pet")
	proxy.SetPreserveReference(true)

	// Verify flag is set via marshaling behavior
	result, err := proxy.MarshalYAML()
	require.NoError(t, err)

	node, ok := result.(*yaml.Node)
	require.True(t, ok)
	// Should render as $ref
	assert.Equal(t, yaml.MappingNode, node.Kind)
	assert.Equal(t, 2, len(node.Content))
	assert.Equal(t, "$ref", node.Content[0].Value)
	assert.Equal(t, "#/components/schemas/Pet", node.Content[1].Value)
}

func TestSetPreserveReference_WithBundlingMode(t *testing.T) {
	// First, reset to known state
	for IsBundlingMode() {
		SetBundlingMode(false)
	}

	// Test interaction with bundling mode
	SetBundlingMode(true)
	defer SetBundlingMode(false)

	proxy := CreateSchemaProxyRef("#/components/schemas/Test")

	// Without SetPreserveReference, should still render as ref in bundling mode
	result, err := proxy.MarshalYAML()
	require.NoError(t, err)
	node := result.(*yaml.Node)
	assert.Equal(t, "$ref", node.Content[0].Value)
}

func TestSetPreserveReference_MarshalYAMLInline(t *testing.T) {
	// Test that SetPreserveReference affects MarshalYAMLInline behavior
	proxy := CreateSchemaProxyRef("#/components/schemas/Pet")
	proxy.SetPreserveReference(true)

	// MarshalYAMLInline should return ref node when preserveReference is true
	result, err := proxy.MarshalYAMLInline()
	require.NoError(t, err)

	node, ok := result.(*yaml.Node)
	require.True(t, ok)
	// Should render as $ref
	assert.Equal(t, yaml.MappingNode, node.Kind)
	assert.Equal(t, 2, len(node.Content))
	assert.Equal(t, "$ref", node.Content[0].Value)
	assert.Equal(t, "#/components/schemas/Pet", node.Content[1].Value)
}

func TestSetPreserveReference_MarshalYAMLInline_NilRefNode(t *testing.T) {
	// Test the fallback path when GetReferenceNode returns nil
	// This happens when the proxy has refStr but no backing schema
	proxy := &SchemaProxy{
		refStr:            "#/components/schemas/Test",
		preserveReference: true,
		lock:              &sync.Mutex{},
	}

	result, err := proxy.MarshalYAMLInline()
	require.NoError(t, err)

	node, ok := result.(*yaml.Node)
	require.True(t, ok)
	// Should create a ref node using utils.CreateRefNode fallback
	assert.Equal(t, yaml.MappingNode, node.Kind)
	assert.Equal(t, "$ref", node.Content[0].Value)
	assert.Equal(t, "#/components/schemas/Test", node.Content[1].Value)
}

func TestMarshalYAMLInline_BundlingMode_PreservesLocalComponentRefs(t *testing.T) {
	// Test that in bundling mode, local #/components/ refs pointing to schemas
	// in the same root document are preserved (not inlined).
	// This covers lines 356-373 in schema_proxy.go

	// Reset bundling mode state
	for IsBundlingMode() {
		SetBundlingMode(false)
	}

	// Create a document with components
	const ymlComponents = `components:
  schemas:
    Pet:
      type: object
      properties:
        name:
          type: string
    Dog:
      type: object
      properties:
        pet:
          $ref: '#/components/schemas/Pet'`

	var idxNode yaml.Node
	err := yaml.Unmarshal([]byte(ymlComponents), &idxNode)
	require.NoError(t, err)

	// Create spec index and rolodex
	cfg := index.CreateOpenAPIIndexConfig()
	idx := index.NewSpecIndexWithConfig(&idxNode, cfg)

	// Create a rolodex and set it up - use SetRootIndex to make this the root
	rolodex := index.NewRolodex(cfg)
	rolodex.SetRootNode(&idxNode)
	rolodex.SetRootIndex(idx)
	idx.SetRolodex(rolodex)

	// Build a schema proxy referencing #/components/schemas/Pet
	const ref = "#/components/schemas/Pet"
	const ymlSchema = `$ref: '` + ref + `'`
	var node yaml.Node
	_ = yaml.Unmarshal([]byte(ymlSchema), &node)

	lowProxy := new(lowbase.SchemaProxy)
	err = lowProxy.Build(context.Background(), nil, node.Content[0], idx)
	require.NoError(t, err)

	lowRef := low.NodeReference[*lowbase.SchemaProxy]{
		Value:     lowProxy,
		ValueNode: node.Content[0],
	}

	sp := NewSchemaProxy(&lowRef)

	// Enable bundling mode
	SetBundlingMode(true)
	defer SetBundlingMode(false)

	// MarshalYAMLInline should preserve the ref since the schema is in the root index
	result, err := sp.MarshalYAMLInline()
	require.NoError(t, err)

	resultNode, ok := result.(*yaml.Node)
	require.True(t, ok)

	// Should render as $ref
	assert.Equal(t, yaml.MappingNode, resultNode.Kind)
	require.GreaterOrEqual(t, len(resultNode.Content), 2)
	assert.Equal(t, "$ref", resultNode.Content[0].Value)
	assert.Equal(t, ref, resultNode.Content[1].Value)
}

func TestMarshalYAMLInline_CircularReferenceDetection_WithReference(t *testing.T) {
	// Test that circular reference detection returns the ref node and error
	// when a reference proxy is already being rendered.
	// This covers lines 388-390 in schema_proxy.go

	// Reset bundling mode state
	for IsBundlingMode() {
		SetBundlingMode(false)
	}

	// Create a schema proxy with a refStr to generate a render key
	ref := "#/components/schemas/CircularTest"
	proxy := &SchemaProxy{
		refStr: ref,
		lock:   &sync.Mutex{},
	}

	// Pre-load the render key to simulate being mid-render (cycle detected)
	renderKey := proxy.getInlineRenderKey()
	require.NotEmpty(t, renderKey, "render key should be generated from refStr")

	// Store the key in the tracker to simulate a cycle
	inlineRenderingTracker.Store(renderKey, true)
	defer inlineRenderingTracker.Delete(renderKey)

	// MarshalYAMLInline should detect the cycle and return ref node + error
	result, err := proxy.MarshalYAMLInline()

	// Should return an error about circular reference
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circular reference")
	assert.Contains(t, err.Error(), ref)

	// Result should be a ref node (fallback for reference proxy)
	resultNode, ok := result.(*yaml.Node)
	require.True(t, ok)
	assert.Equal(t, yaml.MappingNode, resultNode.Kind)
	assert.Equal(t, "$ref", resultNode.Content[0].Value)
}

func TestMarshalYAMLInline_CircularReferenceDetection_WithoutReference(t *testing.T) {
	// Test that circular reference detection returns an empty map node and error
	// when an inline (non-reference) proxy is already being rendered.
	// This covers lines 392-394 in schema_proxy.go

	// Reset bundling mode state
	for IsBundlingMode() {
		SetBundlingMode(false)
	}

	// Create an inline schema proxy (no refStr, with value node for position)
	valueNode := &yaml.Node{
		Kind:   yaml.MappingNode,
		Line:   10,
		Column: 5,
	}

	lowProxy := &lowbase.SchemaProxy{}

	lowRef := low.NodeReference[*lowbase.SchemaProxy]{
		Value:     lowProxy,
		ValueNode: valueNode,
	}

	proxy := NewSchemaProxy(&lowRef)

	// Get the render key (should be position-based for inline schemas)
	renderKey := proxy.getInlineRenderKey()
	require.NotEmpty(t, renderKey, "render key should be generated from node position")

	// Store the key in the tracker to simulate a cycle
	inlineRenderingTracker.Store(renderKey, true)
	defer inlineRenderingTracker.Delete(renderKey)

	// MarshalYAMLInline should detect the cycle and return empty map + error
	result, err := proxy.MarshalYAMLInline()

	// Should return an error about circular reference
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circular reference")
	assert.Contains(t, err.Error(), "inline rendering")

	// Result should be an empty mapping node (fallback for inline schemas)
	resultNode, ok := result.(*yaml.Node)
	require.True(t, ok)
	assert.Equal(t, yaml.MappingNode, resultNode.Kind)
	assert.Equal(t, "!!map", resultNode.Tag)
}

func TestGetInlineRenderKey_ReferenceWithoutIndex(t *testing.T) {
	// Test line 324: return ref when IsReference() is true but index is nil
	// This covers the path where we have a reference but no index to get the path from
	// Need schema.Value to be non-nil but GetIndex() to return nil

	// Create a low-level proxy that is a reference but has no index
	lowProxy := &lowbase.SchemaProxy{}
	lowProxy.SetReference("#/components/schemas/TestSchema", nil)
	// Don't call Build() so index stays nil

	lowRef := low.NodeReference[*lowbase.SchemaProxy]{
		Value: lowProxy,
	}

	proxy := &SchemaProxy{
		schema: &lowRef,
		lock:   &sync.Mutex{},
	}

	renderKey := proxy.getInlineRenderKey()

	// Should return just the ref since there's no index
	assert.Equal(t, "#/components/schemas/TestSchema", renderKey)
}

func TestGetInlineRenderKey_NilSchemaReturnsRefStr(t *testing.T) {
	// Test line 312: return refStr when schema is nil
	// This covers the early return path

	proxy := &SchemaProxy{
		refStr: "#/components/schemas/EarlyReturn",
		lock:   &sync.Mutex{},
	}

	renderKey := proxy.getInlineRenderKey()

	// Should return refStr via early return path
	assert.Equal(t, "#/components/schemas/EarlyReturn", renderKey)
}

func TestGetInlineRenderKey_NilSchemaValueReturnsRefStr(t *testing.T) {
	// Test line 312: return refStr when schema.Value is nil
	// This covers the early return path

	lowRef := low.NodeReference[*lowbase.SchemaProxy]{
		Value: nil, // nil value
	}

	proxy := &SchemaProxy{
		refStr: "#/components/schemas/AnotherEarlyReturn",
		schema: &lowRef,
		lock:   &sync.Mutex{},
	}

	renderKey := proxy.getInlineRenderKey()

	// Should return refStr via early return path since schema.Value is nil
	assert.Equal(t, "#/components/schemas/AnotherEarlyReturn", renderKey)
}

func TestMarshalYAMLInline_PreserveReference_ViaLowLevel(t *testing.T) {
	// Test preserveReference path when reference is set via low-level proxy
	// (refStr is empty, so GetReferenceNode uses low-level path)

	lowProxy := &lowbase.SchemaProxy{}
	lowProxy.SetReference("#/components/schemas/TestRef", nil)

	lowRef := low.NodeReference[*lowbase.SchemaProxy]{
		Value: lowProxy,
	}

	proxy := &SchemaProxy{
		schema:            &lowRef,
		preserveReference: true,
		lock:              &sync.Mutex{},
	}

	result, err := proxy.MarshalYAMLInline()
	require.NoError(t, err)

	node, ok := result.(*yaml.Node)
	require.True(t, ok)
	assert.Equal(t, yaml.MappingNode, node.Kind)
	assert.Equal(t, "$ref", node.Content[0].Value)
	assert.Equal(t, "#/components/schemas/TestRef", node.Content[1].Value)
}

func TestMarshalYAMLInline_BundlingMode_ViaLowLevelRef(t *testing.T) {
	// Test bundling mode preserves refs when schema is in root index
	// Reference is set via low-level proxy (not refStr)

	// Reset bundling mode state
	for IsBundlingMode() {
		SetBundlingMode(false)
	}

	// Create a minimal spec with components
	const ymlComponents = `components:
  schemas:
    TestSchema:
      type: object`

	var idxNode yaml.Node
	err := yaml.Unmarshal([]byte(ymlComponents), &idxNode)
	require.NoError(t, err)

	cfg := index.CreateOpenAPIIndexConfig()
	idx := index.NewSpecIndexWithConfig(&idxNode, cfg)

	// Create rolodex and set as root
	rolodex := index.NewRolodex(cfg)
	rolodex.SetRootNode(&idxNode)
	rolodex.SetRootIndex(idx)
	idx.SetRolodex(rolodex)

	// Create a low-level proxy using Build with the ref
	const ref = "#/components/schemas/TestSchema"
	const ymlRef = `$ref: '` + ref + `'`
	var refNode yaml.Node
	_ = yaml.Unmarshal([]byte(ymlRef), &refNode)

	lowProxy := &lowbase.SchemaProxy{}
	err = lowProxy.Build(context.Background(), nil, refNode.Content[0], idx)
	require.NoError(t, err)

	lowRef := low.NodeReference[*lowbase.SchemaProxy]{
		Value: lowProxy,
	}

	proxy := &SchemaProxy{
		schema: &lowRef,
		lock:   &sync.Mutex{},
	}

	// Enable bundling mode
	SetBundlingMode(true)
	defer SetBundlingMode(false)

	result, err := proxy.MarshalYAMLInline()
	require.NoError(t, err)

	node, ok := result.(*yaml.Node)
	require.True(t, ok)
	assert.Equal(t, yaml.MappingNode, node.Kind)
	assert.Equal(t, "$ref", node.Content[0].Value)
	assert.Equal(t, "#/components/schemas/TestSchema", node.Content[1].Value)
}
