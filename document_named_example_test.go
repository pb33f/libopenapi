package libopenapi

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/testify/require"
)

func TestBuildV3Model_NamedExampleSchemaCycles(t *testing.T) {
	for _, name := range []string{"example", "examples"} {
		for _, container := range []string{"components/schemas", "definitions", "$defs", "dependentSchemas", "external"} {
			t.Run(container+"/"+name, func(t *testing.T) {
				prefix := "#/" + container + "/"
				if container == "external" {
					prefix = "#/"
				}
				schemas := fmt.Sprintf(`%s:
  type: object
  required: [child]
  properties:
    child: {$ref: '%sNode'}
Node:
  type: object
  required: [back]
  properties:
    back: {$ref: '%s%s'}
`, name, prefix, prefix, name)
				header := "openapi: 3.1.1\ninfo: {title: Named schema cycles, version: '1'}\npaths: {}\n"
				config := datamodel.NewDocumentConfiguration()
				var source string
				switch container {
				case "external":
					config = externalRegressionConfig(t, map[string]string{"schemas.yaml": schemas})
					source = header + "components:\n  schemas:\n    Root: {$ref: './schemas.yaml#/" + name + "'}\n"
				case "components/schemas":
					source = header + "components:\n  schemas:\n" + indentNamedSchema(schemas, 4)
				default:
					source = header + "components:\n  schemas:\n    Root: {$ref: '" + prefix + name + "'}\n" + container + ":\n" + indentNamedSchema(schemas, 2)
				}
				doc, err := NewDocumentWithConfiguration([]byte(source), config)
				require.NoError(t, err)
				_, err = doc.BuildV3Model()
				require.ErrorContains(t, err, "infinite circular reference detected")
			})
		}
	}
}

func indentNamedSchema(source string, spaces int) string {
	indent := strings.Repeat(" ", spaces)
	return indent + strings.ReplaceAll(strings.TrimSuffix(source, "\n"), "\n", "\n"+indent) + "\n"
}

func TestBuildV3Model_ExternalNamedExampleObject(t *testing.T) {
	config := externalRegressionConfig(t, map[string]string{
		"examples.yaml": "example:\n  value:\n    $ref: '#/payload-only'\n",
	})
	doc, err := NewDocumentWithConfiguration([]byte(`openapi: 3.1.1
info: {title: External example, version: '1'}
paths: {}
components:
  examples:
    External: {$ref: './examples.yaml#/example'}
`), config)
	require.NoError(t, err)
	model, err := doc.BuildV3Model()
	require.NoError(t, err)
	require.NotNil(t, model)
	value := model.Model.Components.Examples.GetOrZero("External").Value
	require.Equal(t, "$ref", value.Content[0].Value)
	require.Equal(t, "#/payload-only", value.Content[1].Value)
}
