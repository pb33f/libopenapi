// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// failingReader is a reader that fails after reading a certain number of bytes
type failingReader struct {
	failAfter int
	read      int
}

func (f *failingReader) Read(p []byte) (n int, err error) {
	if f.read >= f.failAfter {
		return 0, fmt.Errorf("simulated read failure")
	}
	f.read++
	return 0, fmt.Errorf("simulated read failure")
}

// TestDetectContentTypeComprehensive tests all edge cases for content type detection
func TestDetectContentTypeComprehensive(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected FileExtension
	}{
		{
			name:     "JSON object with negative brace count",
			data:     []byte(`{"key": "value"} extra closing }`),
			expected: UNSUPPORTED, // negative brace count should return UNSUPPORTED
		},
		{
			name:     "JSON array with negative bracket count",
			data:     []byte(`["item1", "item2"] extra closing ]`),
			expected: UNSUPPORTED, // negative bracket count should return UNSUPPORTED
		},
		{
			name:     "Content with key containing slash (rejected by YAML logic)",
			data:     []byte("key/with/slash: value\nsecond: value"),
			expected: UNSUPPORTED, // keys with slashes are rejected
		},
		{
			name:     "Content with key containing space (rejected by YAML logic)",
			data:     []byte("key with space: value\nsecond: value"),
			expected: UNSUPPORTED, // keys with spaces are rejected
		},
		{
			name:     "YAML content with exactly one pattern (insufficient)",
			data:     []byte("single_key: value"),
			expected: UNSUPPORTED, // single pattern is insufficient (needs >= 2)
		},
		{
			name: "YAML content checking line limit (>10 lines)",
			data: []byte(`line1: value1
line2: value2
line3: value3
line4: value4
line5: value5
line6: value6
line7: value7
line8: value8
line9: value9
line10: value10
line11: value11  # This line should not be checked due to i > 10 limit
line12: value12  # This line should not be checked due to i > 10 limit`),
			expected: YAML, // Should detect YAML from first 10 lines with multiple patterns
		},
		{
			name:     "Content with colons in URLs (should be ignored by YAML detection)",
			data:     []byte("http://example.com:8080/path\nhttps://another.com:443/path"),
			expected: UNSUPPORTED, // HTTP URLs should be ignored, no valid YAML patterns
		},
		{
			name:     "YAML with exactly 2 patterns (threshold case)",
			data:     []byte("key1: value1\nkey2: value2"),
			expected: YAML, // exactly 2 patterns should be detected as YAML
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectContentType(tt.data)
			assert.Equal(t, tt.expected, result, "detectContentType failed for case: %s", tt.name)
		})
	}
}

// TestFetchWithRetryComprehensive tests all retry scenarios including errors
func TestFetchWithRetryComprehensive(t *testing.T) {
	t.Run("HTTP error on all attempts leading to final failure", func(t *testing.T) {
		attempt := 0
		handler := func(url string) (*http.Response, error) {
			attempt++
			return &http.Response{
				StatusCode: 500,
				Status:     "Internal Server Error",
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		}

		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

		_, err := fetchWithRetry("http://test.com", handler, 1024, logger)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch after 3 attempts")
		assert.Equal(t, 3, attempt, "Should retry exactly 3 times")
	})

	t.Run("ReadAll error during content reading", func(t *testing.T) {
		attempt := 0
		handler := func(url string) (*http.Response, error) {
			attempt++
			// Create a reader that will fail
			failingReader := &failingReader{failAfter: 0}
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(failingReader),
			}, nil
		}

		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

		_, err := fetchWithRetry("http://test.com", handler, 1024, logger)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch after 3 attempts")
		assert.Equal(t, 3, attempt, "Should retry exactly 3 times on ReadAll error")
	})

	t.Run("Logger nil case", func(t *testing.T) {
		handler := func(url string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("success")),
			}, nil
		}

		data, err := fetchWithRetry("http://test.com", handler, 1024, nil)

		assert.NoError(t, err)
		assert.Equal(t, []byte("success"), data)
	})
}

