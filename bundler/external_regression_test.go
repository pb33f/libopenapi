package bundler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/utils"
	"github.com/pb33f/testify/require"
)

func externalRegressionConfig(t *testing.T, files map[string]string) *datamodel.DocumentConfiguration {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	}
	return &datamodel.DocumentConfiguration{BasePath: dir, AllowFileReferences: true}
}

func TestBundleBytesComposed_EncodedPathItem(t *testing.T) {
	config := externalRegressionConfig(t, map[string]string{"paths.yaml": `openapi: 3.1.0
info: {title: Paths, version: "1"}
paths:
  /items/{id}:
    get:
      parameters: [{name: id, in: path, required: true, schema: {type: string}}]
      responses:
        '200': {description: ok}
`})
	root := []byte(`openapi: 3.1.0
info: {title: Encoded paths, version: "1"}
paths:
  /items/{id}: {$ref: './paths.yaml#/paths/~1items~1%7Bid%7D'}
`)
	out, err := BundleBytesComposed(root, config, nil)
	require.NoError(t, err)
	var bundled yaml.Node
	require.NoError(t, yaml.Unmarshal(out, &bundled))
	_, paths := utils.FindKeyNodeTop("paths", bundled.Content[0].Content)
	require.NotNil(t, paths)
	_, pathItem := utils.FindKeyNodeTop("/items/{id}", paths.Content)
	require.NotNil(t, pathItem)
	require.Nil(t, utils.GetRefValueNode(pathItem), "path item must contain the operation, not a self-reference")
	doc, err := libopenapi.NewDocument(out)
	require.NoError(t, err)
	model, err := doc.BuildV3Model()
	require.NoError(t, err)
	item := model.Model.Paths.PathItems.GetOrZero("/items/{id}")
	require.NotNil(t, item.Get)
	require.Len(t, item.Get.Parameters, 1)
}

func TestBundleBytesComposed_SharedAliasTarget(t *testing.T) {
	config := externalRegressionConfig(t, map[string]string{
		"schemas/List.yaml":  "type: object\nproperties:\n  data:\n    type: array\n    items: {$ref: ./Item.yaml}\n",
		"schemas/Alias.yaml": "$ref: ./Item.yaml\n",
		"schemas/Item.yaml":  "type: object\nproperties:\n  id: {type: string}\n",
	})
	root := []byte(`openapi: 3.1.0
info: {title: Shared aliases, version: "1"}
paths:
  /items:
    get:
      responses:
        '200':
          description: ok
          content: {application/json: {schema: {$ref: ./schemas/List.yaml}}}
  /item:
    get:
      responses:
        '200':
          description: ok
          content: {application/json: {schema: {$ref: ./schemas/Alias.yaml}}}
`)
	out, err := BundleBytesComposed(root, config, nil)
	require.NoError(t, err)
	doc, err := libopenapi.NewDocument(out)
	require.NoError(t, err)
	model, err := doc.BuildV3Model()
	require.NoError(t, err)
	list := model.Model.Components.Schemas.GetOrZero("List").Schema()
	require.NotNil(t, list.Properties.GetOrZero("data"), string(out))
	require.Equal(t, 2, model.Model.Components.Schemas.Len(), string(out))
	item := list.Properties.GetOrZero("data").Schema().Items.A.Schema()
	require.Equal(t, []string{"object"}, item.Type)
	require.NotNil(t, item.Properties.GetOrZero("id"))
}

func TestBundleBytes_RecursiveSchemaAtDifferentDepths(t *testing.T) {
	for depth := 0; depth <= 5; depth++ {
		t.Run(fmt.Sprint(depth), func(t *testing.T) {
			files := map[string]string{
				"schemas/Step.yaml": "properties:\n  steps:\n    items: {$ref: ./Step.yaml}\n",
				"schemas/V.yaml":    "properties:\n  steps:\n    items: {$ref: ./Step.yaml}\n  styles: {$ref: ./W.yaml}\n",
				"schemas/W.yaml":    "properties:\n  c: {$ref: ./E.yaml}\n",
				"schemas/E.yaml":    "{}\n",
			}
			ref := "V.yaml"
			for i := depth; i > 0; i-- {
				name := fmt.Sprintf("A%d.yaml", i)
				files["schemas/"+name] = "properties:\n  p: {$ref: ./" + ref + "}\n"
				ref = name
			}
			files["paths/item.yaml"] = "post:\n  requestBody:\n    content:\n      application/json:\n        schema: {$ref: ../schemas/" + ref + "}\n  responses:\n    '200': {description: ok}\n"
			config := externalRegressionConfig(t, files)
			root := []byte("openapi: 3.0.3\ninfo: {title: Nested recursion, version: '1'}\npaths:\n  /items: {$ref: ./paths/item.yaml}\n")
			out, err := BundleBytes(root, config)
			require.NoError(t, err)
			doc, err := libopenapi.NewDocument(out)
			require.NoError(t, err)
			model, err := doc.BuildV3Model()
			require.NoError(t, err)
			step := model.Model.Components.Schemas.GetOrZero("Step")
			require.NotNil(t, step, string(out))
			require.Equal(t, "#/components/schemas/Step", step.Schema().Properties.GetOrZero("steps").Schema().Items.A.GetReference())
		})
	}
}

func TestBundleBytesComposed_AliasDiscriminatorMapping(t *testing.T) {
	cfg := externalRegressionConfig(t, map[string]string{"schemas/Alias.yaml": "$ref: ./Dog.yaml\n", "schemas/Dog.yaml": "type: object\nproperties:\n  kind: {type: string, enum: [dog]}\n"})
	out, err := BundleBytesComposed([]byte(`openapi: 3.1.0
info: {title: Aliases, version: '1'}
paths: {}
components:
  schemas:
    Animal:
      oneOf:
        - {$ref: ./schemas/Alias.yaml}
      discriminator:
        propertyName: kind
        mapping:
          dog: ./schemas/Alias.yaml
`), cfg, nil)
	require.NoError(t, err)

	doc, err := libopenapi.NewDocument(out)
	require.NoError(t, err)
	model, err := doc.BuildV3Model()
	require.NoError(t, err)
	mapping := model.Model.Components.Schemas.GetOrZero("Animal").Schema().Discriminator.Mapping.GetOrZero("dog")
	require.True(t, strings.HasPrefix(mapping, "#/components/schemas/"), mapping)
	require.NotNil(t, model.Model.Components.Schemas.GetOrZero(strings.TrimPrefix(mapping, "#/components/schemas/")), mapping)
}
