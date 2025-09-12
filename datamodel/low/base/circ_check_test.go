// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package base

import (
	"context"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
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
