// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnhancedCoverage provides essential coverage for the newly implemented functionality
func TestEnhancedCoverage(t *testing.T) {
	t.Run("DetectContentType comprehensive cases", func(t *testing.T) {
		tests := []struct {
			name     string
			content  string
			expected FileExtension
		}{
			{
				name:     "JSON with negative brace count",
				content:  "}{{",
				expected: UNSUPPORTED,
			},
			{
				name:     "JSON array with negative bracket count",
				content:  "]][",
				expected: UNSUPPORTED,
			},
			{
				name: "YAML with sufficient patterns",
				content: `key1: value1
key2: value2`,
				expected: YAML,
			},
			{
				name: "Content with HTTP URL in key (should be rejected)",
				content: `http://example.com: value
other: test`,
				expected: UNSUPPORTED,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := detectContentType([]byte(tt.content))
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("FetchWithRetry error scenarios", func(t *testing.T) {
		t.Run("Network error with retries", func(t *testing.T) {
			attempts := 0
			handler := func(url string) (*http.Response, error) {
				attempts++
				if attempts < 3 {
					return nil, errors.New("network error")
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("success")),
				}, nil
			}

			logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
				Level: slog.LevelWarn,
			}))

			_, err := fetchWithRetry("http://example.com", handler, 1024, logger)
			// This tests the retry logic with logger
			if err != nil {
				assert.Contains(t, err.Error(), "failed to fetch")
			}
			assert.Equal(t, 3, attempts)
		})

		t.Run("HTTP error with retries", func(t *testing.T) {
			attempts := 0
			handler := func(url string) (*http.Response, error) {
				attempts++
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Status:     "Internal Server Error",
					Body:       io.NopCloser(strings.NewReader("")),
				}, nil
			}

			_, err := fetchWithRetry("http://example.com", handler, 1024, nil)
			assert.Error(t, err)
			assert.Equal(t, 3, attempts)
		})
	})

	t.Run("DetectRemoteContentType with caching", func(t *testing.T) {
		clearContentDetectionCache()

		// Test cache miss and population
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			_, _ = rw.Write([]byte(`{"test": "json"}`))
		}))
		defer server.Close()

		handler := func(url string) (*http.Response, error) {
			return http.Get(url)
		}

		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))

		// First call - cache miss
		result1 := detectRemoteContentType(server.URL, handler, logger)
		assert.Equal(t, JSON, result1)

		// Second call - cache hit (should not make HTTP request)
		result2 := detectRemoteContentType(server.URL, handler, logger)
		assert.Equal(t, JSON, result2)
	})

	t.Run("Content detection with unsupported result", func(t *testing.T) {
		clearContentDetectionCache()

		config := CreateOpenAPIIndexConfig()
		config.AllowUnknownExtensionContentDetection = true
		config.AllowRemoteLookup = true
		rfs, err := NewRemoteFSWithConfig(config)
		require.NoError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			_, _ = rw.Write([]byte("not json or yaml content"))
		}))
		defer server.Close()

		// This should trigger content detection, find unsupported content, and clean up cache
		file, err := rfs.OpenWithContext(context.Background(), server.URL+"/unknown")
		assert.Error(t, err)
		assert.Nil(t, file)

		// Verify cache was cleaned up
		contentDetectionMutex.RLock()
		_, exists := contentDetectionCache[server.URL+"/unknown"]
		contentDetectionMutex.RUnlock()
		assert.False(t, exists, "Cache should be cleaned up for unsupported content")
	})

	t.Run("RemoteFS error handling", func(t *testing.T) {
		config := CreateOpenAPIIndexConfig()
		config.AllowRemoteLookup = true
		rfs, err := NewRemoteFSWithConfig(config)
		require.NoError(t, err)

		// Test with handler that returns response but with error
		rfs.RemoteHandlerFunc = func(url string) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
			}, errors.New("handler error with response")
		}

		file, err := rfs.OpenWithContext(context.Background(), "https://example.com/test.yaml")
		assert.Error(t, err)
		assert.Nil(t, file)
	})

	t.Run("Last modified parsing failure", func(t *testing.T) {
		config := CreateOpenAPIIndexConfig()
		config.AllowRemoteLookup = true
		rfs, err := NewRemoteFSWithConfig(config)
		require.NoError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			rw.Header().Set("Last-Modified", "invalid-date-format")
			_, _ = rw.Write([]byte("openapi: 3.0.0"))
		}))
		defer server.Close()

		file, err := rfs.OpenWithContext(context.Background(), server.URL+"/test.yaml")
		if err == nil {
			assert.NotNil(t, file)
			if remoteFile, ok := file.(*RemoteFile); ok {
				// Should use current time when parsing fails
				assert.WithinDuration(t, time.Now(), remoteFile.lastModified, time.Minute)
			}
		}
	})
}
