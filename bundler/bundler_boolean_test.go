// Copyright 2026 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package bundler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/testify/require"
)

func TestBundlePreservesBooleanSchemas(t *testing.T) {
	const source = `openapi: 3.1.0
info:
  title: Rule action options
  version: 1.0.0
paths: {}
components:
  schemas:
    Allow: true
    Deny: false
    AllowRef:
      $ref: './options.yaml#/Allow'
    DenyRef:
      $ref: './options.yaml#/Deny'
    Action:
      type: object
      properties:
        allow: true
        deny: false
        allowRef:
          $ref: './options.yaml#/Allow'
        denyRef:
          $ref: './options.yaml#/Deny'
        allowItems:
          type: array
          items: true
        denyItems:
          type: array
          items: false
        allowItemsRef:
          type: array
          items:
            $ref: './options.yaml#/Allow'
        denyItemsRef:
          type: array
          items:
            $ref: './options.yaml#/Deny'
`
	directory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(directory, "options.yaml"), []byte("Allow: true\nDeny: false\n"), 0600))
	for _, mode := range []string{"inline", "compose"} {
		t.Run(mode, func(t *testing.T) {
			cfg := &datamodel.DocumentConfiguration{BasePath: directory, AllowFileReferences: true}
			var output []byte
			var err error
			if mode == "inline" {
				output, err = BundleBytes([]byte(source), cfg)
			} else {
				output, err = BundleBytesComposed([]byte(source), cfg, &BundleCompositionConfig{})
			}
			require.NoError(t, err)
			var doc map[string]any
			require.NoError(t, yaml.Unmarshal(output, &doc))
			schemas := doc["components"].(map[string]any)["schemas"].(map[string]any)
			resolve := func(value any) any {
				for depth := 0; depth <= len(schemas); depth++ {
					object, ok := value.(map[string]any)
					if !ok {
						return value
					}
					ref, ok := object["$ref"].(string)
					if !ok {
						return value
					}
					require.True(t, strings.HasPrefix(ref, "#/components/schemas/"), "unexpected unresolved reference %s", ref)
					var exists bool
					value, exists = schemas[strings.TrimPrefix(ref, "#/components/schemas/")]
					require.True(t, exists, "missing target %s", ref)
				}
				t.Fatal("boolean reference became circular")
				return nil
			}
			for _, name := range []string{"Allow", "AllowRef"} {
				require.Equal(t, true, resolve(schemas[name]), "%s", name)
			}
			for _, name := range []string{"Deny", "DenyRef"} {
				require.Equal(t, false, resolve(schemas[name]), "%s", name)
			}
			properties := schemas["Action"].(map[string]any)["properties"].(map[string]any)
			for _, name := range []string{"allow", "allowRef"} {
				require.Equal(t, true, resolve(properties[name]), "%s", name)
			}
			for _, name := range []string{"deny", "denyRef"} {
				require.Equal(t, false, resolve(properties[name]), "%s", name)
			}
			for _, name := range []string{"allowItems", "allowItemsRef", "denyItems", "denyItemsRef"} {
				require.Equal(t, strings.HasPrefix(name, "allow"), resolve(properties[name].(map[string]any)["items"]), "%s", name)
			}
		})
	}
}
