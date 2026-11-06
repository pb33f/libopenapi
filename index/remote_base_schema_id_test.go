// Copyright 2026 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package index

import (
	"fmt"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
	"net/url"
	"testing"
)

func TestRemoteBasePreservesSchemaID(t *testing.T) {
	for _, id := range []string{"urn:example:thing", "thing.json"} {
		t.Run(id, func(t *testing.T) {
			var root yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(fmt.Sprintf(`openapi: 3.1.0
info:
  title: Absolute schema identifier
  version: 1.0.0
paths: {}
components:
  schemas:
    Thing:
      $id: '%s'
      type: integer
`, id)), &root))
			cfg := CreateClosedAPIIndexConfig()
			cfg.BaseURL, _ = url.Parse("https://api.example/")
			idx := NewSpecIndexWithConfig(&root, cfg)
			require.NotNil(t, idx.ResolveRefViaSchemaId(id))
			found, _ := idx.SearchIndexForReference(id)
			require.NotNil(t, found)
			require.Equal(t, "#/components/schemas/Thing", found.FullDefinition)

		})
	}
}
