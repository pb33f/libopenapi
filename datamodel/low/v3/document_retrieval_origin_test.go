// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/testify/require"
)

func TestCreateDocument_RetrievalOriginKeepsAcceptedRemoteIdentities(t *testing.T) {
	const root = "https://root.example/schemas/root.yaml?token=root"
	info, err := datamodel.ExtractSpecInfo([]byte(`openapi: 3.1.0
info: {title: Retrieval origin, version: '1'}
paths: {}
components:
  schemas:
    Item:
      $ref: './item.yaml?version=2'
`))
	require.NoError(t, err)
	sources := map[string]string{
		"https://root.example/schemas/item.yaml?version=2": "type: object\nproperties:\n  id: {$ref: 'https://types.example/id.yaml?token=leaf'}\n",
		"https://types.example/id.yaml?token=leaf":         "type: string\nformat: uuid\n",
		"https://root.example/id.yaml?token=leaf":          "type: integer\n",
	}
	var mu sync.Mutex
	calls := make(map[string]int)
	config := datamodel.NewDocumentConfiguration()
	config.SpecFilePath = root
	config.AllowRemoteReferences = true
	config.RemoteURLHandler = func(location string) (*http.Response, error) {
		mu.Lock()
		defer mu.Unlock()
		calls[location]++
		content, ok := sources[location]
		if !ok {
			return nil, fmt.Errorf("unexpected source: %s", location)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(content))}, nil
	}
	document, err := CreateDocumentFromConfig(info, config)
	require.NoError(t, err)
	defer document.Rolodex.Release()
	require.Equal(t, root, document.Index.GetSpecAbsolutePath())
	item := document.Components.Value.FindSchema("Item")
	require.NotNil(t, item)
	schema := item.Value.Schema()
	require.NotNil(t, schema)
	id := schema.FindProperty("id")
	require.NotNil(t, id)
	identifier := id.Value.Schema()
	require.NotNil(t, identifier)
	require.Equal(t, "string", identifier.Type.Value.A)
	require.Equal(t, "uuid", identifier.Format.Value)
	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, map[string]int{
		"https://root.example/schemas/item.yaml?version=2": 1,
		"https://types.example/id.yaml?token=leaf":         1,
	}, calls)
}
