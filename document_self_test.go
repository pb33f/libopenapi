// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package libopenapi

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestNewDocument_SelfRemoteReferences(t *testing.T) {
	const remoteSchema = `openapi: 3.2.0
info: {title: Schemas, version: '1'}
paths: {}
components:
  schemas:
    Value:
      type: string
`
	const entry = `{
 "openapi":"3.2.0",
 %s
 "info":{"title":"Cross-host reference","version":"1"},
 "paths":{"/items":{"get":{"responses":{"200":{"description":"OK","content":{"application/json":{"schema":{"$ref":%q}}}}}}}}
}`
	for _, tc := range []struct {
		name       string
		self       string
		base       string
		ref        string
		want       string
		nestedRef  string
		nestedWant string
	}{
		{
			name: "absolute without self",
			ref:  "https://schemas.example.net/shared.yaml",
			want: "https://schemas.example.net/shared.yaml",
		},
		{
			name: "absolute with self",
			self: "https://api.example.com/openapi.json",
			ref:  "https://schemas.example.net/shared.yaml",
			want: "https://schemas.example.net/shared.yaml",
		},
		{
			name: "absolute scheme port and query with self",
			self: "https://api.example.com/openapi.json",
			ref:  "http://schemas.example.net:8080/shared.yaml?version=1",
			want: "http://schemas.example.net:8080/shared.yaml?version=1",
		},
		{
			name: "relative with self",
			self: "https://api.example.com/schemas/",
			ref:  "shared.yaml",
			want: "https://api.example.com/schemas/shared.yaml",
		},
		{
			name:       "nested relative stays on external host",
			self:       "https://api.example.com/openapi.json",
			ref:        "https://schemas.example.net/models/shared.yaml",
			want:       "https://schemas.example.net/models/shared.yaml",
			nestedRef:  "value.yaml",
			nestedWant: "https://schemas.example.net/models/value.yaml",
		},
		{
			name:       "nested absolute keeps third host",
			self:       "https://api.example.com/openapi.json",
			ref:        "https://schemas.example.net/models/shared.yaml",
			want:       "https://schemas.example.net/models/shared.yaml",
			nestedRef:  "https://types.example.org/value.yaml",
			nestedWant: "https://types.example.org/value.yaml",
		},
		{
			name: "explicit base overrides absolute with self",
			self: "https://api.example.com/openapi.json",
			base: "https://override.example.org",
			ref:  "http://schemas.example.net/shared.yaml",
			want: "https://override.example.org/shared.yaml",
		},
		{
			name: "explicit base overrides absolute without self",
			base: "https://override.example.org",
			ref:  "http://schemas.example.net/shared.yaml",
			want: "https://override.example.org/shared.yaml",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			var requested []string
			config := datamodel.NewDocumentConfiguration()
			config.SpecFilePath = "https://retrieval.example/root.yaml?token=private"
			config.AllowRemoteReferences = true
			if tc.base != "" {
				var err error
				config.BaseURL, err = url.Parse(tc.base)
				require.NoError(t, err)
			}
			config.RemoteURLHandler = func(uri string) (*http.Response, error) {
				mu.Lock()
				requested = append(requested, uri)
				mu.Unlock()
				body := remoteSchema
				switch uri {
				case tc.want:
					if tc.nestedRef != "" {
						body = strings.Replace(remoteSchema, "type: string",
							fmt.Sprintf("$ref: %q", tc.nestedRef+"#/components/schemas/Value"), 1)
					}
				case tc.nestedWant:
				default:
					return nil, fmt.Errorf("unexpected retrieval URI: %s", uri)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(body)),
					Header:     http.Header{"Content-Type": []string{"application/yaml"}},
				}, nil
			}
			self := ""
			if tc.self != "" {
				self = fmt.Sprintf(`"$self":%q,`, tc.self)
			}
			doc, err := NewDocumentWithConfiguration([]byte(fmt.Sprintf(entry, self,
				tc.ref+"#/components/schemas/Value")), config)
			require.NoError(t, err)
			defer doc.Release()
			model, err := doc.BuildV3Model()
			wantRequests := []string{tc.want}
			if tc.nestedWant != "" {
				wantRequests = append(wantRequests, tc.nestedWant)
			}
			mu.Lock()
			assert.Equal(t, wantRequests, requested)
			mu.Unlock()
			require.NoError(t, err)
			require.NotNil(t, model)
			schema := model.Model.Paths.PathItems.GetOrZero("/items").Get.Responses.Codes.GetOrZero("200").
				Content.GetOrZero("application/json").Schema.Schema()
			require.NotNil(t, schema)
			assert.Equal(t, []string{"string"}, schema.Type)
		})
	}
}
