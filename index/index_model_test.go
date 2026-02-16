// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"encoding/json"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestSpecIndex_GetConfig(t *testing.T) {
	idx1 := NewTestSpecIndex().Load().(*SpecIndex)
	c := SpecIndexConfig{}
	id := c.GetId()
	assert.NotNil(t, id)
	idx1.config = &c
	assert.Equal(t, &c, idx1.GetConfig())
}

func TestSpecIndex_Rolodex(t *testing.T) {
	idx1 := NewTestSpecIndex().Load().(*SpecIndex)
	assert.Nil(t, idx1.GetResolver())
	idx1.SetResolver(&Resolver{})
	assert.NotNil(t, idx1.GetResolver())
	assert.NotNil(t, idx1.GetConfig().GetId())
}

func Test_MarshalJSON(t *testing.T) {
	rm := &ReferenceMapped{
		OriginalReference: &Reference{
			FullDefinition: "full definition",
			Path:           "path",
			Node: &yaml.Node{
				Line:   1,
				Column: 1,
				Content: []*yaml.Node{
					{
						Line:   9,
						Column: 10,
					},
					{
						Value: "lemon cake",
					},
				},
			},
		},
		Reference: &Reference{
			FullDefinition: "full definition",
			Path:           "path",
			Node: &yaml.Node{
				Line:   2,
				Column: 2,
			},
			KeyNode: &yaml.Node{
				Line:   3,
				Column: 3,
			},
		},
		Definition:     "definition",
		FullDefinition: "full definition",
		IsPolymorphic:  true,
	}

	bytes, _ := json.Marshal(rm)
	assert.Len(t, bytes, 173)
}

func TestSpecIndexConfig_ToDocumentConfiguration_Nil(t *testing.T) {
	var config *SpecIndexConfig = nil
	result := config.ToDocumentConfiguration()
	assert.Nil(t, result)
}

func TestSpecIndexConfig_ToDocumentConfiguration_AllFields(t *testing.T) {
	baseURL, _ := url.Parse("https://example.com")
	config := &SpecIndexConfig{
		BaseURL:                               baseURL,
		BasePath:                              "/api",
		SpecFilePath:                          "/path/to/spec.yaml",
		AllowFileLookup:                       true,
		AllowRemoteLookup:                     true,
		SkipDocumentCheck:                     true,
		IgnorePolymorphicCircularReferences:   true,
		IgnoreArrayCircularReferences:         true,
		UseSchemaQuickHash:                    true,
		AllowUnknownExtensionContentDetection: true,
		TransformSiblingRefs:                  true,
	}

	result := config.ToDocumentConfiguration()

	assert.NotNil(t, result)
	assert.Equal(t, baseURL, result.BaseURL)
	assert.Equal(t, "/api", result.BasePath)
	assert.Equal(t, "/path/to/spec.yaml", result.SpecFilePath)
	assert.True(t, result.AllowFileReferences)
	assert.True(t, result.AllowRemoteReferences)
	assert.True(t, result.BypassDocumentCheck)
	assert.True(t, result.IgnorePolymorphicCircularReferences)
	assert.True(t, result.IgnoreArrayCircularReferences)
	assert.True(t, result.UseSchemaQuickHash)
	assert.True(t, result.AllowUnknownExtensionContentDetection)
	assert.True(t, result.TransformSiblingRefs)
	assert.False(t, result.MergeReferencedProperties) // default disabled for index configs
}

func TestSpecIndexConfig_ToDocumentConfiguration_SkipExternalRefResolution(t *testing.T) {
	config := &SpecIndexConfig{
		SkipExternalRefResolution: true,
	}
	result := config.ToDocumentConfiguration()
	assert.NotNil(t, result)
	assert.True(t, result.SkipExternalRefResolution)
}

func TestSpecIndexConfig_ToDocumentConfiguration_SkipExternalRefResolution_False(t *testing.T) {
	config := &SpecIndexConfig{}
	result := config.ToDocumentConfiguration()
	assert.NotNil(t, result)
	assert.False(t, result.SkipExternalRefResolution)
}

