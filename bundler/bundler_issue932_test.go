// Copyright 2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package bundler

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestBundleBytesComposed_ExternalSecurityScheme(t *testing.T) {
	tests := []struct {
		name              string
		ref               string
		expectedComponent string
		extraFiles        map[string]string
	}{
		{
			name:              "bare file",
			ref:               "./bearer-auth.yaml",
			expectedComponent: "bearer-auth",
			extraFiles: map[string]string{
				"bearer-auth.yaml": `type: http
scheme: bearer
bearerFormat: JWT
description: JWT bearer token.
`,
			},
		},
		{
			name:              "component fragment",
			ref:               "./shared.yaml#/components/securitySchemes/bearerAuth",
			expectedComponent: "bearerAuth__shared",
			extraFiles: map[string]string{
				"shared.yaml": `components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
      description: JWT bearer token.
`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			for name, contents := range tt.extraFiles {
				require.NoError(t, os.WriteFile(filepath.Join(tmpDir, name), []byte(contents), 0o644))
			}

			root := `openapi: 3.2.0
info:
  title: Repro
  version: 1.0.0
paths: {}
components:
  securitySchemes:
    bearerAuth:
      $ref: "` + tt.ref + `"
`
			config := datamodel.NewDocumentConfiguration()
			config.BasePath = tmpDir

			bundled, err := BundleBytesComposed([]byte(root), config, nil)
			require.NoError(t, err)

			output := string(bundled)
			assert.Contains(t, output, `$ref: "#/components/securitySchemes/`+tt.expectedComponent+`"`)
			assert.Contains(t, output, tt.expectedComponent+":\n      type: http")
			assert.Contains(t, output, "scheme: bearer")
			assert.Contains(t, output, "bearerFormat: JWT")
			assert.NotContains(t, output, "#/components/schemas/")
			assert.NotContains(t, output, tt.ref)
		})
	}
}
