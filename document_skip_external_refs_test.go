// Copyright 2026 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package libopenapi

import (
	"io"
	"log/slog"
	"testing"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

// With SkipExternalRefResolution enabled, an external $ref used as a map entry (a media type, header,
// link, example, encoding, callback or webhook) is replaced by a placeholder that is never built, so its
// node store is nil. The parent's Build then recorded the entry's key line on that nil store and
// panicked on a goroutine started by TranslateSliceParallel, which callers cannot recover.
func TestSkipExternalRefResolution_ExternalRefMapEntriesDoNotPanic(t *testing.T) {
	const h30 = "openapi: 3.0.3\ninfo: {title: t, version: '1'}\n"
	cases := map[string]string{
		"request body content": h30 + `paths:
  /x:
    post:
      requestBody:
        content:
          application/json: {$ref: 'other.yaml#/mt'}
      responses: {}
`,
		"response content": h30 + `paths:
  /x:
    get:
      responses:
        '200':
          description: ok
          content:
            application/json: {$ref: 'other.yaml#/mt'}
`,
		"response headers": h30 + `paths:
  /x:
    get:
      responses:
        '200':
          description: ok
          headers:
            X-A: {$ref: 'other.yaml#/h'}
`,
		"response links": h30 + `paths:
  /x:
    get:
      responses:
        '200':
          description: ok
          links:
            l: {$ref: 'other.yaml#/l'}
`,
		"parameter content": h30 + `paths:
  /x:
    get:
      parameters:
        - name: a
          in: query
          content:
            application/json: {$ref: 'other.yaml#/mt'}
      responses: {}
`,
		"parameter examples": h30 + `paths:
  /x:
    get:
      parameters:
        - name: a
          in: query
          schema: {type: string}
          examples:
            e: {$ref: 'other.yaml#/e'}
      responses: {}
`,
		"media type examples": h30 + `paths:
  /x:
    post:
      requestBody:
        content:
          application/json:
            examples:
              e: {$ref: 'other.yaml#/e'}
      responses: {}
`,
		"media type encoding": h30 + `paths:
  /x:
    post:
      requestBody:
        content:
          multipart/form-data:
            encoding:
              f: {$ref: 'other.yaml#/enc'}
      responses: {}
`,
		"encoding headers": h30 + `paths:
  /x:
    post:
      requestBody:
        content:
          multipart/form-data:
            encoding:
              f:
                headers:
                  X-A: {$ref: 'other.yaml#/h'}
      responses: {}
`,
		"operation callbacks": h30 + `paths:
  /x:
    get:
      callbacks:
        cb: {$ref: 'other.yaml#/cb'}
      responses: {}
`,
		"webhooks": "openapi: 3.1.0\ninfo: {title: t, version: '1'}\npaths: {}\n" + `webhooks:
  w: {$ref: 'other.yaml#/pi'}
`,
		"media type item encoding": "openapi: 3.2.0\ninfo: {title: t, version: '1'}\n" + `paths:
  /x:
    post:
      requestBody:
        content:
          multipart/mixed:
            itemEncoding:
              f: {$ref: 'other.yaml#/enc'}
      responses: {}
`,
	}
	for name, spec := range cases {
		t.Run(name, func(t *testing.T) {
			cfg := datamodel.NewDocumentConfiguration()
			cfg.SkipExternalRefResolution = true
			cfg.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))

			doc, err := NewDocumentWithConfiguration([]byte(spec), cfg)
			require.NoError(t, err)
			model, err := doc.BuildV3Model()
			assert.NoError(t, err)
			assert.NotNil(t, model)
		})
	}
}
