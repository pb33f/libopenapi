// Copyright 2026 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package libopenapi

import (
	"fmt"
	"github.com/pb33f/testify/require"
	"testing"
)

func TestBuildV3Model_ExternalSchemaAliasChain(t *testing.T) {
	for _, aliases := range []int{0, 1, 2, 5} {
		t.Run(fmt.Sprint(aliases), func(t *testing.T) {
			files := map[string]string{"schemas/target.yaml": "type: object\nproperties:\n  id: {type: string}\n"}
			ref := "./target.yaml"
			for i := aliases; i > 0; i-- {
				name := fmt.Sprintf("alias%d.yaml", i)
				files["schemas/"+name] = "$ref: " + ref + "\n"
				ref = "./" + name
			}
			config := externalRegressionConfig(t, files)
			doc, err := NewDocumentWithConfiguration([]byte(fmt.Sprintf(`openapi: 3.1.0
info: {title: Alias chains, version: "1"}
paths: {}
components:
  schemas:
    Item: {$ref: ./schemas/%s}
`, ref[2:])), config)
			require.NoError(t, err)
			model, err := doc.BuildV3Model()
			require.NoError(t, err)
			proxy := model.Model.Components.Schemas.GetOrZero("Item")
			schema := proxy.Schema()
			require.NoError(t, proxy.GetBuildError())
			require.NotNil(t, schema)
			require.Equal(t, []string{"object"}, schema.Type)
			require.NotNil(t, schema.Properties.GetOrZero("id"))
		})
	}
}
