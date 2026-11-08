// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package libopenapi

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

func TestDocumentRetrievalURLPreservesRemoteOriginsAndQueries(t *testing.T) {
	const root = "https://root.example/schemas/root.yaml?token=root"
	for _, tc := range []struct{ reference, item string }{
		{"./models/item.yaml", "https://root.example/schemas/models/item.yaml"},
		{"./models/item.yaml#/components/schemas/Item", "https://root.example/schemas/models/item.yaml"},
		{"./models/item.yaml?version=2", "https://root.example/schemas/models/item.yaml?version=2"},
		{"./models/item.yaml?version=2#/components/schemas/Item", "https://root.example/schemas/models/item.yaml?version=2"},
		{"/item.yaml", "https://root.example/item.yaml"},
		{"//other.example/item.yaml", "https://other.example/item.yaml"},
		{"./item%2Fleaf.yaml", "https://root.example/schemas/item%2Fleaf.yaml"},
		{"./item%2Fleaf.yaml#/components/schemas/Item", "https://root.example/schemas/item%2Fleaf.yaml"},
		{"./item.yaml?sig=a%26b%3Dc", "https://root.example/schemas/item.yaml?sig=a%26b%3Dc"},
		{"./item.yaml?sig=a%2Fb", "https://root.example/schemas/item.yaml?sig=a%2Fb"},
		{"./item%2Fleaf.yaml?sig=a%26b%3Dc#/components/schemas/Item%20Name", "https://root.example/schemas/item%2Fleaf.yaml?sig=a%26b%3Dc"},
	} {
		t.Run(tc.reference, func(t *testing.T) {
			reference, item := tc.reference, tc.item
			spec := []byte(fmt.Sprintf("openapi: 3.1.0\ninfo: {title: Retained origins, version: '1'}\npaths: {}\ncomponents:\n  schemas:\n    Item:\n      $ref: '%s'\n", reference))
			body := "type: object\nproperties:\n  id: {$ref: 'https://types.example/id.yaml?token=leaf'}\n"
			if strings.Contains(reference, "#") {
				body = "components:\n  schemas:\n    Item:\n      " + strings.ReplaceAll(strings.TrimSpace(body), "\n", "\n      ") + "\n"
			}
			if strings.Contains(reference, "Item%20Name") {
				body = strings.Replace(body, "    Item:", "    Item Name:", 1)
			}
			sources := map[string]string{
				item: body,
				"https://root.example/id.yaml?token=leaf":  "type: integer\n",
				"https://types.example/id.yaml?token=leaf": "$ref: './identifier.yaml'\n",
				"https://types.example/identifier.yaml":    "type: string\nformat: uuid\n",
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
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(content))}, nil
			}
			document, err := NewDocumentWithConfiguration(spec, config)
			require.NoError(t, err)
			defer document.Release()
			built, err := document.BuildV3Model()
			require.NoError(t, err)
			require.Equal(t, root, built.Index.GetSpecAbsolutePath())
			schema := built.Model.Components.Schemas.GetOrZero("Item").Schema()
			require.NotNil(t, schema)
			id := schema.Properties.GetOrZero("id").Schema()
			require.NotNil(t, id)
			require.Equal(t, []string{"string"}, id.Type)
			require.Equal(t, "uuid", id.Format)
			mu.Lock()
			defer mu.Unlock()
			require.Equal(t, map[string]int{item: 1, "https://types.example/id.yaml?token=leaf": 1, "https://types.example/identifier.yaml": 1}, calls)
		})
	}
}
