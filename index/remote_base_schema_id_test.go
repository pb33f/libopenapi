// Copyright 2026 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package index

import (
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
	"net/url"
	"testing"
)

func TestRemoteBasePreservesAbsoluteSchemaID(t *testing.T) {
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(`openapi: 3.1.0
info:
  title: Absolute schema identifier
  version: 1.0.0
paths: {}
components:
  schemas:
    Thing:
      $id: 'urn:example:thing'
      type: integer
`), &root))
	cfg := CreateClosedAPIIndexConfig()
	cfg.BaseURL, _ = url.Parse("https://api.example/")
	idx := NewSpecIndexWithConfig(&root, cfg)
	require.NotNil(t, idx.ResolveRefViaSchemaId("urn:example:thing"))
	found, _ := idx.SearchIndexForReference("urn:example:thing")
	require.NotNil(t, found)
	require.Equal(t, "#/components/schemas/Thing", found.FullDefinition)
}
