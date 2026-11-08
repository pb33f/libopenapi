// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"net/url"
	"path/filepath"
	"testing"

	"github.com/pb33f/testify/require"
)

func TestResolveReferenceTarget_PreservesAbsoluteLocalFilePointer(t *testing.T) {
	directory := t.TempDir()
	file := filepath.Join(directory, "types", "item.yaml")
	for _, root := range []string{"", filepath.Join(directory, "schemas", "root.yaml")} {
		t.Run(root, func(t *testing.T) {
			config := CreateOpenAPIIndexConfig()
			config.BaseURL, _ = url.Parse("https://remote.example/schemas/")
			idx := &SpecIndex{config: config, specAbsolutePath: root}
			for _, pointer := range []string{"#/components/schemas/Item", "#/components/schemas/"} {
				// a remote lookup base must not reinterpret an explicit local filename.
				reference := file + pointer
				resolved, component := idx.resolveReferenceTarget(reference)
				require.Equal(t, reference, resolved)
				require.Equal(t, pointer, component)
			}
		})
	}
}
