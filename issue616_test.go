// Copyright 2026 Princess Beef Heavy Industries / Dave Shanley
// https://pb33f.io
// SPDX-License-Identifier: MIT
package libopenapi

import (
	"testing"

	"github.com/pb33f/testify/require"
)

const issue616Spec = `openapi: 3.0.2
info:
  title: t
  version: 1.0.0
paths:
  /p:
    get:
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Nope'
    post:
      responses:
        '200':
          description: ok
`

// A dangling $ref in one operation must not leave a sibling operation with a nil
// embedded *low.Reference that panics when IsReference is called on the model.
func TestIssue616SiblingOperationIsReferenceDoesNotPanic(t *testing.T) {
	doc, err := NewDocument([]byte(issue616Spec))
	require.NoError(t, err)

	model, _ := doc.BuildV3Model()
	require.NotNil(t, model)

	pi, ok := model.Model.Paths.PathItems.Get("/p")
	require.True(t, ok)
	require.NotNil(t, pi.Post)

	require.NotPanics(t, func() {
		_ = pi.Post.GoLow().IsReference()
	})
	require.False(t, pi.Post.GoLow().IsReference())
}
