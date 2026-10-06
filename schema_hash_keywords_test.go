// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package libopenapi

import (
	"testing"

	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

const schemaHashKeywordsSpec = `openapi: 3.1.0
info:
  title: schema hash keywords
  version: 1.0.0
paths:
  /min:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: integer
              minimum: 5
  /max:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: integer
              maximum: 5
  /one-of:
    post:
      requestBody:
        content:
          application/json:
            schema:
              oneOf:
                - $ref: '#/components/schemas/Widget'
                - type: object
  /all-of:
    post:
      requestBody:
        content:
          application/json:
            schema:
              allOf:
                - $ref: '#/components/schemas/Widget'
                - type: object
  /widget:
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Widget'
  /another-widget:
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Widget'
  /inline-widget:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                id:
                  type: string
components:
  schemas:
    Widget:
      type: object
      properties:
        id:
          type: string
`

// libopenapi-validator caches every compiled schema under schema.GoLow().Hash(). While the hash ignored which
// keyword held a value, /max was validated with the schema compiled for /min, and an allOf body with the schema
// compiled for a oneOf over the same members. The same schema, reached directly or through a $ref, must still share
// one hash so the cache keeps working.
func TestSchemaHashDistinguishesKeywordsAcrossOperations(t *testing.T) {
	doc, err := NewDocument([]byte(schemaHashKeywordsSpec))
	require.NoError(t, err)
	model, errs := doc.BuildV3Model()
	require.Nil(t, errs)

	bodySchemaHash := func(path string) uint64 {
		item, ok := model.Model.Paths.PathItems.Get(path)
		require.Truef(t, ok, "path %s missing", path)
		mediaType, ok := item.Post.RequestBody.Content.Get("application/json")
		require.Truef(t, ok, "path %s has no application/json body", path)
		return mediaType.Schema.Schema().GoLow().Hash()
	}

	assert.NotEqual(t, bodySchemaHash("/min"), bodySchemaHash("/max"))
	assert.NotEqual(t, bodySchemaHash("/one-of"), bodySchemaHash("/all-of"))

	assert.Equal(t, bodySchemaHash("/widget"), bodySchemaHash("/another-widget"))
	assert.Equal(t, bodySchemaHash("/widget"), bodySchemaHash("/inline-widget"))
}
