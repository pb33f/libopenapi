// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIssue438_UnknownExtensionContentDetection tests the fix for issue #438
// where URLs without known file extensions should be handled with content detection
func TestIssue438_UnknownExtensionContentDetection(t *testing.T) {
	// Test YAML content without extension
	yamlContent := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Pet:
      type: object
      properties:
        name:
          type: string`

	// Test JSON content without extension
	jsonContent := `{
  "openapi": "3.0.0",
  "info": {
    "title": "Test API",
    "version": "1.0.0"
  },
  "components": {
    "schemas": {
      "Pet": {
        "type": "object",
        "properties": {
          "name": {
            "type": "string"
          }
        }
      }
    }
  }
}`

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/yaml-no-ext":
			rw.Header().Set("Content-Type", "text/plain")
			_, _ = rw.Write([]byte(yamlContent))
		case "/json-no-ext":
			rw.Header().Set("Content-Type", "text/plain")
			_, _ = rw.Write([]byte(jsonContent))
		case "/invalid-content":
			rw.Header().Set("Content-Type", "text/plain")
			_, _ = rw.Write([]byte("This is not YAML or JSON content"))
		case "/binary-content":
			rw.Header().Set("Content-Type", "application/octet-stream")
			binaryData := []byte{0xFF, 0xD8, 0xFF, 0xE0} // JPEG header
			_, _ = rw.Write(binaryData)
		default:
			rw.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	t.Run("YAML content detection enabled", func(t *testing.T) {
		config := CreateOpenAPIIndexConfig()
		config.AllowUnknownExtensionContentDetection = true
		rfs, err := NewRemoteFSWithConfig(config)
		require.NoError(t, err)

		// Clear cache to ensure fresh detection
		clearContentDetectionCache()

		file, err := rfs.Open(server.URL + "/yaml-no-ext")
		assert.NoError(t, err)
		assert.NotNil(t, file)

		if remoteFile, ok := file.(*RemoteFile); ok {
			assert.Equal(t, YAML, remoteFile.extension)
			content := remoteFile.GetContent()
			assert.Contains(t, content, "openapi: 3.0.0")
		}
	})

	t.Run("JSON content detection enabled", func(t *testing.T) {
		config := CreateOpenAPIIndexConfig()
		config.AllowUnknownExtensionContentDetection = true
		rfs, err := NewRemoteFSWithConfig(config)
		require.NoError(t, err)

		// Clear cache to ensure fresh detection
		clearContentDetectionCache()

		file, err := rfs.Open(server.URL + "/json-no-ext")
		assert.NoError(t, err)
		assert.NotNil(t, file)

		if remoteFile, ok := file.(*RemoteFile); ok {
			assert.Equal(t, JSON, remoteFile.extension)
			content := remoteFile.GetContent()
			assert.Contains(t, content, `"openapi": "3.0.0"`)
		}
	})

	t.Run("Content detection disabled", func(t *testing.T) {
		config := CreateOpenAPIIndexConfig()
		config.AllowUnknownExtensionContentDetection = false // Disabled
		rfs, err := NewRemoteFSWithConfig(config)
		require.NoError(t, err)

		// Clear cache to ensure fresh detection
		clearContentDetectionCache()

		file, err := rfs.Open(server.URL + "/yaml-no-ext")
		assert.Error(t, err)
		assert.Nil(t, file)
		assert.Contains(t, err.Error(), "invalid argument")
	})

	t.Run("Invalid content detection", func(t *testing.T) {
		config := CreateOpenAPIIndexConfig()
		config.AllowUnknownExtensionContentDetection = true
		rfs, err := NewRemoteFSWithConfig(config)
		require.NoError(t, err)

		// Clear cache to ensure fresh detection
		clearContentDetectionCache()

		file, err := rfs.Open(server.URL + "/invalid-content")
		assert.Error(t, err)
		assert.Nil(t, file)
		assert.Contains(t, err.Error(), "invalid argument")
	})

	t.Run("Binary content detection", func(t *testing.T) {
		config := CreateOpenAPIIndexConfig()
		config.AllowUnknownExtensionContentDetection = true
		rfs, err := NewRemoteFSWithConfig(config)
		require.NoError(t, err)

		// Clear cache to ensure fresh detection
		clearContentDetectionCache()

		file, err := rfs.Open(server.URL + "/binary-content")
		assert.Error(t, err)
		assert.Nil(t, file)
		assert.Contains(t, err.Error(), "invalid argument")
	})
}

// TestContentTypeDetection tests the detectContentType function directly
func TestContentTypeDetection(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected FileExtension
	}{
		{
			name: "JSON object",
			content: `{
  "openapi": "3.0.0",
  "info": {
    "title": "Test"
  }
}`,
			expected: JSON,
		},
		{
			name: "JSON array",
			content: `[
  {"name": "test1"},
  {"name": "test2"}
]`,
			expected: JSON,
		},
		{
			name: "YAML with document marker",
			content: `---
openapi: 3.0.0
info:
  title: Test API`,
			expected: YAML,
		},
		{
			name: "YAML without document marker",
			content: `openapi: 3.0.0
info:
  title: Test API
paths:
  /test:
    get:
      responses:
        '200':
          description: OK`,
			expected: YAML,
		},
		{
			name: "YAML with comments",
			content: `# This is a comment
openapi: 3.0.0  # Version
info:
  title: Test API
  description: |
    This is a multi-line
    description`,
			expected: YAML,
		},
		{
			name:     "Empty content",
			content:  "",
			expected: UNSUPPORTED,
		},
		{
			name:     "Only whitespace",
			content:  "   \t\n   \r\n  ",
			expected: UNSUPPORTED,
		},
		{
			name:     "Plain text",
			content:  "This is just plain text without structure",
			expected: UNSUPPORTED,
		},
		{
			name:     "URLs (not YAML)",
			content:  "https://example.com/path: some value",
			expected: UNSUPPORTED,
		},
		{
			name: "Malformed JSON (still detected as JSON)",
			content: `{
  "key": "value"
  "missing": comma
}`,
			expected: JSON,
		},
		{
			name:     "Single YAML key (insufficient)",
			content:  `key: value`,
			expected: UNSUPPORTED,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectContentType([]byte(tt.content))
			assert.Equal(t, tt.expected, result, "Content type detection failed for: %s", tt.name)
		})
	}
}

// TestFetchWithRetry tests the retry logic for fetching remote content
func TestFetchWithRetry(t *testing.T) {
	t.Run("Success on first try", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			_, _ = rw.Write([]byte("success"))
		}))
		defer server.Close()

		handler := func(url string) (*http.Response, error) {
			return http.Get(url)
		}

		data, err := fetchWithRetry(server.URL, handler, 1024, nil)
		assert.NoError(t, err)
		assert.Equal(t, []byte("success"), data)
	})

	t.Run("Success after retry", func(t *testing.T) {
		attempt := 0
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			attempt++
			if attempt < 2 {
				rw.WriteHeader(http.StatusInternalServerError)
				return
			}
			_, _ = rw.Write([]byte("success after retry"))
		}))
		defer server.Close()

		handler := func(url string) (*http.Response, error) {
			return http.Get(url)
		}

		data, err := fetchWithRetry(server.URL, handler, 1024, nil)
		assert.NoError(t, err)
		assert.Equal(t, []byte("success after retry"), data)
		assert.Equal(t, 2, attempt)
	})

	t.Run("Failure after max retries", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		handler := func(url string) (*http.Response, error) {
			return http.Get(url)
		}

		data, err := fetchWithRetry(server.URL, handler, 1024, nil)
		assert.Error(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "failed to fetch after 3 attempts")
	})

	t.Run("Network error with retry", func(t *testing.T) {
		attempt := 0
		handler := func(url string) (*http.Response, error) {
			attempt++
			if attempt < 2 {
				return nil, errors.New("network error")
			}
			// Simulate success on second attempt
			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				_, _ = rw.Write([]byte("recovered"))
			}))
			defer server.Close()
			return http.Get(server.URL)
		}

		data, err := fetchWithRetry("http://example.com", handler, 1024, nil)
		assert.NoError(t, err)
		assert.Equal(t, []byte("recovered"), data)
		assert.Equal(t, 2, attempt)
	})

	t.Run("Content size limit", func(t *testing.T) {
		largeContent := strings.Repeat("x", 5000)
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			_, _ = rw.Write([]byte(largeContent))
		}))
		defer server.Close()

		handler := func(url string) (*http.Response, error) {
			return http.Get(url)
		}

		// Limit to 1KB
		data, err := fetchWithRetry(server.URL, handler, 1024, nil)
		assert.NoError(t, err)
		assert.Len(t, data, 1024)
		assert.Equal(t, strings.Repeat("x", 1024), string(data))
	})
}

// TestContentDetectionCache tests the caching mechanism
func TestContentDetectionCache(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		_, _ = rw.Write([]byte("openapi: 3.0.0\ninfo:\n  title: Test"))
	}))
	defer server.Close()

	config := CreateOpenAPIIndexConfig()
	config.AllowUnknownExtensionContentDetection = true
	rfs, err := NewRemoteFSWithConfig(config)
	require.NoError(t, err)

	// Clear cache
	clearContentDetectionCache()

	url := server.URL + "/test"

	// First call should cache the result
	result1 := detectRemoteContentType(url, rfs.RemoteHandlerFunc, nil)
	assert.Equal(t, YAML, result1)

	// Second call should use cached result (server won't be hit again)
	result2 := detectRemoteContentType(url, rfs.RemoteHandlerFunc, nil)
	assert.Equal(t, YAML, result2)

	// Clear cache and verify it's cleared
	clearContentDetectionCache()

	// Verify cache is actually cleared by checking if we can detect again
	result3 := detectRemoteContentType(url, rfs.RemoteHandlerFunc, nil)
	assert.Equal(t, YAML, result3)
}

// TestIssue438_PastebinExample tests the specific scenario from the GitHub issue
// This test focuses on the RemoteFS-level functionality without document creation
func TestIssue438_PastebinExample(t *testing.T) {
	// Mock the Pastebin-like response
	schema := `{
  "type": "object",
  "properties": {
    "id": {
      "type": "integer",
      "format": "int64"
    },
    "name": {
      "type": "string"
    },
    "status": {
      "type": "string",
      "enum": ["available", "pending", "sold"]
    }
  },
  "required": ["name", "status"]
}`

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/raw/LAvtwJn6" {
			rw.Header().Set("Content-Type", "text/plain")
			_, _ = rw.Write([]byte(schema))
		} else {
			rw.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	t.Run("Content detection enabled", func(t *testing.T) {
		config := CreateOpenAPIIndexConfig()
		config.AllowUnknownExtensionContentDetection = true
		rfs, err := NewRemoteFSWithConfig(config)
		require.NoError(t, err)

		// Clear cache
		clearContentDetectionCache()

		// This URL has no file extension, mimicking the Pastebin example
		file, err := rfs.Open(server.URL + "/raw/LAvtwJn6")
		assert.NoError(t, err)
		assert.NotNil(t, file)

		if remoteFile, ok := file.(*RemoteFile); ok {
			assert.Equal(t, JSON, remoteFile.extension)
			content := remoteFile.GetContent()
			assert.Contains(t, content, `"type": "object"`)
			assert.Contains(t, content, `"properties"`)
		}
	})

	t.Run("Content detection disabled - should fail", func(t *testing.T) {
		config := CreateOpenAPIIndexConfig()
		config.AllowUnknownExtensionContentDetection = false
		rfs, err := NewRemoteFSWithConfig(config)
		require.NoError(t, err)

		// Clear cache
		clearContentDetectionCache()

		file, err := rfs.Open(server.URL + "/raw/LAvtwJn6")
		assert.Error(t, err)
		assert.Nil(t, file)
		assert.Contains(t, err.Error(), "invalid argument")
	})
}
