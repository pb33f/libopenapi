// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// strictFS is a test file system that strictly enforces the fs.FS interface contract
// by rejecting absolute paths and paths with backslashes
type strictFS struct {
	fs.FS
}

func (s strictFS) Open(name string) (fs.File, error) {
	// Enforce fs.FS interface requirements
	if filepath.IsAbs(name) {
		return nil, fmt.Errorf("fs.FS violation: absolute path not allowed: %s", name)
	}
	if filepath.Separator == '\\' && filepath.ToSlash(name) != name {
		return nil, fmt.Errorf("fs.FS violation: backslash not allowed in path: %s", name)
	}
	return s.FS.Open(name)
}

func TestRolodex_FSCompatibility_RelativePath(t *testing.T) {
	t.Parallel()
	
	// Create a test filesystem that strictly enforces fs.FS interface
	testFS := strictFS{
		FS: fstest.MapFS{
			"spec.yaml":             {Data: []byte("test content"), ModTime: time.Now()},
			"refs/common.yaml":      {Data: []byte("common ref"), ModTime: time.Now()},
			"schemas/pet.yaml":      {Data: []byte("pet schema"), ModTime: time.Now()},
		},
	}

	baseDir := "/project/api"
	
	rolo := NewRolodex(CreateOpenAPIIndexConfig())
	rolo.AddLocalFS(baseDir, testFS)

	// Test 1: Open with relative path should work
	f, err := rolo.Open("spec.yaml")
	require.NoError(t, err, "Should successfully open file with relative path")
	assert.Equal(t, "test content", f.GetContent())

	// Test 2: Open with nested relative path should work
	f2, err := rolo.Open("refs/common.yaml")
	require.NoError(t, err, "Should successfully open nested file with relative path")
	assert.Equal(t, "common ref", f2.GetContent())

	// Test 3: Open with deeper nested path
	f3, err := rolo.Open("schemas/pet.yaml")
	require.NoError(t, err, "Should successfully open deeply nested file")
	assert.Equal(t, "pet schema", f3.GetContent())
}

func TestRolodex_FSCompatibility_AbsolutePath(t *testing.T) {
	t.Parallel()
	
	// Create a test filesystem that strictly enforces fs.FS interface
	testFS := strictFS{
		FS: fstest.MapFS{
			"api/spec.yaml":      {Data: []byte("api spec"), ModTime: time.Now()},
			"common/base.yaml":   {Data: []byte("base spec"), ModTime: time.Now()},
		},
	}

	baseDir, _ := filepath.Abs("/tmp/test")
	
	rolo := NewRolodex(CreateOpenAPIIndexConfig())
	rolo.AddLocalFS(baseDir, testFS)

	// Test with absolute path (which gets converted internally)
	// The rolodex should handle this by converting to relative path before calling Open
	f, err := rolo.Open(filepath.Join(baseDir, "api", "spec.yaml"))
	require.NoError(t, err, "Should handle absolute path by converting to relative")
	assert.Equal(t, "api spec", f.GetContent())
}

func TestRolodex_FSCompatibility_MultipleFS(t *testing.T) {
	t.Parallel()
	
	// For this test, we don't need strict enforcement since we're testing
	// the ability to find files across multiple file systems
	// The strict enforcement is tested in other test cases
	apiFS := fstest.MapFS{
		"openapi.yaml": {Data: []byte("api spec"), ModTime: time.Now()},
	}
	
	schemasFS := fstest.MapFS{
		"pet.json":    {Data: []byte("pet schema"), ModTime: time.Now()},
		"store.json":  {Data: []byte("store schema"), ModTime: time.Now()},
	}

	rolo := NewRolodex(CreateOpenAPIIndexConfig())
	rolo.AddLocalFS("/api", apiFS)
	rolo.AddLocalFS("/schemas", schemasFS)

	// Test opening from first FS - this should work as the file exists in first FS
	f1, err := rolo.Open("openapi.yaml")
	require.NoError(t, err, "Should open from first FS")
	assert.Equal(t, "api spec", f1.GetContent())

	// Test opening from second FS - this should work as the file exists in second FS
	f2, err := rolo.Open("pet.json")
	require.NoError(t, err, "Should open from second FS")
	assert.Equal(t, "pet schema", f2.GetContent())
}

func TestRolodex_FSCompatibility_StandardFS(t *testing.T) {
	t.Parallel()
	
	// Test with various standard fs.FS implementations
	testCases := []struct {
		name string
		fs   fs.FS
	}{
		{
			name: "fstest.MapFS",
			fs: fstest.MapFS{
				"test.yaml": {Data: []byte("test data"), ModTime: time.Now()},
			},
		},
		// Can add more fs.FS implementations here to test compatibility
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rolo := NewRolodex(CreateOpenAPIIndexConfig())
			rolo.AddLocalFS("/base", tc.fs)
			
			f, err := rolo.Open("test.yaml")
			require.NoError(t, err, "Should work with %s", tc.name)
			assert.Equal(t, "test data", f.GetContent())
		})
	}
}