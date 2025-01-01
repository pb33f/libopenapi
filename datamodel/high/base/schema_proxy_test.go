// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
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

	_ = os.WriteFile("schema.yaml", []byte(ymlComponents), 0777)
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

	indexingError := rolodex.IndexTheRolodex()
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

	_ = os.WriteFile("schema_n.yaml", []byte(ymlComponents), 0777)
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

	indexingError := rolodex.IndexTheRolodex()
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
	rErr := rolodex.IndexTheRolodex()

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
