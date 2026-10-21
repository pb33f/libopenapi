// Copyright 2026 Princess Beef Heavy Industries / Dave Shanley
// https://pb33f.io
// SPDX-License-Identifier: MIT
package libopenapi

import (
	"math"
	"testing"

	"github.com/pb33f/testify/require"
)

const issue665Spec = `openapi: 3.1.0
info:
  title: issue 665
  version: 1.0.0
paths: {}
components:
  schemas:
    FloatWhole:
      type: string
      maxLength: 2.0
    HugeMaxItems:
      type: array
      maxItems: 1.7976931348623157E+308
    Integer:
      type: string
      maxLength: 5
    Fractional:
      type: string
      maxLength: 2.5
`

func TestIssue665FloatTaggedInt64Keywords(t *testing.T) {
	doc, err := NewDocument([]byte(issue665Spec))
	require.NoError(t, err)

	model, err := doc.BuildV3Model()
	require.NoError(t, err)
	require.NotNil(t, model)

	floatWhole := model.Model.Components.Schemas.GetOrZero("FloatWhole").Schema()
	require.NotNil(t, floatWhole)
	require.NotNil(t, floatWhole.MaxLength)
	require.Equal(t, int64(2), *floatWhole.MaxLength)

	hugeMaxItems := model.Model.Components.Schemas.GetOrZero("HugeMaxItems").Schema()
	require.NotNil(t, hugeMaxItems)
	require.NotNil(t, hugeMaxItems.MaxItems)
	require.Equal(t, int64(math.MaxInt64), *hugeMaxItems.MaxItems)

	integer := model.Model.Components.Schemas.GetOrZero("Integer").Schema()
	require.NotNil(t, integer)
	require.NotNil(t, integer.MaxLength)
	require.Equal(t, int64(5), *integer.MaxLength)

	fractional := model.Model.Components.Schemas.GetOrZero("Fractional").Schema()
	require.NotNil(t, fractional)
	require.Nil(t, fractional.MaxLength)
}
