// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package low_test

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestGenerateHashString_SchemaProxyAndSchemaTypeCheck(t *testing.T) {
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("type: string"), &node))

	var proxy base.SchemaProxy
	require.NoError(t, proxy.Build(context.Background(), nil, node.Content[0], nil))

	low.ClearHashCache()
	proxyHash := low.GenerateHashString(&proxy)
	assert.NotEmpty(t, proxyHash)

	schema := proxy.Schema()
	require.NotNil(t, schema)
	schemaHash := low.GenerateHashString(schema)
	assert.NotEmpty(t, schemaHash)
}
