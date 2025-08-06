// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIssue361_FSInterfaceCompliance tests the fix for issue #361
// where Rolodex was calling fs.FS.Open() with absolute paths,
// violating the fs.FS interface specification.
func TestIssue361_FSInterfaceCompliance(t *testing.T) {
	// Create a standard fs.FS implementation (fstest.MapFS)
	// This would fail with the old implementation when given absolute paths
	testFS := fstest.MapFS{
		"openapi.yaml": {
			Data: []byte(`openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        '200':
          description: OK`),
			ModTime: time.Now(),
		},
		"schemas/pet.yaml": {
			Data: []byte(`type: object
properties:
  name:
    type: string
  age:
    type: integer`),
			ModTime: time.Now(),
		},
	}

	// Create a Rolodex and add the standard fs.FS
	config := CreateOpenAPIIndexConfig()
	rolo := NewRolodex(config)
	
	// Add the fs.FS with a base directory
	// The fix ensures that when opening files, relative paths are used
	// with the fs.FS interface, not absolute paths
	// Use a temporary directory to ensure cross-platform compatibility
	tempDir, err := os.MkdirTemp("", "rolodex-test")
	require.NoError(t, err, "Should be able to create temp dir")
	defer os.RemoveAll(tempDir)
	
	baseDir := filepath.Join(tempDir, "api", "v1")
	rolo.AddLocalFS(baseDir, testFS)
	
	// Test 1: Open a file at the root of the FS
	f1, err := rolo.Open("openapi.yaml")
	require.NoError(t, err, "Should open file using relative path with fs.FS")
	assert.Contains(t, f1.GetContent(), "Test API")
	
	// Test 2: Open a nested file
	f2, err := rolo.Open("schemas/pet.yaml")
	require.NoError(t, err, "Should open nested file using relative path with fs.FS")
	assert.Contains(t, f2.GetContent(), "type: object")
	
	// Test 3: Verify absolute paths are converted correctly
	// Even if we pass an absolute path matching the base + relative path,
	// it should work by converting to relative
	absolutePath := filepath.Join(baseDir, "openapi.yaml")
	f3, err := rolo.Open(absolutePath)
	require.NoError(t, err, "Should handle absolute paths by converting to relative")
	assert.Contains(t, f3.GetContent(), "Test API")
}

// TestIssue361_MultipleFileSystems tests that the fix works correctly
// when multiple file systems are registered and files need to be found
// across them.
func TestIssue361_MultipleFileSystems(t *testing.T) {
	// Create multiple standard fs.FS implementations
	apiFS := fstest.MapFS{
		"api.yaml": {Data: []byte("api content"), ModTime: time.Now()},
	}
	
	schemaFS := fstest.MapFS{
		"schema.json": {Data: []byte("schema content"), ModTime: time.Now()},
	}
	
	// Create Rolodex with multiple file systems
	config := CreateOpenAPIIndexConfig()
	rolo := NewRolodex(config)
	
	// Use temporary directories for cross-platform compatibility
	tempDir, err := os.MkdirTemp("", "rolodex-multi-test")
	require.NoError(t, err, "Should be able to create temp dir")
	defer os.RemoveAll(tempDir)
	
	rolo.AddLocalFS(filepath.Join(tempDir, "apis"), apiFS)
	rolo.AddLocalFS(filepath.Join(tempDir, "schemas"), schemaFS)
	
	// Files should be found in their respective file systems
	f1, err := rolo.Open("api.yaml")
	require.NoError(t, err, "Should find api.yaml in first FS")
	assert.Equal(t, "api content", f1.GetContent())
	
	f2, err := rolo.Open("schema.json")
	require.NoError(t, err, "Should find schema.json in second FS")
	assert.Equal(t, "schema content", f2.GetContent())
	
	// Non-existent file should return error
	_, err = rolo.Open("nonexistent.yaml")
	assert.Error(t, err, "Should return error for non-existent file")
}