// Copyright 2026 Dave Shanley / Quobix
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
	"github.com/pb33f/testify/require"
)

func TestRemoteNestedReferencesPreserveOrigin(t *testing.T) {
	spec := []byte(`openapi: 3.1.0
info:
  title: Remote origin regression
  version: 1.0.0
paths: {}
components:
  schemas:
    Item:
      $ref: './schemas/item.yaml#/components/schemas/Item'
`)
	sources := map[string]string{
		"https://api.example/schemas/item.yaml": "components:\n  schemas:\n    Item:\n      type: object\n      properties:\n        id:\n          $ref: 'http://types.example/id.yaml#/components/schemas/Id'\n",
		"http://types.example/id.yaml":          "components:\n  schemas:\n    Id:\n      type: integer\n",
	}
	var mu sync.Mutex
	calls := map[string]int{}
	base, _ := url.Parse("https://api.example/")
	doc, err := NewDocumentWithConfiguration(spec, &datamodel.DocumentConfiguration{BaseURL: base, ResolveNestedRefsWithDocumentContext: true, AllowRemoteReferences: true, RemoteURLHandler: func(location string) (*http.Response, error) {
		mu.Lock()
		defer mu.Unlock()
		calls[location]++
		body, ok := sources[location]
		if !ok {
			return nil, fmt.Errorf("unexpected source: %s", location)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	}})
	require.NoError(t, err)
	defer doc.Release()
	model, err := doc.BuildV3Model()
	require.NoError(t, err)
	item := model.Model.Components.Schemas.GetOrZero("Item").Schema()
	require.NotNil(t, item)
	id := item.Properties.GetOrZero("id").Schema()
	require.NotNil(t, id)
	require.Equal(t, []string{"integer"}, id.Type)
	found, _ := model.Index.SearchIndexForReference("./schemas/item.yaml#/components/schemas/Item")
	require.NotNil(t, found, "raw relative lookups must use the same remote identity as extraction")
	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, map[string]int{"https://api.example/schemas/item.yaml": 1, "http://types.example/id.yaml": 1}, calls)
}

func TestRemoteOriginsWithSamePathStayDistinct(t *testing.T) {
	spec := []byte(`openapi: 3.1.0
info:
  title: Distinct remote origins
  version: 1.0.0
paths: {}
components:
  schemas:
    Left:
      $ref: 'https://left.example/type.yaml'
    Right:
      $ref: 'https://right.example/type.yaml'
`)
	base, _ := url.Parse("https://root.example/")
	doc, err := NewDocumentWithConfiguration(spec, &datamodel.DocumentConfiguration{BaseURL: base, AllowRemoteReferences: true, RemoteURLHandler: func(location string) (*http.Response, error) {
		kind := ""
		switch location {
		case "https://left.example/type.yaml":
			kind = "integer"
		case "https://right.example/type.yaml":
			kind = "string"
		default:
			return nil, fmt.Errorf("unexpected origin: %s", location)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("type: " + kind))}, nil
	}})
	require.NoError(t, err)
	defer doc.Release()
	model, err := doc.BuildV3Model()
	require.NoError(t, err)
	require.Equal(t, []string{"integer"}, model.Model.Components.Schemas.GetOrZero("Left").Schema().Type)
	require.Equal(t, []string{"string"}, model.Model.Components.Schemas.GetOrZero("Right").Schema().Type)
}

func TestRemoteOriginCycleBuildsOncePerFile(t *testing.T) {
	spec := []byte(`openapi: 3.1.0
info:
  title: Remote origin cycle
  version: 1.0.0
paths: {}
components:
  schemas:
    Item:
      $ref: 'https://a.example/type.yaml'
`)
	var mu sync.Mutex
	calls := map[string]int{}
	doc, err := NewDocumentWithConfiguration(spec, &datamodel.DocumentConfiguration{AllowRemoteReferences: true, RemoteURLHandler: func(location string) (*http.Response, error) {
		mu.Lock()
		calls[location]++
		mu.Unlock()
		next := ""
		switch location {
		case "https://a.example/type.yaml":
			next = "https://b.example/type.yaml"
		case "https://b.example/type.yaml":
			next = "https://a.example/type.yaml"
		default:
			return nil, fmt.Errorf("unexpected source: %s", location)
		}
		body := fmt.Sprintf("type: object\nproperties:\n  child:\n    $ref: '%s'\n", next)
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	}})
	require.NoError(t, err)
	defer doc.Release()
	model, err := doc.BuildV3Model()
	require.NoError(t, err)
	require.NotNil(t, model.Model.Components.Schemas.GetOrZero("Item").Schema())
	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, map[string]int{"https://a.example/type.yaml": 1, "https://b.example/type.yaml": 1}, calls)
}
