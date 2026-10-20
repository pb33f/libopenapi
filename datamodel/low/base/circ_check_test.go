// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package base

import (
	"context"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"testing"
)

func TestCheckSchemaProxyForCircularRefs(t *testing.T) {

	rolo := index.NewRolodex(&index.SpecIndexConfig{})
	dummyNode := &yaml.Node{Content: []*yaml.Node{{Content: []*yaml.Node{}}}}

	ref := low.Reference{}
	ref.SetReference("minty-fresh", dummyNode)

	schema := &SchemaProxy{
		Reference: ref,
	}
	rootIndex := index.NewSpecIndex(dummyNode)
	_ = schema.Build(context.Background(), dummyNode, dummyNode, rootIndex)

	assert.False(t, CheckSchemaProxyForCircularRefs(schema)) // no rolodex yet.

	rootIndex.SetRolodex(rolo)
	rolo.SetRootNode(dummyNode)
	rolo.SetRootIndex(rootIndex)
	rolo.SetSafeCircularReferences([]*index.CircularReferenceResult{
		{
			LoopPoint: &index.Reference{
				FullDefinition: "minty-fresh",
			},
		},
	})

	assert.True(t, CheckSchemaProxyForCircularRefs(schema)) // is circular

	ref = low.Reference{}
	ref.SetReference("tasty-burger", dummyNode)
	schema = &SchemaProxy{
		Reference: ref,
	}
	_ = schema.Build(context.Background(), dummyNode, dummyNode, rootIndex)

	assert.False(t, CheckSchemaProxyForCircularRefs(schema)) // not circular
}

func TestCheckSchemaProxyForCircularRefs_JourneyCheck(t *testing.T) {

	rolo := index.NewRolodex(&index.SpecIndexConfig{})
	dummyNode := &yaml.Node{Content: []*yaml.Node{{Content: []*yaml.Node{}}}}

	ref := low.Reference{}
	ref.SetReference("minty-fresh", dummyNode)

	schema := &SchemaProxy{
		Reference: ref,
	}
	rootIndex := index.NewSpecIndex(dummyNode)
	_ = schema.Build(context.Background(), dummyNode, dummyNode, rootIndex)

	rootIndex.SetRolodex(rolo)
	rolo.SetRootNode(dummyNode)
	rolo.SetRootIndex(rootIndex)
	rolo.SetSafeCircularReferences([]*index.CircularReferenceResult{
		{
			LoopPoint: &index.Reference{
				FullDefinition: "not-minty-fresh",
			},
			Journey: []*index.Reference{
				{
					FullDefinition: "minty-fresh",
				},
				{
					FullDefinition: "minty-fresh",
				},
			},
		},
	})

	assert.True(t, CheckSchemaProxyForCircularRefs(schema)) // no rolodex yet.

}

func TestCheckSchemaProxyForCircularRefs_JourneyCheck_LeadIn(t *testing.T) {

	rolo := index.NewRolodex(&index.SpecIndexConfig{})
	dummyNode := &yaml.Node{Content: []*yaml.Node{{Content: []*yaml.Node{}}}}

	ref := low.Reference{}
	ref.SetReference("minty-fresh", dummyNode)

	schema := &SchemaProxy{
		Reference: ref,
	}
	rootIndex := index.NewSpecIndex(dummyNode)
	_ = schema.Build(context.Background(), dummyNode, dummyNode, rootIndex)

	rootIndex.SetRolodex(rolo)
	rolo.SetRootNode(dummyNode)
	rolo.SetRootIndex(rootIndex)
	rolo.SetSafeCircularReferences([]*index.CircularReferenceResult{
		{
			LoopPoint: &index.Reference{
				FullDefinition: "tasty-burger",
			},
			Journey: []*index.Reference{
				{
					FullDefinition: "minty-fresh",
				},
				{
					FullDefinition: "tasty-burger",
				},
				{
					FullDefinition: "salty-fries",
				},
				{
					FullDefinition: "tasty-burger",
				},
			},
		},
	})

	assert.False(t, CheckSchemaProxyForCircularRefs(schema)) // only leads into the loop.
}

