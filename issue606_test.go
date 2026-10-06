// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package libopenapi

import (
	"testing"

	"github.com/pb33f/testify/require"
)

func TestIssue606MalformedOperationPreservesValidSiblings(t *testing.T) {
	spec := `openapi: 3.2.0
info:
  title: Operation recovery
  version: 1.0.0
paths:
  /items:
    get:
      parameters:
    post:
      responses:
        '201':
          description: created
    additionalOperations:
      COPY:
        parameters:
      MOVE:
        responses:
          '200':
            description: moved
`
	doc, err := NewDocument([]byte(spec))
	require.NoError(t, err)
	model, errs := doc.BuildV3Model()
	// Paths retains partially valid path items without exposing operation errors.
	require.Empty(t, errs)
	require.NotNil(t, model)
	path, ok := model.Model.Paths.PathItems.Get("/items")
	require.True(t, ok)
	require.NotNil(t, path.Post)
	require.NotNil(t, path.Post.Responses)
	response, ok := path.Post.Responses.Codes.Get("201")
	require.True(t, ok)
	require.Equal(t, "created", response.Description)
	require.NotNil(t, path.AdditionalOperations)
	move, ok := path.AdditionalOperations.Get("MOVE")
	require.True(t, ok)
	require.NotNil(t, move.Responses)
	response, ok = move.Responses.Codes.Get("200")
	require.True(t, ok)
	require.Equal(t, "moved", response.Description)
}
