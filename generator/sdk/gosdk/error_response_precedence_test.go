// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package gosdk

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/generator/sdk"
)

func TestGeneratedErrorResponsesPreserveUntypedOverrides(t *testing.T) {
	document, err := libopenapi.NewDocument([]byte(errorResponsePrecedenceSpec))
	if err != nil {
		t.Fatal(err)
	}
	model, err := document.BuildV3Model()
	if err != nil {
		t.Fatal(err)
	}
	result, err := Generate(&model.Model, Options{PackageName: "example", Prepare: sdk.PrepareOptions{
		Names: map[string]sdk.OperationName{"getWidgets": {Resource: "Widgets", Method: "Get"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	for _, file := range result.Files {
		if err := os.WriteFile(filepath.Join(directory, file.Path), file.Content, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for path, content := range map[string]string{
		"go.mod":         "module generated.example/response-precedence\n\ngo 1.26.0\n",
		"client_test.go": errorResponsePrecedenceRuntime,
	} {
		if err := os.WriteFile(filepath.Join(directory, path), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	command := exec.Command("go", "test", "./...")
	command.Dir = directory
	command.Env = append(os.Environ(), "GOWORK=off")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("generated error handling failed: %v\n%s", err, output)
	}
}

const errorResponsePrecedenceSpec = `openapi: 3.1.0
info: {title: Widget service, version: '1'}
paths:
  /widgets:
    get:
      operationId: getWidgets
      responses:
        '200': {description: OK}
        '400':
          description: Invalid request
          content:
            application/json:
              schema: {type: string}
        '404':
          description: Missing widget
          content:
            text/plain:
              schema: {type: string}
        '405': {description: Method not allowed}
        '4XX':
          description: Other client error
          content:
            application/json:
              schema: {type: integer}
        '5XX':
          description: Server error
          content:
            text/plain:
              schema: {type: string}
        '503':
          description: Service unavailable
          content:
            application/json:
              schema: {type: boolean}
        default:
          description: Other error
          content:
            application/json:
              schema: {type: boolean}
`

const errorResponsePrecedenceRuntime = `package example

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type responseDoer struct { status int; body string }

func (d responseDoer) Do(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: d.status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(d.body))}, nil
}

func TestErrorResponsePrecedence(t *testing.T) {
	for _, test := range []struct { name string; status int; body string; typed bool }{
		{"typed exact over typed range", 400, "\"invalid\"", true},
		{"untyped exact over typed range", 404, "missing", false},
		{"bodyless exact over typed range", 405, "", false},
		{"typed range over default", 418, "42", true},
		{"untyped range over default", 502, "unavailable", false},
		{"typed exact over untyped range", 503, "true", true},
		{"typed default", 302, "false", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			client, err := NewClient("https://example.com", WithHTTPClient(responseDoer{test.status, test.body}))
			if err != nil { t.Fatal(err) }
			_, err = client.Widgets.Get(context.Background())
			var apiError *APIError
			if !errors.As(err, &apiError) { t.Fatalf("error = %v", err) }
			if apiError.StatusCode != test.status || string(apiError.Body) != test.body || apiError.Cause != nil {
				t.Fatalf("response evidence = %#v", apiError)
			}
			if (apiError.Value != nil) != test.typed { t.Fatalf("decoded value = %#v", apiError.Value) }
		})
	}
}
`