func TestCheckSchemaProxyForCircularRefs_JourneyCheck_DepthLimit(t *testing.T) {

	rolo := index.NewRolodex(&index.SpecIndexConfig{})
	dummyNode := &yaml.Node{Content: []*yaml.Node{{Content: []*yaml.Node{}}}}

	ref := low.Reference{}
	ref.SetReference("minty-fresh", dummyNode)

	schema := &SchemaProxy{
		Reference: ref,
	}
	rootIndex := index.NewSpecIndex(dummyNode)
	_ = schema.Build(context.Background(), dummyNode, dummyNode, rootIndex)

	rootIndex.SetRolodex(rolo)
	rolo.SetRootNode(dummyNode)
	rolo.SetRootIndex(rootIndex)
	rolo.SetSafeCircularReferences([]*index.CircularReferenceResult{
		{
			LoopPoint: &index.Reference{
				FullDefinition: "tasty-burger",
			},
			Journey: []*index.Reference{
				{
					FullDefinition: "minty-fresh",
				},
				{
					FullDefinition: "tasty-burger",
				},
			},
		},
	})

	assert.True(t, CheckSchemaProxyForCircularRefs(schema)) // the loop point only ends the journey, so all of it counts.
}

func TestCheckSchemaProxyForCircularRefs_ResolvesReference(t *testing.T) {
	yml := `openapi: 3.1.0
components:
  schemas:
    Pet:
      type: object
    A:
      type: object
      properties:
        b:
          $ref: '#/components/schemas/B'
    B:
      type: object
      properties:
        a:
          $ref: '#/components/schemas/A'`

	var rootNode yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yml), &rootNode))
	config := index.CreateOpenAPIIndexConfig()
	config.SpecAbsolutePath = "/specs/root.yaml" // journey entries hold absolute definitions, as in a real document.
	rolo := index.NewRolodex(config)
	rolo.SetRootNode(&rootNode)
	require.NoError(t, rolo.IndexTheRolodex(context.Background()))
	rolo.CheckForCircularReferences()
	idx := rolo.GetRootIndex()
	require.NotEmpty(t, idx.GetCircularReferences())

	// a proxy built from a reference node, as written in the document.
	proxy := func(ref string) *SchemaProxy {
		var node yaml.Node
		_ = yaml.Unmarshal([]byte(`$ref: '`+ref+`'`), &node)
		sp := new(SchemaProxy)
		_ = sp.Build(context.Background(), nil, node.Content[0], idx)
		return sp
	}

	// whichever schema is the loop point, the other is found by resolving the reference.
	assert.True(t, CheckSchemaProxyForCircularRefs(proxy("#/components/schemas/A")))
	assert.True(t, CheckSchemaProxyForCircularRefs(proxy("#/components/schemas/B")))
	assert.False(t, CheckSchemaProxyForCircularRefs(proxy("#/components/schemas/Pet")))
	assert.False(t, CheckSchemaProxyForCircularRefs(proxy("#/components/schemas/Missing")))

	// a child schema proxy holds the node its reference already resolved to.
	property := func(schema, name string) *SchemaProxy {
		node := idx.FindComponent(context.Background(), "#/components/schemas/"+schema).Node
		var sch Schema
		_ = low.BuildModel(node, &sch)
		_ = sch.Build(context.Background(), node, idx)
		return sch.FindProperty(name).Value
	}
	assert.True(t, CheckSchemaProxyForCircularRefs(property("A", "b")))
	assert.True(t, CheckSchemaProxyForCircularRefs(property("B", "a")))

	// an inline schema is not a reference.
	var inline yaml.Node
	_ = yaml.Unmarshal([]byte(`type: object`), &inline)
	sp := new(SchemaProxy)
	_ = sp.Build(context.Background(), nil, inline.Content[0], idx)
	assert.False(t, CheckSchemaProxyForCircularRefs(sp))
}