func TestSpecIndex_Release(t *testing.T) {
	cfg := CreateOpenAPIIndexConfig()
	rootNode := &yaml.Node{Value: "root"}
	resolver := &Resolver{resolvedRoot: &yaml.Node{Value: "resolved"}}

	rolodex := NewRolodex(cfg)
	rolodex.rootNode = &yaml.Node{Value: "rolodex-root"}

	idx := &SpecIndex{
		config:                   cfg,
		root:                     rootNode,
		pathsNode:                &yaml.Node{},
		tagsNode:                 &yaml.Node{},
		schemasNode:              &yaml.Node{},
		allRefs:                  map[string]*Reference{"ref": {}},
		rawSequencedRefs:         []*Reference{{}},
		allMappedRefs:            map[string]*Reference{"mapped": {}},
		allMappedRefsSequenced:   []*ReferenceMapped{{}},
		nodeMap:                  map[int]map[int]*yaml.Node{1: {1: &yaml.Node{}}},
		allDescriptions:          []*DescriptionReference{{}},
		allEnums:                 []*EnumReference{{}},
		circularReferences:       []*CircularReferenceResult{{}},
		refErrors:                []error{nil},
		resolver:                 resolver,
		rolodex:                  rolodex,
		allComponentSchemas:      map[string]*Reference{"schema": {}},
		allExternalDocuments:     map[string]*Reference{"ext": {}},
		externalSpecIndex:        map[string]*SpecIndex{"ext": {}},
		schemaIdRegistry:         map[string]*SchemaIdEntry{"id": {}},
		uri:                      []string{"test"},
	}

	idx.Release()

	// yaml.Node fields
	assert.Nil(t, idx.root)
	assert.Nil(t, idx.pathsNode)
	assert.Nil(t, idx.tagsNode)
	assert.Nil(t, idx.schemasNode)

	// reference maps
	assert.Nil(t, idx.allRefs)
	assert.Nil(t, idx.rawSequencedRefs)
	assert.Nil(t, idx.allMappedRefs)
	assert.Nil(t, idx.allMappedRefsSequenced)

	// node map
	assert.Nil(t, idx.nodeMap)

	// descriptions, enums
	assert.Nil(t, idx.allDescriptions)
	assert.Nil(t, idx.allEnums)

	// circular refs, errors
	assert.Nil(t, idx.circularReferences)
	assert.Nil(t, idx.refErrors)

	// component schemas, external docs
	assert.Nil(t, idx.allComponentSchemas)
	assert.Nil(t, idx.allExternalDocuments)
	assert.Nil(t, idx.externalSpecIndex)

	// schema ID registry, uri, logger
	assert.Nil(t, idx.schemaIdRegistry)
	assert.Nil(t, idx.uri)
	assert.Nil(t, idx.logger)

	// resolver released and niled
	assert.Nil(t, idx.resolver)
	assert.Nil(t, resolver.specIndex)
	assert.Nil(t, resolver.resolvedRoot)

	// rolodex released and niled
	assert.Nil(t, idx.rolodex)
	assert.Nil(t, rolodex.rootNode)
	assert.Nil(t, rolodex.indexes)

	// config niled
	assert.Nil(t, idx.config)
}

func TestSpecIndex_Release_Nil(t *testing.T) {
	var idx *SpecIndex
	idx.Release() // must not panic
}

func TestSpecIndex_Release_Idempotent(t *testing.T) {
	idx := &SpecIndex{
		root:     &yaml.Node{},
		config:   CreateOpenAPIIndexConfig(),
		resolver: &Resolver{},
		rolodex:  NewRolodex(CreateOpenAPIIndexConfig()),
	}
	idx.Release()
	idx.Release() // second call must not panic
	assert.Nil(t, idx.root)
	assert.Nil(t, idx.config)
	assert.Nil(t, idx.resolver)
	assert.Nil(t, idx.rolodex)
}

func TestSpecIndex_Release_NilConfig(t *testing.T) {
	idx := &SpecIndex{root: &yaml.Node{}}
	idx.Release() // config is nil, must not panic
	assert.Nil(t, idx.root)
}

func TestSpecIndex_Release_ConfigWithNilSpecInfo(t *testing.T) {
	idx := &SpecIndex{
		config: &SpecIndexConfig{}, // SpecInfo is nil
	}
	idx.Release() // SpecInfo.Release() called on nil, must not panic
	assert.Nil(t, idx.config)
}

func TestSpecIndex_Release_NilResolverAndRolodex(t *testing.T) {
	idx := &SpecIndex{root: &yaml.Node{}}
	// resolver and rolodex are nil
	idx.Release() // must not panic
	assert.Nil(t, idx.root)
}