// TestDetectRemoteContentTypeComprehensive tests caching and error scenarios
func TestDetectRemoteContentTypeComprehensive(t *testing.T) {
	// Clear cache before test
	clearContentDetectionCache()

	t.Run("Network error that gets cached", func(t *testing.T) {
		handler := func(url string) (*http.Response, error) {
			return nil, fmt.Errorf("network error")
		}

		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

		// First call - should fail and cache the UNSUPPORTED result
		result1 := detectRemoteContentType("http://failing.com", handler, logger)
		assert.Equal(t, UNSUPPORTED, result1)

		// Second call - should return cached UNSUPPORTED result without calling handler
		calledAgain := false
		handler2 := func(url string) (*http.Response, error) {
			calledAgain = true
			return nil, fmt.Errorf("should not be called")
		}

		result2 := detectRemoteContentType("http://failing.com", handler2, logger)
		assert.Equal(t, UNSUPPORTED, result2)
		assert.False(t, calledAgain, "Handler should not be called for cached result")
	})

	t.Run("Logger nil case", func(t *testing.T) {
		handler := func(url string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"json": "content"}`)),
			}, nil
		}

		result := detectRemoteContentType("http://test-nil-logger.com", handler, nil)
		assert.Equal(t, JSON, result)
	})

	// Clear cache after test
	clearContentDetectionCache()
}

// TestRemoteFSOpenWithContextComprehensive tests all edge cases for the main OpenWithContext method
func TestRemoteFSOpenWithContextComprehensive(t *testing.T) {
	t.Run("Content detection cleanup when unsupported", func(t *testing.T) {
		// Create config with content detection enabled
		config := CreateOpenAPIIndexConfig()
		config.AllowUnknownExtensionContentDetection = true

		rfs, err := NewRemoteFSWithConfig(config)
		assert.NoError(t, err)

		// Mock handler that returns unsupported content
		rfs.RemoteHandlerFunc = func(url string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("binary content that's not JSON or YAML")),
			}, nil
		}

		// Try to open a file with unknown extension
		_, err = rfs.OpenWithContext(context.Background(), "http://test.com/unknown.bin")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid argument")

		// Verify that cache entry was cleaned up (defer function executed)
		contentDetectionMutex.RLock()
		_, exists := contentDetectionCache["http://test.com/unknown.bin"]
		contentDetectionMutex.RUnlock()
		assert.False(t, exists, "Cache entry should be cleaned up after unsupported detection")
	})

	t.Run("Non-HTTP URL handling", func(t *testing.T) {
		config := CreateOpenAPIIndexConfig()
		rfs, err := NewRemoteFSWithConfig(config)
		assert.NoError(t, err)

		// Test file:// URL (not HTTP)
		_, err = rfs.OpenWithContext(context.Background(), "file:///local/file.yaml")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not a remote file")
	})

	t.Run("Already processing file - waiter scenario", func(t *testing.T) {
		config := CreateOpenAPIIndexConfig()
		rfs, err := NewRemoteFSWithConfig(config)
		assert.NoError(t, err)

		// Create a mock waiter in processing
		waiter := &waiterRemote{
			f:    "/test.yaml",
			done: true,
			file: &RemoteFile{filename: "test.yaml", data: []byte("test: data")},
			mu:   sync.Mutex{},
		}

		parsedURL, _ := url.Parse("http://test.com/test.yaml")
		rfs.ProcessingFiles.Store(parsedURL.Path, waiter)

		// Should return the file from waiter
		file, err := rfs.OpenWithContext(context.Background(), "http://test.com/test.yaml")

		assert.NoError(t, err)
		assert.NotNil(t, file)
		assert.Equal(t, "test.yaml", file.(*RemoteFile).GetFileName())
	})

	t.Run("Processing file with error", func(t *testing.T) {
		config := CreateOpenAPIIndexConfig()
		rfs, err := NewRemoteFSWithConfig(config)
		assert.NoError(t, err)

		// Create a mock waiter with error
		waitError := fmt.Errorf("processing error")
		waiter := &waiterRemote{
			f:     "/error-test.yaml",
			done:  true,
			file:  nil,
			error: waitError,
			mu:    sync.Mutex{},
		}

		parsedURL, _ := url.Parse("http://test.com/error-test.yaml")
		rfs.ProcessingFiles.Store(parsedURL.Path, waiter)

		// Should return the error from waiter
		file, err := rfs.OpenWithContext(context.Background(), "http://test.com/error-test.yaml")

		assert.Nil(t, file)
		assert.Error(t, err)
		assert.Equal(t, waitError, err)
	})

	t.Run("Rolodex integration - AddExternalIndex path", func(t *testing.T) {
		config := CreateOpenAPIIndexConfig()
		rfs, err := NewRemoteFSWithConfig(config)
		assert.NoError(t, err)

		// Create a mock rolodex
		rolodex := NewRolodex(config)
		rfs.rolodex = rolodex

		// Mock handler that returns valid YAML
		rfs.RemoteHandlerFunc = func(url string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("openapi: 3.0.0\ninfo:\n  title: Test\n  version: 1.0.0\npaths: {}")),
				Header:     http.Header{"Last-Modified": []string{"Wed, 21 Oct 2015 07:28:00 GMT"}},
			}, nil
		}

		// Open the file - this should trigger the rolodex.AddExternalIndex path
		file, err := rfs.OpenWithContext(context.Background(), "http://test.com/spec.yaml")

		assert.NoError(t, err)
		assert.NotNil(t, file)

		// Verify rolodex has the external index
		assert.True(t, len(rolodex.GetCaughtErrors()) == 0, "Should have no errors in rolodex")
	})

	t.Run("Index error handling - errors joined", func(t *testing.T) {
		config := CreateOpenAPIIndexConfig()
		rfs, err := NewRemoteFSWithConfig(config)
		assert.NoError(t, err)

		// Mock handler that returns content that will fail indexing
		rfs.RemoteHandlerFunc = func(url string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("")), // Empty content causes index error
				Header:     http.Header{"Last-Modified": []string{"Wed, 21 Oct 2015 07:28:00 GMT"}},
			}, nil
		}

		// Add some existing errors to test error joining
		rfs.remoteErrors = []error{fmt.Errorf("existing error 1"), fmt.Errorf("existing error 2")}

		// Open the file - this should cause an index error and join it with existing errors
		_, err = rfs.OpenWithContext(context.Background(), "http://test.com/empty.yaml")

		assert.Error(t, err)
		// The error should contain all errors joined together
		errString := err.Error()
		assert.Contains(t, errString, "existing error 1")
		assert.Contains(t, errString, "existing error 2")
		assert.Contains(t, errString, "nothing was extracted")
	})

	t.Run("Content detection success with different types", func(t *testing.T) {
		tests := []struct {
			name        string
			content     string
			expectedLog string
		}{
			{
				name:        "JSON detection",
				content:     `{"openapi": "3.0.0"}`,
				expectedLog: "JSON",
			},
			{
				name:        "YAML detection",
				content:     `openapi: 3.0.0\ninfo:\n  title: test`,
				expectedLog: "YAML",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				config := CreateOpenAPIIndexConfig()
				config.AllowUnknownExtensionContentDetection = true

				rfs, err := NewRemoteFSWithConfig(config)
				assert.NoError(t, err)

				rfs.RemoteHandlerFunc = func(url string) (*http.Response, error) {
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader(tt.content)),
						Header:     http.Header{"Last-Modified": []string{"Wed, 21 Oct 2015 07:28:00 GMT"}},
					}, nil
				}

				// This should trigger successful content detection and the logging path
				file, err := rfs.OpenWithContext(context.Background(), "http://test.com/unknown.bin")

				assert.NoError(t, err)
				assert.NotNil(t, file)
			})
		}
	})

	t.Run("Already cached file lookup", func(t *testing.T) {
		config := CreateOpenAPIIndexConfig()
		rfs, err := NewRemoteFSWithConfig(config)
		assert.NoError(t, err)

		// Create a cached file
		testFile := &RemoteFile{
			filename: "cached.yaml",
			name:     "/cached.yaml",
			data:     []byte("cached: content"),
		}

		// Add file to cache using the path as key
		rfs.Files.Store("/cached.yaml", testFile)

		// Request should return cached file without going to handler
		file, err := rfs.OpenWithContext(context.Background(), "http://test.com/cached.yaml")

		assert.NoError(t, err)
		assert.NotNil(t, file)
		assert.Equal(t, "cached.yaml", file.(*RemoteFile).GetFileName())
	})

	t.Run("Content detection disabled for unknown extension", func(t *testing.T) {
		// Create config with content detection disabled (default)
		config := CreateOpenAPIIndexConfig()
		config.AllowUnknownExtensionContentDetection = false // explicitly disable

		rfs, err := NewRemoteFSWithConfig(config)
		assert.NoError(t, err)

		// Should fail for unknown extension without content detection
		_, err = rfs.OpenWithContext(context.Background(), "http://test.com/unknown.bin")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid argument")

		// Verify error was added to remoteErrors
		assert.Greater(t, len(rfs.GetErrors()), 0)
	})

	t.Run("File extension UNSUPPORTED", func(t *testing.T) {
		config := CreateOpenAPIIndexConfig()
		rfs, err := NewRemoteFSWithConfig(config)
		assert.NoError(t, err)

		// Use a real unsupported extension
		_, err = rfs.OpenWithContext(context.Background(), "http://test.com/binary.exe")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid argument")
	})

	t.Run("Client error with nil response", func(t *testing.T) {
		config := CreateOpenAPIIndexConfig()
		rfs, err := NewRemoteFSWithConfig(config)
		assert.NoError(t, err)

		rfs.RemoteHandlerFunc = func(url string) (*http.Response, error) {
			return nil, fmt.Errorf("client error")
		}

		_, err = rfs.OpenWithContext(context.Background(), "http://test.com/test.yaml")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "client error")
	})

	t.Run("Successful indexing without rolodex", func(t *testing.T) {
		config := CreateOpenAPIIndexConfig()
		rfs, err := NewRemoteFSWithConfig(config)
		assert.NoError(t, err)

		// Ensure no rolodex is set (it's nil by default)
		rfs.rolodex = nil

		rfs.RemoteHandlerFunc = func(url string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("openapi: 3.0.0\ninfo:\n  title: Test\n  version: 1.0.0\npaths: {}")),
				Header:     http.Header{"Last-Modified": []string{"Wed, 21 Oct 2015 07:28:00 GMT"}},
			}, nil
		}

		file, err := rfs.OpenWithContext(context.Background(), "http://test.com/spec.yaml")

		assert.NoError(t, err)
		assert.NotNil(t, file)
		// Since rolodex is nil, the rolodex.AddExternalIndex branch should not be taken
	})

	t.Run("Waiter with listeners count", func(t *testing.T) {
		config := CreateOpenAPIIndexConfig()
		rfs, err := NewRemoteFSWithConfig(config)
		assert.NoError(t, err)

		// Create a waiter with specific listeners count
		waiter := &waiterRemote{
			f:         "/test-listeners.yaml",
			done:      true,
			file:      &RemoteFile{filename: "test.yaml", data: []byte("test: data")},
			listeners: 5, // This field is logged
			mu:        sync.Mutex{},
		}

		parsedURL, _ := url.Parse("http://test.com/test-listeners.yaml")
		rfs.ProcessingFiles.Store(parsedURL.Path, waiter)

		file, err := rfs.OpenWithContext(context.Background(), "http://test.com/test-listeners.yaml")

		assert.NoError(t, err)
		assert.NotNil(t, file)
	})

	t.Run("BaseURL scheme override", func(t *testing.T) {
		// Create config with base URL that has different scheme/host
		config := CreateOpenAPIIndexConfig()
		baseURL, _ := url.Parse("https://override.com/base")
		config.BaseURL = baseURL

		rfs, err := NewRemoteFSWithConfig(config)
		assert.NoError(t, err)

		rfs.RemoteHandlerFunc = func(url string) (*http.Response, error) {
			// Verify that the URL was rewritten to use override.com
			assert.Contains(t, url, "override.com")
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("test: data")),
				Header:     http.Header{"Last-Modified": []string{"Wed, 21 Oct 2015 07:28:00 GMT"}},
			}, nil
		}

		file, err := rfs.OpenWithContext(context.Background(), "http://original.com/test.yaml")

		assert.NoError(t, err)
		assert.NotNil(t, file)
	})

	t.Run("Empty response body scenario", func(t *testing.T) {
		config := CreateOpenAPIIndexConfig()
		rfs, err := NewRemoteFSWithConfig(config)
		assert.NoError(t, err)

		rfs.RemoteHandlerFunc = func(url string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("")), // Empty body
				Header:     http.Header{},
			}, nil
		}

		// This should trigger the "nothing was extracted" error path in indexing
		file, err := rfs.OpenWithContext(context.Background(), "http://test.com/empty.yaml")

		// The file should be created but indexing should fail
		assert.NotNil(t, file)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nothing was extracted")
	})

	t.Run("Successful parse but no last modified header", func(t *testing.T) {
		config := CreateOpenAPIIndexConfig()
		rfs, err := NewRemoteFSWithConfig(config)
		assert.NoError(t, err)

		rfs.RemoteHandlerFunc = func(url string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("openapi: 3.0.0\ninfo:\n  title: Test\n  version: 1.0.0\npaths: {}")),
				Header:     http.Header{}, // No Last-Modified header
			}, nil
		}

		file, err := rfs.OpenWithContext(context.Background(), "http://test.com/no-lastmod.yaml")

		assert.NoError(t, err)
		assert.NotNil(t, file)

		// Should use current time since Last-Modified parsing fails
		assert.True(t, file.(*RemoteFile).GetLastModified().Unix() > 0)
	})

	t.Run("Malformed Last-Modified header", func(t *testing.T) {
		config := CreateOpenAPIIndexConfig()
		rfs, err := NewRemoteFSWithConfig(config)
		assert.NoError(t, err)

		rfs.RemoteHandlerFunc = func(url string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("openapi: 3.0.0\ninfo:\n  title: Test\n  version: 1.0.0\npaths: {}")),
				Header:     http.Header{"Last-Modified": []string{"not a valid date"}},
			}, nil
		}

		file, err := rfs.OpenWithContext(context.Background(), "http://test.com/bad-lastmod.yaml")

		assert.NoError(t, err)
		assert.NotNil(t, file)

		// Should use current time since Last-Modified parsing fails
		assert.True(t, file.(*RemoteFile).GetLastModified().Unix() > 0)
	})

	t.Run("Content detection success with nil logger from config", func(t *testing.T) {
		// The real scenario is when logger is nil from NewRemoteFSWithConfig
		config := CreateOpenAPIIndexConfig()
		config.Logger = nil // Logger is nil in config
		config.AllowUnknownExtensionContentDetection = true

		rfs, err := NewRemoteFSWithConfig(config)
		assert.NoError(t, err)

		rfs.RemoteHandlerFunc = func(url string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"openapi": "3.0.0", "info": {"title": "test", "version": "1.0.0"}}`)),
				Header:     http.Header{"Last-Modified": []string{"Wed, 21 Oct 2015 07:28:00 GMT"}},
			}, nil
		}

		// This should work with the default logger created when config.Logger is nil
		file, err := rfs.OpenWithContext(context.Background(), "http://test.com/file.unknown")
		assert.NoError(t, err)
		assert.NotNil(t, file)
	})
}
