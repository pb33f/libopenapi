// Copyright 2026 Princess Beef Heavy Industries / Dave Shanley
// https://pb33f.io
// SPDX-License-Identifier: MIT
package libopenapi

import (
	"testing"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

const issue597Spec = `openapi: 3.1.0
info:
  title: Common schemas
  version: 0.0.1
servers:
  - url: http://localhost:8080/
components:
  schemas:
    foo:
      type: object
      properties:
        hello:
          type: string
        world:
          type: string
      x-custom: true
`

func TestIssue597SchemaExtensionMarshalYAMLDoesNotChangeLowHash(t *testing.T) {
	tests := []struct {
		name    string
		marshal func(*orderedmap.Map[string, *yaml.Node]) error
	}{
		{
			name: "direct MarshalYAML",
			marshal: func(extensions *orderedmap.Map[string, *yaml.Node]) error {
				_, err := extensions.MarshalYAML()
				return err
			},
		},
		{
			name: "yaml Marshal",
			marshal: func(extensions *orderedmap.Map[string, *yaml.Node]) error {
				_, err := yaml.Marshal(extensions)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := NewDocumentWithConfiguration([]byte(issue597Spec), &datamodel.DocumentConfiguration{
				ExtractRefsSequentially: true,
			})
			require.NoError(t, err)

			v3Model, err := doc.BuildV3Model()
			require.NoError(t, err)

			schema := v3Model.Model.Components.Schemas.Value("foo").Schema()
			lowSchema := schema.GoLow()
			lowExtension := low.FindItemInOrderedMap[*yaml.Node]("x-custom", lowSchema.GetExtensions())
			require.NotNil(t, lowExtension)
			require.Equal(t, "!!bool", lowExtension.Value.Tag)

			initialHash := lowSchema.Hash()
			low.ClearHashCache()

			require.NoError(t, tt.marshal(schema.Extensions))
			require.Equal(t, "!!bool", lowExtension.Value.Tag)
			require.Equal(t, "true", lowExtension.Value.Value)

			low.ClearHashCache()
			require.Equal(t, initialHash, lowSchema.Hash())
		})
	}
}
