// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"testing"

	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestParseSpecificationVersion_SupportedFamilies(t *testing.T) {
	tests := []struct {
		authored string
		family   FeatureFamily
		patch    int
	}{
		{authored: "1.0.99", family: FeatureFamily10, patch: 99},
		{authored: "1.1.0", family: FeatureFamily11, patch: 0},
		{authored: "1.1.2-rc.1", family: FeatureFamily11, patch: 2},
	}
	for _, test := range tests {
		version, err := ParseSpecificationVersion(test.authored)
		require.NoError(t, err)
		assert.Equal(t, test.authored, version.Authored)
		assert.Equal(t, 1, version.Major)
		assert.Equal(t, test.family, version.Family)
		assert.Equal(t, test.patch, version.Patch)
	}
}

func TestParseSpecificationVersion_Unsupported(t *testing.T) {
	for _, authored := range []string{
		"",
		"1.1",
		"1..0",
		"1.one.0",
		"1.-1.0",
		"1.1.0-",
		"2.0.0",
	} {
		version, err := ParseSpecificationVersion(authored)
		assert.Nil(t, version)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrUnsupportedVersion)
	}
}

func TestArazzo_GetSpecificationVersion(t *testing.T) {
	document := &Arazzo{Arazzo: "1.1.7"}
	version, err := document.GetSpecificationVersion()
	require.NoError(t, err)
	assert.Equal(t, FeatureFamily11, version.Family)
	assert.Equal(t, 7, version.Patch)

	var nilDocument *Arazzo
	version, err = nilDocument.GetSpecificationVersion()
	assert.Nil(t, version)
	assert.ErrorIs(t, err, ErrUnsupportedVersion)
}

func TestResolveDocumentOrigin_Precedence(t *testing.T) {
	absolute, err := ResolveDocumentOrigin(
		"https://identity.example/workflows/root.yaml",
		"file:///tmp/root.yaml",
		"https://application.example/default/",
	)
	require.NoError(t, err)
	assert.Equal(t, "https://identity.example/workflows/root.yaml", absolute.ResolvedIdentity)
	assert.Equal(t, absolute.ResolvedIdentity, absolute.EffectiveBaseURI)
	assert.Equal(t, "file:///tmp/root.yaml", absolute.RetrievalURI)

	relativeRetrieval, err := ResolveDocumentOrigin(
		"../portable/root.yaml",
		"https://retrieval.example/workflows/source.yaml",
		"https://application.example/default/",
	)
	require.NoError(t, err)
	assert.Equal(t, "https://retrieval.example/portable/root.yaml", relativeRetrieval.ResolvedIdentity)

	relativeApplication, err := ResolveDocumentOrigin(
		"portable/root.yaml",
		"",
		"https://application.example/default/",
	)
	require.NoError(t, err)
	assert.Equal(t, "https://application.example/default/portable/root.yaml", relativeApplication.ResolvedIdentity)

	unresolvedRelative, err := ResolveDocumentOrigin("portable/root.yaml", "", "")
	require.NoError(t, err)
	assert.Equal(t, "portable/root.yaml", unresolvedRelative.ResolvedIdentity)

	retrievalOnly, err := ResolveDocumentOrigin("", "file:///tmp/root.yaml", "https://ignored.example/")
	require.NoError(t, err)
	assert.Equal(t, "file:///tmp/root.yaml", retrievalOnly.ResolvedIdentity)

	applicationOnly, err := ResolveDocumentOrigin("", "", "https://application.example/root.yaml")
	require.NoError(t, err)
	assert.Equal(t, "https://application.example/root.yaml", applicationOnly.ResolvedIdentity)

	empty, err := ResolveDocumentOrigin("", "", "")
	require.NoError(t, err)
	assert.Empty(t, empty.ResolvedIdentity)
	assert.Empty(t, empty.EffectiveBaseURI)
}

func TestResolveDocumentOrigin_InvalidURIs(t *testing.T) {
	for _, test := range []struct {
		retrieval   string
		application string
	}{
		{retrieval: "https://example.com/%zz"},
		{application: "https://example.com/%zz"},
	} {
		origin, err := ResolveDocumentOrigin("", test.retrieval, test.application)
		assert.Nil(t, origin)
		require.Error(t, err)
	}
}

func TestResolveDocumentOrigin_MalformedAuthoredSelfIsPreserved(t *testing.T) {
	origin, err := ResolveDocumentOrigin(
		"https://identity.example/%zz",
		"https://retrieval.example/workflows/root.yaml",
		"https://application.example/default/",
	)
	require.NoError(t, err)
	require.NotNil(t, origin)
	assert.Equal(t, "https://identity.example/%zz", origin.AuthoredSelf)
	assert.Equal(t, "https://retrieval.example/workflows/root.yaml", origin.ResolvedIdentity)
	assert.Equal(t, "https://retrieval.example/workflows/root.yaml", origin.EffectiveBaseURI)

	origin, err = ResolveDocumentOrigin("https://identity.example/%zz", "", "")
	require.NoError(t, err)
	require.NotNil(t, origin)
	assert.Equal(t, "https://identity.example/%zz", origin.AuthoredSelf)
	assert.Empty(t, origin.ResolvedIdentity)
	assert.Empty(t, origin.EffectiveBaseURI)
}

func TestArazzo_DocumentOriginIsCopied(t *testing.T) {
	lowDocument := buildHighArazzo(t, arazzo11HighYAML).GoLow()
	origin := &DocumentOrigin{ResolvedIdentity: "https://example.com/original"}
	document := NewArazzoWithOrigin(lowDocument, origin)
	origin.ResolvedIdentity = "changed"

	first := document.GetDocumentOrigin()
	require.NotNil(t, first)
	assert.Equal(t, "https://example.com/original", first.ResolvedIdentity)
	first.ResolvedIdentity = "mutated-copy"
	assert.Equal(t, "https://example.com/original", document.GetDocumentOrigin().ResolvedIdentity)

	assert.Nil(t, NewArazzo(lowDocument).GetDocumentOrigin())
	var nilDocument *Arazzo
	assert.Nil(t, nilDocument.GetDocumentOrigin())
}
