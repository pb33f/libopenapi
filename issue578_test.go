// Copyright 2026 Princess Beef Heavy Industries / Dave Shanley
// https://pb33f.io
// SPDX-License-Identifier: MIT
package libopenapi

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"testing"

	"github.com/pb33f/libopenapi/datamodel"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/testify/require"
)

const issue578Spec = `openapi: 3.1.0
info:
  title: t
  version: 1.0.0
paths:
  /x:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "%s"
`

func issue578Config(sequential bool, requested *[]string, mu *sync.Mutex) *datamodel.DocumentConfiguration {
	cfg := datamodel.NewDocumentConfiguration()
	cfg.AllowRemoteReferences = true
	cfg.ExtractRefsSequentially = sequential
	cfg.RemoteURLHandler = func(u string) (*http.Response, error) {
		mu.Lock()
		*requested = append(*requested, u)
		mu.Unlock()
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString("type: object\n")),
		}, nil
	}
	return cfg
}

// A BaseURL without a scheme must not strip the scheme from absolute remote refs. The rolodex
// used to receive no file and no error, then panic reading it; when refs were extracted
// concurrently the panic was re-raised on a fresh goroutine and killed the process.
func TestIssue578SchemelessBaseURLDoesNotPanic(t *testing.T) {
	tests := []struct {
		name string
		base string
	}{
		{name: "no scheme or host", base: "example.com/specs/"},
		{name: "host without scheme", base: "//example.com/specs/"},
	}
	for _, tt := range tests {
		for _, sequential := range []bool{true, false} {
			name := tt.name + " concurrent"
			if sequential {
				name = tt.name + " sequential"
			}
			t.Run(name, func(t *testing.T) {
				var mu sync.Mutex
				var requested []string
				cfg := issue578Config(sequential, &requested, &mu)
				baseURL, err := url.Parse(tt.base)
				require.NoError(t, err)
				cfg.BaseURL = baseURL

				spec := []byte(fmt.Sprintf(issue578Spec, "https://example.com/schemas/pet.yaml"))
				doc, err := NewDocumentWithConfiguration(spec, cfg)
				require.NoError(t, err)

				var model *DocumentModel[v3high.Document]
				require.NotPanics(t, func() {
					model, err = doc.BuildV3Model()
				})
				require.NoError(t, err)
				require.NotNil(t, model)

				mu.Lock()
				require.Equal(t, []string{"https://example.com/schemas/pet.yaml"}, requested)
				mu.Unlock()

				pathItem, ok := model.Model.Paths.PathItems.Get("/x")
				require.True(t, ok)
				mediaType, ok := pathItem.Get.Responses.Codes.GetOrZero("200").Content.Get("application/json")
				require.True(t, ok)
				schema := mediaType.Schema.Schema()
				require.NotNil(t, schema)
				require.Equal(t, []string{"object"}, schema.Type)
			})
		}
	}
}

// A local ref that merely starts with 'http' is routed to the remote file system, where it has no
// scheme. It must fail to resolve with an error rather than panic.
func TestIssue578HttpPrefixedLocalRefDoesNotPanic(t *testing.T) {
	for _, sequential := range []bool{true, false} {
		name := "concurrent"
		if sequential {
			name = "sequential"
		}
		t.Run(name, func(t *testing.T) {
			var mu sync.Mutex
			var requested []string
			var logs bytes.Buffer
			cfg := issue578Config(sequential, &requested, &mu)
			cfg.Logger = slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelError}))

			doc, err := NewDocumentWithConfiguration([]byte(fmt.Sprintf(issue578Spec, "httpdocs/pet.yaml")), cfg)
			require.NoError(t, err)

			require.NotPanics(t, func() {
				_, err = doc.BuildV3Model()
			})
			require.EqualError(t, err, "component `httpdocs/pet.yaml` does not exist in the specification")
			require.Contains(t, logs.String(), "remote URL 'httpdocs/pet.yaml' has no scheme, unable to fetch it")

			mu.Lock()
			require.Empty(t, requested)
			mu.Unlock()
		})
	}
}
