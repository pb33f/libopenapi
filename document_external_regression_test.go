// Copyright 2026 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package libopenapi

import (
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func externalRegressionConfig(t *testing.T, files map[string]string) *datamodel.DocumentConfiguration {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	}
	return &datamodel.DocumentConfiguration{BasePath: dir, AllowFileReferences: true}
}
