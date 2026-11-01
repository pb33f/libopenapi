// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package libopenapi

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/pb33f/libopenapi/datamodel"
)

func TestRootBaseURLDoesNotSendWorkingDirectoryToRemoteServer(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var requested []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requested = append(requested, r.URL.Path)
		mu.Unlock()
		if r.URL.Path == "/schemas/thing.yaml" {
			_, _ = w.Write([]byte("type: object\n"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	base, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	spec := []byte(`openapi: 3.1.0
info:
  title: root base
  version: 1.0.0
paths:
  /thing:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: ./schemas/thing.yaml
`)
	doc, err := NewDocumentWithConfiguration(spec, &datamodel.DocumentConfiguration{
		BaseURL:               base,
		AllowRemoteReferences: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	model, err := doc.BuildV3Model()
	if err != nil {
		t.Fatal(err)
	}

	operation := model.Model.Paths.PathItems.GetOrZero("/thing").Get
	mediaType := operation.Responses.Codes.GetOrZero("200").Content.GetOrZero("application/json")
	if schema := mediaType.Schema.Schema(); schema == nil {
		t.Fatalf("schema did not resolve: %v", mediaType.Schema.GetBuildError())
	}

	mu.Lock()
	defer mu.Unlock()
	for _, path := range requested {
		if strings.HasPrefix(path, cwd) {
			t.Fatalf("working directory was sent to the remote server: %s", path)
		}
	}
}
