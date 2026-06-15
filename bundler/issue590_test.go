// Copyright 2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package bundler

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

type issue590LogCapture struct {
	mu      sync.Mutex
	records []string
}

func (h *issue590LogCapture) Enabled(_ context.Context, level slog.Level) bool {
	return level >= slog.LevelError
}

func (h *issue590LogCapture) Handle(_ context.Context, record slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	var b strings.Builder
	b.WriteString(record.Message)
	record.Attrs(func(attr slog.Attr) bool {
		b.WriteByte(' ')
		b.WriteString(attr.Key)
		b.WriteByte('=')
		b.WriteString(attr.Value.String())
		return true
	})
	h.records = append(h.records, b.String())
	return nil
}

func (h *issue590LogCapture) WithAttrs(_ []slog.Attr) slog.Handler {
	return h
}

func (h *issue590LogCapture) WithGroup(_ string) slog.Handler {
	return h
}

func (h *issue590LogCapture) joined() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return strings.Join(h.records, "\n")
}

func TestBundleDocument_Issue590_NoFalseRolodexErrors(t *testing.T) {
	tmp := t.TempDir()
	specDir := filepath.Join(tmp, "oapispec")
	require.NoError(t, os.MkdirAll(filepath.Join(specDir, "ops", "somefolder"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(specDir, "parameters"), 0755))

	openapi := `openapi: 3.1.0
info:
  title: Common schemas
  version: 0.0.1
servers:
  - url: http://localhost:8080/
paths:
  /somePath:
    $ref: './ops/somefolder/someOp.yaml'
components:
  parameters:
    foo:
      $ref: "./parameters/list.yaml#/foo"
    bar:
      in: query
      name: bar
      schema:
        type: integer
      required: false
`
	operation := `get:
  parameters:
    - $ref: '../../parameters/list.yaml#/foo'
  responses:
    "200":
      content:
        application/json:
          schema:
            type: string
`
	parameters := `foo:
  name: SomeName
  in: path
  required: true
  schema:
    type: string
`

	require.NoError(t, os.WriteFile(filepath.Join(specDir, "openapi.yaml"), []byte(openapi), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(specDir, "ops", "somefolder", "someOp.yaml"), []byte(operation), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(specDir, "parameters", "list.yaml"), []byte(parameters), 0644))

	logs := &issue590LogCapture{}
	config := datamodel.NewDocumentConfiguration()
	config.AllowFileReferences = true
	config.ExtractRefsSequentially = true
	config.BasePath = specDir
	config.Logger = slog.New(logs)

	specBytes, err := os.ReadFile(filepath.Join(specDir, "openapi.yaml"))
	require.NoError(t, err)

	doc, err := libopenapi.NewDocumentWithConfiguration(specBytes, config)
	require.NoError(t, err)

	v3Model, errs := doc.BuildV3Model()
	require.Empty(t, errs)

	bundled, err := BundleDocument(&v3Model.Model)
	require.NoError(t, err)

	output := string(bundled)
	assert.Contains(t, output, "name: SomeName")
	assert.Equal(t, 2, strings.Count(output, "name: SomeName"))
	assert.NotContains(t, output, "parameters/list.yaml")
	assert.NotContains(t, output, "$ref")

	logOutput := logs.joined()
	assert.NotContains(t, logOutput, "unable to open the rolodex file")
	assert.NotContains(t, logOutput, filepath.Join(tmp, "parameters", "list.yaml"))
	assert.NotContains(t, logOutput, filepath.Join(specDir, "parameters", "parameters", "list.yaml"))
}
