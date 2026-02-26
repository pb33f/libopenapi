// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	high "github.com/pb33f/libopenapi/datamodel/high/arazzo"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveSources_PopulatesDocumentWithConfiguredFactories(t *testing.T) {
	doc := &high.Arazzo{
		SourceDescriptions: []*high.SourceDescription{
			{Name: "petstore", URL: "https://example.com/openapi.yaml", Type: "openapi"},
			{Name: "flows", URL: "https://example.com/flows.arazzo.yaml", Type: "arazzo"},
		},
	}

	payloads := map[string]string{
		"https://example.com/openapi.yaml":      "openapi: 3.1.0",
		"https://example.com/flows.arazzo.yaml": "arazzo: 1.0.1",
	}

	openAPIDoc := &v3high.Document{}
	arazzoDoc := &high.Arazzo{}

	config := &ResolveConfig{
		HTTPHandler: func(rawURL string) ([]byte, error) {
			body, ok := payloads[rawURL]
			if !ok {
				return nil, fmt.Errorf("unexpected url: %s", rawURL)
			}
			return []byte(body), nil
		},
		OpenAPIFactory: func(sourceURL string, data []byte) (*v3high.Document, error) {
			return openAPIDoc, nil
		},
		ArazzoFactory: func(sourceURL string, data []byte) (*high.Arazzo, error) {
			return arazzoDoc, nil
		},
	}

	resolved, err := ResolveSources(doc, config)
	require.NoError(t, err)
	require.Len(t, resolved, 2)

	assert.Equal(t, "openapi", resolved[0].Type)
	assert.Equal(t, "https://example.com/openapi.yaml", resolved[0].URL)
	assert.Same(t, openAPIDoc, resolved[0].OpenAPIDocument)
	assert.Nil(t, resolved[0].ArazzoDocument)

	assert.Equal(t, "arazzo", resolved[1].Type)
	assert.Equal(t, "https://example.com/flows.arazzo.yaml", resolved[1].URL)
	assert.Same(t, arazzoDoc, resolved[1].ArazzoDocument)
	assert.Nil(t, resolved[1].OpenAPIDocument)
}

func TestResolveSources_AutoAttachesOpenAPIDocs(t *testing.T) {
	doc := &high.Arazzo{
		SourceDescriptions: []*high.SourceDescription{
			{Name: "petstore", URL: "https://example.com/openapi.yaml", Type: "openapi"},
		},
	}

	openAPIDoc := &v3high.Document{}

	config := &ResolveConfig{
		HTTPHandler: func(_ string) ([]byte, error) {
			return []byte("openapi: 3.1.0"), nil
		},
		OpenAPIFactory: func(_ string, _ []byte) (*v3high.Document, error) {
			return openAPIDoc, nil
		},
	}

	_, err := ResolveSources(doc, config)
	require.NoError(t, err)

	attached := doc.GetOpenAPISourceDocuments()
	require.Len(t, attached, 1)
	assert.Same(t, openAPIDoc, attached[0])
}

func TestResolveSources_DefaultTypeUsesOpenAPIFactory(t *testing.T) {
	doc := &high.Arazzo{
		SourceDescriptions: []*high.SourceDescription{
			{Name: "defaultType", URL: "https://example.com/default.yaml"},
		},
	}

	var openAPIFactoryCalls int
	config := &ResolveConfig{
		HTTPHandler: func(rawURL string) ([]byte, error) {
			assert.Equal(t, "https://example.com/default.yaml", rawURL)
			return []byte("openapi: 3.1.0"), nil
		},
		OpenAPIFactory: func(sourceURL string, data []byte) (*v3high.Document, error) {
			openAPIFactoryCalls++
			return &v3high.Document{}, nil
		},
	}

	resolved, err := ResolveSources(doc, config)
	require.NoError(t, err)
	require.Len(t, resolved, 1)
	assert.Equal(t, "openapi", resolved[0].Type)
	assert.Equal(t, 1, openAPIFactoryCalls)
}

func TestResolveSources_MissingFactoryReturnsLoadError(t *testing.T) {
	doc := &high.Arazzo{
		SourceDescriptions: []*high.SourceDescription{
			{Name: "petstore", URL: "https://example.com/openapi.yaml", Type: "openapi"},
		},
	}

	config := &ResolveConfig{
		HTTPHandler: func(_ string) ([]byte, error) {
			return []byte("openapi: 3.1.0"), nil
		},
	}

	_, err := ResolveSources(doc, config)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrSourceDescLoadFailed))
	assert.Contains(t, err.Error(), "no OpenAPIFactory configured")
}

func TestResolveSources_FileSource_UsesFSRoots(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "source.yaml")
	require.NoError(t, os.WriteFile(filePath, []byte("openapi: 3.1.0"), 0o600))

	doc := &high.Arazzo{
		SourceDescriptions: []*high.SourceDescription{
			{Name: "local", URL: "source.yaml", Type: "openapi"},
		},
	}

	config := &ResolveConfig{
		FSRoots: []string{tmpDir},
		OpenAPIFactory: func(sourceURL string, data []byte) (*v3high.Document, error) {
			return &v3high.Document{}, nil
		},
	}

	resolved, err := ResolveSources(doc, config)
	require.NoError(t, err)
	require.Len(t, resolved, 1)
	require.NotNil(t, resolved[0].OpenAPIDocument)

	parsed, parseErr := url.Parse(resolved[0].URL)
	require.NoError(t, parseErr)
	assert.Equal(t, "file", parsed.Scheme)
	assert.Contains(t, parsed.Path, "/source.yaml")
}

func TestResolveFilePath_RejectsSymlinkOutsideRoot(t *testing.T) {
	rootDir := t.TempDir()
	outsideDir := t.TempDir()
	outsideFile := filepath.Join(outsideDir, "secret.yaml")
	require.NoError(t, os.WriteFile(outsideFile, []byte("openapi: 3.1.0"), 0o600))

	symlinkPath := filepath.Join(rootDir, "escaped.yaml")
	if err := os.Symlink(outsideFile, symlinkPath); err != nil {
		t.Skipf("symlinks not supported: %v", err)
	}

	_, err := resolveFilePath("escaped.yaml", []string{rootDir})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "outside configured roots")

	_, err = resolveFilePath(symlinkPath, []string{rootDir})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "outside configured roots")
}

func TestGetResolveHTTPClient_UsesConfigTimeout(t *testing.T) {
	c1 := getResolveHTTPClient(&ResolveConfig{Timeout: 5 * time.Second})
	require.Equal(t, 5*time.Second, c1.Timeout)

	c2 := getResolveHTTPClient(&ResolveConfig{Timeout: 6 * time.Second})
	require.Equal(t, 6*time.Second, c2.Timeout)

	// Each call creates a new client (no global cache).
	require.NotSame(t, c1, c2)

	// Custom client is returned as-is.
	custom := &http.Client{Timeout: 42 * time.Second}
	c3 := getResolveHTTPClient(&ResolveConfig{HTTPClient: custom, Timeout: 1 * time.Second})
	require.Same(t, custom, c3)
}
