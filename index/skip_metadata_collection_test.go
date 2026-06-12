// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func buildSkipMetadataIndex(t *testing.T, skip bool) *SpecIndex {
	data, err := os.ReadFile("../test_specs/burgershop.openapi.yaml")
	require.NoError(t, err)
	var rootNode yaml.Node
	require.NoError(t, yaml.Unmarshal(data, &rootNode))

	cfg := CreateOpenAPIIndexConfig()
	cfg.AllowRemoteLookup = false
	cfg.AllowFileLookup = false
	cfg.SkipMetadataCollection = skip
	return NewSpecIndexWithConfig(&rootNode, cfg)
}

// TestSkipMetadataCollection_RefExtractionParity proves the flag only affects
// diagnostic metadata: reference extraction must be identical with it on or off.
func TestSkipMetadataCollection_RefExtractionParity(t *testing.T) {
	full := buildSkipMetadataIndex(t, false)
	skip := buildSkipMetadataIndex(t, true)

	fullRefs := full.GetRawReferencesSequenced()
	skipRefs := skip.GetRawReferencesSequenced()
	require.Equal(t, len(fullRefs), len(skipRefs))
	for i := range fullRefs {
		assert.Equal(t, fullRefs[i].Definition, skipRefs[i].Definition)
		assert.Equal(t, fullRefs[i].FullDefinition, skipRefs[i].FullDefinition)
	}

	// inline schema collections are still gathered (the resolver searches them by
	// FullDefinition), only their JSONPath Path values are skipped.
	fullInline := full.GetAllInlineSchemas()
	skipInline := skip.GetAllInlineSchemas()
	require.Equal(t, len(fullInline), len(skipInline))
	require.NotEmpty(t, fullInline)
	for i := range fullInline {
		assert.Equal(t, fullInline[i].Definition, skipInline[i].Definition)
		assert.Equal(t, fullInline[i].FullDefinition, skipInline[i].FullDefinition)
		assert.NotEmpty(t, fullInline[i].Path)
		assert.Empty(t, skipInline[i].Path)
	}

	fullRefSchemas := full.GetAllReferenceSchemas()
	skipRefSchemas := skip.GetAllReferenceSchemas()
	require.Equal(t, len(fullRefSchemas), len(skipRefSchemas))

	mapped := full.GetMappedReferences()
	mappedSkip := skip.GetMappedReferences()
	assert.Equal(t, len(mapped), len(mappedSkip))
}

// TestSkipMetadataCollection_MetadataEmpty proves all diagnostic collections are
// intentionally empty when the flag is enabled, and populated when it is not.
func TestSkipMetadataCollection_MetadataEmpty(t *testing.T) {
	full := buildSkipMetadataIndex(t, false)
	skip := buildSkipMetadataIndex(t, true)

	assert.NotEmpty(t, full.GetAllDescriptions())
	assert.NotEmpty(t, full.GetAllSummaries())
	assert.NotEmpty(t, full.GetAllEnums())
	assert.NotEmpty(t, full.GetAllObjectsWithProperties())
	assert.NotEmpty(t, full.GetSecurityRequirementReferences())
	assert.Positive(t, full.descriptionCount)

	assert.Empty(t, skip.GetAllDescriptions())
	assert.Empty(t, skip.GetAllSummaries())
	assert.Empty(t, skip.GetAllEnums())
	assert.Empty(t, skip.GetAllObjectsWithProperties())
	assert.Empty(t, skip.GetSecurityRequirementReferences())
	assert.Zero(t, skip.descriptionCount)
}
