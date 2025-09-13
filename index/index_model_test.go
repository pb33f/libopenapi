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
