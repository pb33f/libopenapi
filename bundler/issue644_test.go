// Copyright 2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package bundler

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

// issue644Bundle writes files into a temporary directory, bundles root with BundleDocumentComposed and
// returns the bundled document along with everything that was logged.
func issue644Bundle(t *testing.T, root string, files map[string]string, sequential bool) (*yaml.Node, string) {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}

	var logs bytes.Buffer
	config := datamodel.NewDocumentConfiguration()
	config.BasePath = dir
	config.ExtractRefsSequentially = sequential
	config.Logger = slog.New(slog.NewTextHandler(&logs, nil))

	doc, err := libopenapi.NewDocumentWithConfiguration([]byte(root), config)
	require.NoError(t, err)
	v3Doc, err := doc.BuildV3Model()
	require.NoError(t, err)

	bundled, err := BundleDocumentComposed(&v3Doc.Model, nil)
	require.NoError(t, err)

	var node yaml.Node
	require.NoError(t, yaml.Unmarshal(bundled, &node))
	return node.Content[0], logs.String()
}

// issue644Keys returns the keys of a mapping node in order.
func issue644Keys(node *yaml.Node) []string {
	var keys []string
	for i := 0; i < len(node.Content); i += 2 {
		keys = append(keys, node.Content[i].Value)
	}
	return keys
}

const issue644Root = `openapi: 3.1.0
info:
  title: repro
  version: 1.0.0
paths:
  /a:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "./schemas/Wrapper.yaml"
  /b:
    get:
      responses:
        "400":
          description: bad
          content:
            application/json:
              schema:
                $ref: "./schemas/Extended.yaml"
`

// https://github.com/pb33f/libopenapi/issues/644
// A schema file referenced from a property and from an item of a top-level allOf (anyOf, oneOf, prefixItems)
// was composed twice: once as Base and once as Base__schemas, with every $ref pointing at the copy and the
// schema holding the property missing from the bundle.
func TestBundleDocumentComposed_Issue644_SchemaReferencedFromSequenceItemAndProperty(t *testing.T) {
	extended := map[string]string{
		"allOf":       "allOf:\n  - $ref: \"./Base.yaml\"\n  - type: object\n    properties:\n      errors:\n        type: array\n        items:\n          type: string\n",
		"anyOf":       "anyOf:\n  - $ref: \"./Base.yaml\"\n  - type: string\n",
		"oneOf":       "oneOf:\n  - type: string\n  - $ref: \"./Base.yaml\"\n",
		"prefixItems": "type: array\nprefixItems:\n  - $ref: \"./Base.yaml\"\n",
	}
	itemIndex := map[string]int{"allOf": 0, "anyOf": 0, "oneOf": 1, "prefixItems": 0}

	for _, keyword := range []string{"allOf", "anyOf", "oneOf", "prefixItems"} {
		for _, sequential := range []bool{true, false} {
			name := keyword + "/concurrent"
			if sequential {
				name = keyword + "/sequential"
			}
			t.Run(name, func(t *testing.T) {
				root, logs := issue644Bundle(t, issue644Root, map[string]string{
					"schemas/Base.yaml":     "type: object\nproperties:\n  title:\n    type: string\n",
					"schemas/Wrapper.yaml":  "type: object\nproperties:\n  error:\n    $ref: \"./Base.yaml\"\n",
					"schemas/Extended.yaml": extended[keyword],
				}, sequential)

				assert.NotContains(t, logs, "unable to locate reference")

				schemas := issue607Value(t, root, "components", "schemas")
				assert.ElementsMatch(t, []string{"Wrapper", "Extended", "Base"}, issue644Keys(schemas))

				assert.Equal(t, "#/components/schemas/Base",
					issue607Value(t, schemas, "Wrapper", "properties", "error", "$ref").Value)

				items := issue607Value(t, schemas, "Extended", keyword)
				require.Equal(t, yaml.SequenceNode, items.Kind)
				assert.Equal(t, "#/components/schemas/Base",
					issue607Value(t, items.Content[itemIndex[keyword]], "$ref").Value)
			})
		}
	}
}

// The same double composition hit parameters: a parameter file referenced from an operation's parameter list
// and from a components/parameters entry in another file was composed twice, as Limit and Limit__params.
func TestBundleDocumentComposed_Issue644_ParameterReferencedFromListAndComponents(t *testing.T) {
	root, logs := issue644Bundle(t, `openapi: 3.1.0
info:
  title: repro
  version: 1.0.0
paths:
  /a:
    get:
      parameters:
        - $ref: "./params/Limit.yaml"
      responses:
        "200":
          description: ok
  /b:
    get:
      parameters:
        - $ref: "./lib.yaml#/components/parameters/PageLimit"
      responses:
        "200":
          description: ok
`, map[string]string{
		"params/Limit.yaml": "name: limit\nin: query\nschema:\n  type: integer\n",
		"lib.yaml":          "components:\n  parameters:\n    PageLimit:\n      $ref: \"./params/Limit.yaml\"\n",
	}, true)

	assert.NotContains(t, logs, "unable to locate reference")

	parameters := issue607Value(t, root, "components", "parameters")
	assert.Equal(t, []string{"Limit", "PageLimit"}, issue644Keys(parameters))
	assert.Equal(t, "limit", issue607Value(t, parameters, "Limit", "name").Value)
	assert.Equal(t, "#/components/parameters/Limit", issue607Value(t, parameters, "PageLimit", "$ref").Value)

	listRef := func(path string) string {
		list := issue607Value(t, root, "paths", path, "get", "parameters")
		require.Len(t, list.Content, 1)
		return issue607Value(t, list.Content[0], "$ref").Value
	}
	assert.Equal(t, "#/components/parameters/Limit", listRef("/a"))
	assert.Equal(t, "#/components/parameters/PageLimit", listRef("/b"))
}

// The issue-928 fixture references Code2Map, Code3Map and CodeUp4Map from a oneOf inside CodeXMap.yaml, and
// from mapping slots elsewhere. Composed bundling lifted each of them twice, as X and X__CodeXMap.
func TestBundleBytesComposed_Issue644_OneOfReferencesComposeOnce(t *testing.T) {
	fixtureDir := filepath.Join("test", "specs", "issue-928")
	spec, err := os.ReadFile(filepath.Join(fixtureDir, "api.yaml"))
	require.NoError(t, err)

	config := &datamodel.DocumentConfiguration{
		BasePath:            fixtureDir,
		AllowFileReferences: true,
	}
	bundled, err := BundleBytesComposed(spec, config, nil)
	require.NoError(t, err)
	assert.NotContains(t, string(bundled), "__CodeXMap")

	var node yaml.Node
	require.NoError(t, yaml.Unmarshal(bundled, &node))
	schemas := issue607Value(t, node.Content[0], "components", "schemas")
	assert.Equal(t, []string{"BugDto", "CodeStringOrMapDto", "Code2Map", "Code3Map", "CodeUp4Map"}, issue644Keys(schemas))
}
