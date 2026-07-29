// Copyright 2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package bundler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/testify/require"
)

// A $ref that resolves to a sequence node with an odd number of elements used
// to panic the composed bundler with "index out of range" because the ref
// walker stepped through the node's content two at a time assuming key/value
// pairs. Both odd- and even-length arrays must bundle without panicking.
func TestBundleBytesComposed_RefToOddLengthArray(t *testing.T) {
	for _, count := range []int{63, 64, 65} {
		t.Run(fmt.Sprintf("%d-tags", count), func(t *testing.T) {
			tmpDir := t.TempDir()

			var tags strings.Builder
			tags.WriteString("tags:\n")
			for i := 0; i < count; i++ {
				tags.WriteString(fmt.Sprintf("- name: Tag%d\n  description: d%d\n", i, i))
			}
			require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "tags.yaml"), []byte(tags.String()), 0o644))

			root := `openapi: 3.0.2
info:
  title: t
  version: 1.0.0
tags:
  $ref: "tags.yaml#/tags"
paths:
  /x:
    get:
      responses:
        "200":
          description: ok
`
			config := datamodel.NewDocumentConfiguration()
			config.BasePath = tmpDir

			bundled, err := BundleBytesComposed([]byte(root), config, nil)
			require.NoError(t, err)
			require.NotEmpty(t, bundled)
		})
	}
}
